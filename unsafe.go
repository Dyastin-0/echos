package echos

import (
	"net/http"

	"github.com/gorilla/websocket"
)

func UnsafeUpgrader() *websocket.Upgrader {
	return &websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
}

func UnSafeAuth(r *http.Request) bool {
	return true
}
