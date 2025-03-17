package echos

import (
	"net/http"
)

type authFunc func(r *http.Request) bool

type websocketMessage struct {
	Name       string `json:"name,omitempty"`
	ID         string `json:"id,omitempty"`
	Event      string `json:"event,omitempty"`
	Data       string `json:"data,omitempty"`
	AdData     string `json:"adData,omitempty"`
	Type       string `json:"type,omitempty"`
	State      bool   `json:"state"`
	Target     string `json:"target,omitempty"`
	AudioState bool   `json:"audioState"`
	VideoState bool   `json:"videoState"`
}
