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
	id              string
	listLock        sync.RWMutex
	peerConnections []peerConnectionState
	trackLocals     map[string]*webrtc.TrackLocalStaticRTP
}

func NewRoom(id string) *Room {
	room := Room{
		id:          id,
		listLock:    sync.RWMutex{},
		trackLocals: map[string]*webrtc.TrackLocalStaticRTP{},
	}

	go func() {
		for range time.NewTicker(time.Second * 3).C {
			room.DispatchKeyFrame()
		}
	}()

	return &room
}

func (r *Room) AddTrack(t *webrtc.TrackRemote) *webrtc.TrackLocalStaticRTP {
	r.listLock.Lock()
	defer func() {
		r.listLock.Unlock()
		r.SignalPeerConnections()
	}()

	trackLocal, err := webrtc.NewTrackLocalStaticRTP(t.Codec().RTPCodecCapability, t.ID(), t.StreamID())
	if err != nil {
		panic(err)
	}

	r.trackLocals[t.ID()] = trackLocal
	return trackLocal
}

func (r *Room) RemoveTrack(t *webrtc.TrackLocalStaticRTP) {
	r.listLock.Lock()
	defer func() {
		r.listLock.Unlock()
		r.SignalPeerConnections()
	}()

	delete(r.trackLocals, t.ID())
}

func (r *Room) DispatchKeyFrame() {
	r.listLock.Lock()
	defer r.listLock.Unlock()

	for i := range r.peerConnections {
		for _, receiver := range r.peerConnections[i].peerConnection.GetReceivers() {
			if receiver.Track() == nil {
				continue
			}

			_ = r.peerConnections[i].peerConnection.WriteRTCP([]rtcp.Packet{
				&rtcp.PictureLossIndication{
					MediaSSRC: uint32(receiver.Track().SSRC()),
				},
			})
		}
	}
}

func (r *Room) SignalPeerConnections() {
	r.listLock.Lock()
	defer func() {
		r.listLock.Unlock()
		r.DeleteSelfIfEmpty()
		r.DispatchKeyFrame()
	}()

	attemptSync := func() (tryAgain bool) {
		for i := range r.peerConnections {
			if r.peerConnections[i].peerConnection.ConnectionState() == webrtc.PeerConnectionStateClosed {
				r.peerConnections = slices.Delete(r.peerConnections, i, i+1)
				return true
			}

			existingSenders := map[string]bool{}

			for _, sender := range r.peerConnections[i].peerConnection.GetSenders() {
				if sender.Track() == nil {
					continue
				}

				existingSenders[sender.Track().ID()] = true

				if _, ok := r.trackLocals[sender.Track().ID()]; !ok {
					if err := r.peerConnections[i].peerConnection.RemoveTrack(sender); err != nil {
						return true
					}
				}
			}

			for _, receiver := range r.peerConnections[i].peerConnection.GetReceivers() {
				if receiver.Track() == nil {
					continue
				}

				existingSenders[receiver.Track().ID()] = true
			}

			for trackID := range r.trackLocals {
				if _, ok := existingSenders[trackID]; !ok {
					if _, err := r.peerConnections[i].peerConnection.AddTrack(r.trackLocals[trackID]); err != nil {
						return true
					}
				}
			}

			offer, err := r.peerConnections[i].peerConnection.CreateOffer(nil)
			if err != nil {
				return true
			}

			if err = r.peerConnections[i].peerConnection.SetLocalDescription(offer); err != nil {
				return true
			}

			offerString, err := json.Marshal(offer)
			if err != nil {
				log.Errorf("Failed to marshal offer to json: %v", err)
				return true
			}

			log.Infof("Send offer to client: %v", offer)

			if err = r.peerConnections[i].websocket.WriteJSON(&websocketMessage{
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
				r.SignalPeerConnections()
			}()
			return
		}

		if !attemptSync() {
			break
		}
	}
}

func (r *Room) DeleteSelfIfEmpty() {
	r.listLock.Lock()
	defer r.listLock.Unlock()

	if _, ok := Rooms[r.id]; len(r.peerConnections) == 0 && ok {
		log.Infof("room delete: %s", r.id)
		delete(Rooms, r.id)
		return
	}
}
