package echos

import (
	"flag"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	"github.com/pion/logging"
)

var (
	addr       = flag.String("addr", ":8080", "http service address")
	stunAddr   = flag.String("stunAddr", "stun.l.google.com:19302", "stun server address")
	Rooms      map[string]*Room
	roomsMutex sync.RWMutex
	log        = logging.NewDefaultLoggerFactory().NewLogger("sfu-ws")
)

func Start(upgrader *websocket.Upgrader, auth authFunc) {
	flag.Parse()
	Rooms = make(map[string]*Room)

	router := chi.NewRouter()

	router.Use(cors)

	router.Post("/create", createRoom)
	router.Post("/check", checkRoom)
	router.Handle("/websocket", websocketHandler(upgrader, auth))

	log.Infof("Starting server on %s", *addr)
	if err := http.ListenAndServe(*addr, router); err != nil {
		log.Errorf("Failed to start HTTP server: %v", err)
	}
}
