package echos

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

func (e *Echos) WebsocketHandler(upgrader *websocket.Upgrader, auth authFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, rq *http.Request) {
		enc := json.NewEncoder(w)

		if !auth(rq) {
			log.Printf("failed to upgrade connection: unauthorized")
			return
		}

		roomID := rq.URL.Query().Get("room")
		if roomID == "" {
			w.WriteHeader(http.StatusBadRequest)
			enc.Encode(HTTPresponse{
				"error": "missing room id",
			})
			return
		}

		peerID := rq.URL.Query().Get("id")
		if peerID == "" {
			w.WriteHeader(http.StatusBadRequest)
			enc.Encode(HTTPresponse{
				"error": "missing peer id",
			})
		}

		peerName := rq.URL.Query().Get("name")
		if peerName == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(HTTPresponse{
				"error": "missing name",
			})
		}

		r, ok := e.Rooms.Load(roomID)
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		conn, err := upgrader.Upgrade(w, rq, nil)
		if err != nil {
			log.Printf("failed to upgrade connection: %v", err)
			return
		}

		ws := NewThreadSafeSocketWriter(conn)
		defer ws.Close()

		peer, err := NewPeer(r.(*Room), ws, peerID, peerName, e.stunAddr)
		if err != nil {
			log.Printf("failed to create new peer: %v", err)
			return
		}
		defer peer.connection.Close()

		r.(*Room).wsListen(peer)
	}
}
