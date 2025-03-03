package echos

import (
	"flag"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/pion/logging"
)

var (
	addr     = flag.String("addr", ":8080", "http service address")
	stunAddr = flag.String("stunAddr", "stun.l.google.com:19302", "stun server address")
	Rooms    map[string]*Room
	log      = logging.NewDefaultLoggerFactory().NewLogger("sfu-ws")
)

func Start(upgrader *websocket.Upgrader, auth authFunc) {
	flag.Parse()

	Rooms = make(map[string]*Room)

	http.HandleFunc("/websocket", websocketHandler(upgrader, auth))

	log.Infof("Starting server on %s", *addr)
	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Errorf("Failed to start HTTP server: %v", err)
	}
}
