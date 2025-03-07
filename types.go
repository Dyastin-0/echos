package echos

import (
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v4"
)

type authFunc func(r *http.Request) bool

type websocketMessage struct {
	Name  string `json:"name,omitempty"`
	ID    string `json:"id,omitempty"`
	Event string `json:"event,omitempty"`
	Data  string `json:"data,omitempty"`
	Type  string `json:"type,omitempty"`
}

type peer struct {
	id         string
	connection *webrtc.PeerConnection
	socket     *threadSafeWriter
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
