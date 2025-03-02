package echos

import (
	"encoding/json"

	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
)

func NewPeer(r *Room, ws *threadSafeWriter, id string) (*peer, error) {
	pc, err := webrtc.NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:" + *stunAddr}},
		},
	})
	if err != nil {
		return nil, err
	}

	for _, typ := range []webrtc.RTPCodecType{webrtc.RTPCodecTypeVideo, webrtc.RTPCodecTypeAudio} {
		if _, err := pc.AddTransceiverFromKind(typ, webrtc.RTPTransceiverInit{
			Direction: webrtc.RTPTransceiverDirectionRecvonly,
		}); err != nil {
			log.Errorf("Failed to add transceiver: %v", err)
			return nil, err
		}
	}

	r.listLock.Lock()
	peer := &peer{
		connection: pc,
		socket:     ws,
		id:         id,
	}
	r.peers = append(r.peers, peer)
	r.listLock.Unlock()

	onICECandidate(pc, ws)
	onConnectionStateChange(pc, r)
	onTrack(pc, r)
	onICEConnectionStateChange(pc)

	r.signalPeerConnections()

	return peer, nil
}

func onICECandidate(pc *webrtc.PeerConnection, ws *threadSafeWriter) {
	pc.OnICECandidate(func(i *webrtc.ICECandidate) {
		if i == nil {
			return
		}

		candidateString, err := json.Marshal(i.ToJSON())
		if err != nil {
			log.Errorf("Failed to marshal candidate to json: %v", err)
			return
		}

		log.Infof("Send candidate to client: %s", candidateString)

		if writeErr := ws.WriteJSON(&websocketMessage{
			Event: "candidate",
			Data:  string(candidateString),
		}); writeErr != nil {
			log.Errorf("Failed to write JSON: %v", writeErr)
		}
	})
}

func onConnectionStateChange(pc *webrtc.PeerConnection, r *Room) {
	pc.OnConnectionStateChange(func(p webrtc.PeerConnectionState) {
		log.Infof("Connection state change: %s", p)

		switch p {
		case webrtc.PeerConnectionStateFailed:
			if err := pc.Close(); err != nil {
				log.Errorf("Failed to close PeerConnection: %v", err)
			}
		case webrtc.PeerConnectionStateClosed:
			r.signalPeerConnections()
		default:
		}
	})
}

func onTrack(pc *webrtc.PeerConnection, r *Room) {
	pc.OnTrack(func(t *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
		log.Errorf("Got remote track: Kind=%s, ID=%s, PayloadType=%d", t.Kind(), t.ID(), t.PayloadType())

		trackLocal := r.addTrack(t)
		defer r.removeTrack(trackLocal)

		buf := make([]byte, 1500)
		rtpPkt := &rtp.Packet{}

		for {
			i, _, err := t.Read(buf)
			if err != nil {
				return
			}

			if err = rtpPkt.Unmarshal(buf[:i]); err != nil {
				log.Errorf("Failed to unmarshal incoming RTP packet: %v", err)
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

func onICEConnectionStateChange(pc *webrtc.PeerConnection) {
	pc.OnICEConnectionStateChange(func(is webrtc.ICEConnectionState) {
		log.Infof("ICE connection state changed: %s", is)
	})
}
