package echos

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v4"
)

type Room struct {
	id          string
	peers       sync.Map
	trackLocals sync.Map
	deletech    chan bool
}

func NewRoom(id string, deletech chan bool) *Room {
	room := Room{
		id:       id,
		deletech: deletech,
	}

	go func() {
		ticker := time.NewTicker(time.Second * 3)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				room.dispatchKeyFrame()
			case <-room.deletech:
				return
			}
		}
	}()

	return &room
}

func (r *Room) addTrack(t *webrtc.TrackRemote) *webrtc.TrackLocalStaticRTP {
	defer r.signalPeerConnections()

	trackLocal, err := webrtc.NewTrackLocalStaticRTP(t.Codec().RTPCodecCapability, t.ID(), t.StreamID())
	if err != nil {
		panic(err)
	}

	r.trackLocals.Store(t.ID(), trackLocal)
	return trackLocal
}

func (r *Room) removeTrack(t *webrtc.TrackLocalStaticRTP) {
	defer r.signalPeerConnections()

	r.trackLocals.Delete(t.ID())

	// if there's no peer, signal delete room
	if r.isEmpty() {
		r.deletech <- true
	}
}

func (r *Room) isEmpty() bool {
	isEmpty := true
	r.peers.Range(func(key, value any) bool {
		isEmpty = false
		return false
	})

	return isEmpty
}

func (r *Room) dispatchKeyFrame() {
	r.peers.Range(func(id, p any) bool {
		for _, receiver := range p.(*peer).connection.GetReceivers() {
			if receiver.Track() == nil {
				continue
			}

			_ = p.(*peer).connection.WriteRTCP([]rtcp.Packet{
				&rtcp.PictureLossIndication{
					MediaSSRC: uint32(receiver.Track().SSRC()),
				},
			})
		}

		return true
	})
}

func (r *Room) signalPeerConnections() {
	defer r.dispatchKeyFrame()

	attemptSync := func() (tryAgain bool) {
		shouldAttempt := false

		r.peers.Range(func(id, p any) bool {
			if p.(*peer).connection.ConnectionState() == webrtc.PeerConnectionStateClosed {
				r.peers.Delete(id)
				shouldAttempt = true
				return false
			}

			existingSenders := map[string]bool{}

			for _, sender := range p.(*peer).connection.GetSenders() {
				if sender.Track() == nil {
					continue
				}

				existingSenders[sender.Track().ID()] = true

				if _, ok := r.trackLocals.Load(sender.Track().ID()); !ok {
					if err := p.(*peer).connection.RemoveTrack(sender); err != nil {
						shouldAttempt = true
						return false
					}
				}
			}

			for _, receiver := range p.(*peer).connection.GetReceivers() {
				if receiver.Track() == nil {
					continue
				}

				existingSenders[receiver.Track().ID()] = true
			}

			r.trackLocals.Range(func(key, t any) bool {
				if _, ok := existingSenders[key.(string)]; !ok {
					track, ok := r.trackLocals.Load(key)
					if !ok {
						shouldAttempt = true
						return false

					}

					if _, err := p.(*peer).connection.AddTrack(track.(webrtc.TrackLocal)); err != nil {
						shouldAttempt = true
						return false
					}

				}

				return true
			})

			if shouldAttempt {
				return false
			}

			offer, err := p.(*peer).connection.CreateOffer(nil)
			if err != nil {
				shouldAttempt = true
				return false

			}

			if err = p.(*peer).connection.SetLocalDescription(offer); err != nil {
				shouldAttempt = true
				return false

			}

			offerBytes, err := json.Marshal(offer)
			if err != nil {
				shouldAttempt = true
				return false
			}

			if err = p.(*peer).socket.WriteJSON(&websocketMessage{
				Event: "offer",
				Data:  string(offerBytes),
			}); err != nil {
				shouldAttempt = true
				return false
			}

			return true
		})

		select {
		case <-r.deletech:
			return false
		default:
		}

		return shouldAttempt
	}

	for syncAttempt := 0; ; syncAttempt++ {
		if syncAttempt == 25 {
			go func() {
				time.Sleep(200 * time.Millisecond)
				r.signalPeerConnections()
			}()
			return
		}

		if !attemptSync() {
			break
		}
	}
}

func (r *Room) wsListen(peer *peer) {
	for {
		message := &websocketMessage{}

		_, raw, err := peer.socket.ReadMessage()
		if err != nil {
			fmt.Printf("failed to read websocket message: %v\n", err)
			r.propagateMessage(&websocketMessage{
				Event:  "message",
				Type:   "disconnect",
				Target: peer.id,
			})
			return
		}

		if err := json.Unmarshal(raw, &message); err != nil {
			fmt.Printf("failed to unmarshal websocket message: %v\n", err)
			r.propagateMessage(&websocketMessage{
				Event:  "message",
				Type:   "disconnect",
				Target: peer.id,
			})
			return
		}

		switch message.Event {
		case "candidate":
			candidate := webrtc.ICECandidateInit{}
			if err := json.Unmarshal([]byte(message.Data), &candidate); err != nil {
				log.Printf("failed to unmarshal ice candidate: %v\n", err)
				return
			}

			if err := peer.connection.AddICECandidate(candidate); err != nil {
				log.Printf("failed to add ice candidate: %v\n", err)
				return
			}
		case "answer":
			answer := webrtc.SessionDescription{}
			if err := json.Unmarshal([]byte(message.Data), &answer); err != nil {
				log.Printf("failed to unmarshal answer: %v\n", err)
				return
			}

			if err := peer.connection.SetRemoteDescription(answer); err != nil {
				log.Printf("failed to set remote description: %v\n", err)
				return
			}
		case "renegotiate":
			var offer webrtc.SessionDescription
			if err := json.Unmarshal([]byte(message.Data), &offer); err != nil {
				log.Printf("failed to unmarshal offer: %v\n", err)
				return
			}

			if err := peer.connection.SetRemoteDescription(offer); err != nil {
				log.Printf("failed to set remote description: %v\n", err)
				return
			}

			answer, err := peer.connection.CreateAnswer(nil)
			if err != nil {
				log.Printf("failed to create answer: %v\n", err)
				return
			}

			if err := peer.connection.SetLocalDescription(answer); err != nil {
				log.Printf("failed to set local description: %v\n", err)
				return
			}

			r.signalPeerConnections()
		case "message":
			message.ID = peer.id
			message.Name = peer.name
			r.propagateMessage(message)
		default:
			log.Printf("unknown event: %s\n", message.Event)
		}
	}
}

func (r *Room) propagateMessage(message *websocketMessage) {
	r.peers.Range(func(id, p any) bool {
		if id.(string) == message.ID {
			return true
		}

		if err := p.(*peer).socket.WriteJSON(message); err != nil {
			log.Printf("failed to write json to %s: %v\n", id.(string), message)
		}

		return true
	})
}
