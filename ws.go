package echos

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v4"
)

func wsListen(pc *webrtc.PeerConnection, ws *threadSafeWriter) {
	message := &websocketMessage{}

	for {
		_, raw, err := ws.ReadMessage()
		if err != nil {
			log.Errorf("Failed to read message: %v", err)
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

			if err := pc.AddICECandidate(candidate); err != nil {
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

			if err := pc.SetRemoteDescription(answer); err != nil {
				log.Errorf("Failed to set remote description: %v", err)
				return
			}
		default:
			log.Errorf("unknown message: %+v", message)
		}
	}
}

func websocketHandler(upgrader *websocket.Upgrader, auth authFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, rq *http.Request) {
		if !auth(rq) {
			log.Errorf("Failed to upgrade HTTP to Websocket: ", fmt.Errorf("unauthorized"))
			return
		}

		roomID := rq.URL.Query().Get("room")

		if roomID == "" {
			log.Errorf("Failed to upgrade HTTP to Websocket: ", fmt.Errorf("bad request"))
			return
		}

		unsafeConn, err := upgrader.Upgrade(w, rq, nil)
		if err != nil {
			log.Errorf("Failed to upgrade HTTP to Websocket: ", err)
			return
		}

		r, ok := Rooms[roomID]
		if !ok {
			Rooms[roomID] = NewRoom()
			r = Rooms[roomID]
		}

		ws := &threadSafeWriter{unsafeConn, sync.Mutex{}}

		defer ws.Close()

		pc, err := NewPeer(r, ws)
		if err != nil {
			log.Errorf("Failed to creates a PeerConnection: %v", err)
			return
		}

		defer pc.Close()

		wsListen(pc, ws)
	}
}
