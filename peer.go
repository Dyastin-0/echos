package echos

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
)

type peer struct {
	id         string
	name       string
	connection *webrtc.PeerConnection
	socket     *ThreadSafeSocketWriter
}

var RTPCodecTypes = []webrtc.RTPCodecType{webrtc.RTPCodecTypeVideo, webrtc.RTPCodecTypeAudio}

func NewPeer(r *Room, ws *ThreadSafeSocketWriter, id, name, stunAddr string) (*peer, error) {
	pc, err := webrtc.NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:" + stunAddr}},
		},
	})
	if err != nil {
		return nil, err
	}

	for _, typ := range RTPCodecTypes {
		if _, err := pc.AddTransceiverFromKind(typ, webrtc.RTPTransceiverInit{
			Direction: webrtc.RTPTransceiverDirectionRecvonly,
		}); err != nil {
			return nil, err
		}
	}

	peer := &peer{
		connection: pc,
		socket:     ws,
		id:         id,
		name:       name,
	}
	r.peers.Store(peer.id, peer)

	peer.start(r)
	r.signalPeerConnections()

	return peer, nil
}

func (p *peer) start(r *Room) {
	p.onICECandidate()
	p.onConnectionStateChange(r)
	p.onTrack(r)
	p.onICEConnectionStateChange()
}

func (p *peer) onICECandidate() {
	p.connection.OnICECandidate(func(i *webrtc.ICECandidate) {
		if i == nil {
			return
		}

		candidateBytes, err := json.Marshal(i.ToJSON())
		if err != nil {
			log.Printf("failed to Marshal canditate: %v\n", err)
			return
		}

		err = p.socket.WriteJSON(
			&websocketMessage{
				Event: "candidate",
				Data:  string(candidateBytes),
			},
		)
		if err != nil {
			log.Printf("failed to write candidate message: %v\n", err)
		}
	})
}

func (p *peer) onConnectionStateChange(r *Room) {
	p.connection.OnConnectionStateChange(func(pcs webrtc.PeerConnectionState) {
		switch pcs {
		case webrtc.PeerConnectionStateFailed:
			if err := p.connection.Close(); err != nil {
				log.Printf("failed to close peer connection: %v\n", err)
			}
		case webrtc.PeerConnectionStateClosed:
			r.signalPeerConnections()
		default:
		}
	})
}

func (p *peer) onTrack(r *Room) {
	p.connection.OnTrack(func(t *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
		fmt.Printf(
			"got remote track: kind=%s, id=%s, payloadType=%d\n",
			t.Kind(),
			t.ID(),
			t.PayloadType(),
		)

		trackLocal := r.addTrack(t)
		defer r.removeTrack(trackLocal)

		buf := make([]byte, 1500)
		rtpPkt := &rtp.Packet{}

		for {
			i, _, err := t.Read(buf)
			if err != nil {
				log.Println("failed to read remote track")
				return
			}

			if err = rtpPkt.Unmarshal(buf[:i]); err != nil {
				log.Printf("failed to unmarshal incoming rtp packet: %v\n", err)
				return
			}

			rtpPkt.Extension = false
			rtpPkt.Extensions = nil

			if err = trackLocal.WriteRTP(rtpPkt); err != nil {
				return
			}
		}
	})
}

func (p *peer) onICEConnectionStateChange() {
	p.connection.OnICEConnectionStateChange(func(is webrtc.ICEConnectionState) {
		log.Printf("ice connection state changed for %s: %s\n", p.id, is)
	})
}
