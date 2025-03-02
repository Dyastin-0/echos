package echos

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

func websocketHandler(upgrader *websocket.Upgrader, auth authFunc) http.HandlerFunc {
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
			peerID = "Anonymous"
		}

		unsafeConn, err := upgrader.Upgrade(w, rq, nil)
		if err != nil {
			log.Errorf("Failed to upgrade HTTP to Websocket: ", err)
			return
		}

		r, ok := Rooms[roomID]
		if !ok {
			Rooms[roomID] = NewRoom(roomID)
			r = Rooms[roomID]
		}

		ws := &threadSafeWriter{unsafeConn, sync.Mutex{}}

		defer ws.Close()

		peer, err := NewPeer(r, ws, peerID)
		if err != nil {
			log.Errorf("Failed to creates a PeerConnection: %v", err)
			return
		}

		defer peer.connection.Close()

		r.wsListen(peer)
	}
}
