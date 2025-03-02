package echos

import (
	"encoding/json"
	"sync"
	"time"

	"slices"

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
		for _, receiver := range r.peers[i].Connection.GetReceivers() {
			if receiver.Track() == nil {
				continue
			}

			_ = r.peers[i].Connection.WriteRTCP([]rtcp.Packet{
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
			if r.peers[i].Connection.ConnectionState() == webrtc.PeerConnectionStateClosed {
				r.peers = slices.Delete(r.peers, i, i+1)
				return true
			}

			existingSenders := map[string]bool{}

			for _, sender := range r.peers[i].Connection.GetSenders() {
				if sender.Track() == nil {
					continue
				}

				existingSenders[sender.Track().ID()] = true

				if _, ok := r.trackLocals[sender.Track().ID()]; !ok {
					if err := r.peers[i].Connection.RemoveTrack(sender); err != nil {
						return true
					}
				}
			}

			for _, receiver := range r.peers[i].Connection.GetReceivers() {
				if receiver.Track() == nil {
					continue
				}

				existingSenders[receiver.Track().ID()] = true
			}

			for trackID := range r.trackLocals {
				if _, ok := existingSenders[trackID]; !ok {
					if _, err := r.peers[i].Connection.AddTrack(r.trackLocals[trackID]); err != nil {
						return true
					}
				}
			}

			offer, err := r.peers[i].Connection.CreateOffer(nil)
			if err != nil {
				return true
			}

			if err = r.peers[i].Connection.SetLocalDescription(offer); err != nil {
				return true
			}

			offerString, err := json.Marshal(offer)
			if err != nil {
				log.Errorf("Failed to marshal offer to json: %v", err)
				return true
			}

			log.Infof("Send offer to client: %v", offer)

			if err = r.peers[i].websocket.WriteJSON(&websocketMessage{
				Event: "offer",
				Data:  string(offerString),
			}); err != nil {
				return true
			}
		}

		return
	}

	for syncAttempt := 0; ; syncAttempt++ {
		if syncAttempt == 25 {
			go func() {
				time.Sleep(time.Second * 3)
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

	if _, ok := Rooms[r.id]; len(r.peers) == 0 && ok {
		log.Infof("room delete: %s", r.id)
		delete(Rooms, r.id)
		return
	}
}
