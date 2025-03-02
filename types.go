package echos

import (
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v4"
)

type authFunc func(r *http.Request) bool

type websocketMessage struct {
	Event string `json:"event"`
	Data  string `json:"data"`
}

type peer struct {
	Connection *webrtc.PeerConnection
	websocket  *threadSafeWriter
}

type threadSafeWriter struct {
	*websocket.Conn
	sync.Mutex
}

func (t *threadSafeWriter) WriteJSON(v interface{}) error {
	t.Lock()
	defer t.Unlock()

	return t.Conn.WriteJSON(v)
}
