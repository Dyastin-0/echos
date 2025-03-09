package echos

import (
	"net/http"
)

type authFunc func(r *http.Request) bool

type websocketMessage struct {
	Name   string `json:"name,omitempty"`
	ID     string `json:"id,omitempty"`
	Event  string `json:"event,omitempty"`
	Data   string `json:"data,omitempty"`
	Type   string `json:"type,omitempty"`
	State  bool   `json:"state"`
	Target string `json:"target,omitempty"`
}
