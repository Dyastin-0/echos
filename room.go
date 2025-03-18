package echos

import (
	"encoding/json"
	"slices"
	"sync"
	"time"

	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v4"
)

type Room struct {
	id          string
	listLock    sync.RWMutex
	peers       []*peer
	trackLocals map[string]*webrtc.TrackLocalStaticRTP
}

func NewRoom(id string) *Room {
	room := Room{
		id:          id,
		listLock:    sync.RWMutex{},
		trackLocals: map[string]*webrtc.TrackLocalStaticRTP{},
	}

	go func() {
		for range time.NewTicker(time.Second * 3).C {
			room.dispatchKeyFrame()
		}
	}()

	return &room
}

func (r *Room) addTrack(t *webrtc.TrackRemote) *webrtc.TrackLocalStaticRTP {
	r.listLock.Lock()
	defer func() {
		r.listLock.Unlock()
		r.signalPeerConnections()
	}()

	trackLocal, err := webrtc.NewTrackLocalStaticRTP(t.Codec().RTPCodecCapability, t.ID(), t.StreamID())
	if err != nil {
		panic(err)
	}

	r.trackLocals[t.ID()] = trackLocal
	return trackLocal
}

func (r *Room) removeTrack(t *webrtc.TrackLocalStaticRTP) {
	r.listLock.Lock()
	defer func() {
		r.listLock.Unlock()
		r.signalPeerConnections()
	}()

	delete(r.trackLocals, t.ID())
}

func (r *Room) dispatchKeyFrame() {
	r.listLock.Lock()
	defer r.listLock.Unlock()

	for i := range r.peers {
		for _, receiver := range r.peers[i].connection.GetReceivers() {
			if receiver.Track() == nil {
				continue
			}

			_ = r.peers[i].connection.WriteRTCP([]rtcp.Packet{
				&rtcp.PictureLossIndication{
					MediaSSRC: uint32(receiver.Track().SSRC()),
				},
			})
		}
	}
}

func (r *Room) signalPeerConnections() {
	r.listLock.Lock()
	defer func() {
		r.listLock.Unlock()
		r.deleteSelfIfEmpty()
		r.dispatchKeyFrame()
	}()

	attemptSync := func() (tryAgain bool) {
		for i := range r.peers {
			if r.peers[i].connection.ConnectionState() == webrtc.PeerConnectionStateClosed {
				r.peers = slices.Delete(r.peers, i, i+1)
				return true
			}

			existingSenders := map[string]bool{}

			for _, sender := range r.peers[i].connection.GetSenders() {
				if sender.Track() == nil {
					continue
				}

				existingSenders[sender.Track().ID()] = true

				if _, ok := r.trackLocals[sender.Track().ID()]; !ok {
					if err := r.peers[i].connection.RemoveTrack(sender); err != nil {
						return true
					}
				}
			}

			for _, receiver := range r.peers[i].connection.GetReceivers() {
				if receiver.Track() == nil {
					continue
				}

				existingSenders[receiver.Track().ID()] = true
			}

			for trackID := range r.trackLocals {
				if _, ok := existingSenders[trackID]; !ok {
					if _, err := r.peers[i].connection.AddTrack(r.trackLocals[trackID]); err != nil {
						return true
					}
				}
			}

			offer, err := r.peers[i].connection.CreateOffer(nil)
			if err != nil {
				return true
			}

			if err = r.peers[i].connection.SetLocalDescription(offer); err != nil {
				return true
			}

			offerBytes, err := json.Marshal(offer)
			if err != nil {
				log.Errorf("Failed to marshal offer to json: %v", err)
				return true
			}

			log.Infof("Send offer to client: %v", offer)

			if err = r.peers[i].socket.WriteJSON(&websocketMessage{
				Event: "offer",
				Data:  string(offerBytes),
			}); err != nil {
				return true
			}
		}

		return
	}

	for syncAttempt := 0; ; syncAttempt++ {
		if syncAttempt == 25 {
			go func() {
				time.Sleep(time.Second * 1)
				r.signalPeerConnections()
			}()
			return
		}

		if !attemptSync() {
			break
		}
	}
}

func (r *Room) deleteSelfIfEmpty() {
	r.listLock.Lock()
	defer r.listLock.Unlock()

	log.Errorf("len: %d", len(r.peers))

	roomsMutex.Lock()
	defer roomsMutex.Unlock()

	if _, ok := Rooms[r.id]; len(r.peers) == 0 && ok {
		log.Errorf("room delete: %s", r.id)
		delete(Rooms, r.id)
		return
	}
}

func (r *Room) wsListen(peer *peer) {
	for {
		message := &websocketMessage{}

		_, raw, err := peer.socket.ReadMessage()
		if err != nil {
			log.Errorf("Failed to read message: %v", err)

			r.propagateMessage(&websocketMessage{
				Event:  "message",
				Type:   "disconnect",
				Target: peer.id,
			})
			return
		}

		log.Infof("Got message: %s", raw)

		if err := json.Unmarshal(raw, &message); err != nil {
			log.Errorf("Failed to unmarshal json to message: %v", err)
			return
		}

		switch message.Event {
		case "candidate":
			candidate := webrtc.ICECandidateInit{}
			if err := json.Unmarshal([]byte(message.Data), &candidate); err != nil {
				log.Errorf("Failed to unmarshal json to candidate: %v", err)
				return
			}

			log.Infof("Got candidate: %v", candidate)

			if err := peer.connection.AddICECandidate(candidate); err != nil {
				log.Errorf("Failed to add ICE candidate: %v", err)
				return
			}
		case "answer":
			answer := webrtc.SessionDescription{}
			if err := json.Unmarshal([]byte(message.Data), &answer); err != nil {
				log.Errorf("Failed to unmarshal json to answer: %v", err)
				return
			}

			log.Infof("Got answer: %v", answer)

			if err := peer.connection.SetRemoteDescription(answer); err != nil {
				log.Errorf("Failed to set remote description: %v", err)
				return
			}
		case "renegotiate":
			var offer webrtc.SessionDescription
			if err := json.Unmarshal([]byte(message.Data), &offer); err != nil {
				log.Infof("Error unmarshaling SDP offer:", err)
				return
			}

			if err := peer.connection.SetRemoteDescription(offer); err != nil {
				log.Infof("Error setting remote description:", err)
				return
			}

			answer, err := peer.connection.CreateAnswer(nil)
			if err != nil {
				log.Infof("Error creating answer:", err)
				return
			}

			if err := peer.connection.SetLocalDescription(answer); err != nil {
				log.Infof("Error setting local description:", err)
				return
			}

			r.signalPeerConnections()
		case "message":
			message.ID = peer.id
			r.propagateMessage(message)
		default:
			log.Errorf("unknown message: %+v", message)
		}
	}
}

func (r *Room) propagateMessage(message *websocketMessage) {
	r.listLock.Lock()
	defer r.listLock.Unlock()

	for _, peer := range r.peers {
		if err := peer.socket.Conn.WriteJSON(message); err != nil {
			log.Errorf("failed to propagate message: %+v", err)
		}
	}
}
