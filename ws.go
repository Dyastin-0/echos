package echos

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
)

func WebsocketHandler(upgrader *websocket.Upgrader, auth authFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, rq *http.Request) {
		if !auth(rq) {
			log.Errorf("Failed to upgrade HTTP to Websocket: ", fmt.Errorf("unauthorized"))
			return
		}

		roomID := rq.URL.Query().Get("room")
		if roomID == "" {
			log.Errorf("Failed to upgrade HTTP to Websocket: ", fmt.Errorf("bad request"))
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		peerID := rq.URL.Query().Get("id")
		if peerID == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(HTTPresponse{
				"error": "ID missing",
			})
		}

		peerName := rq.URL.Query().Get("name")
		if peerName == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(HTTPresponse{
				"error": "Name missing",
			})
		}

		roomsMutex.Lock()
		r, ok := Rooms[roomID]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			roomsMutex.Unlock()
			return
		}
		roomsMutex.Unlock()

		conn, err := upgrader.Upgrade(w, rq, nil)
		if err != nil {
			log.Errorf("Failed to upgrade HTTP to Websocket: ", err)
			return
		}

		ws := NewThreadSafeSocketWriter(conn)

		defer ws.Close()

		peer, err := NewPeer(r, ws, peerID, peerName)
		if err != nil {
			log.Errorf("Failed to creates a PeerConnection: %v", err)
			return
		}

		defer peer.connection.Close()

		r.wsListen(peer)
	}
}
