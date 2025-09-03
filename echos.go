package echos

import (
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	"github.com/pion/logging"
)

type Echos struct {
	addr     string stunAddr string Rooms    sync.Map
	log      logging.LeveledLogger
}

func New(addr, stunAddr string) *Echos {
	return &Echos{
		addr:     addr,
		stunAddr: stunAddr,
		log:      logging.NewDefaultLoggerFactory().NewLogger("echos"),
	}
}

func (e *Echos) Start(upgrader *websocket.Upgrader, auth authFunc) error {
	router := chi.NewRouter()
	router.Use(e.cors)

	router.Post("/api/create", e.CreateRoom)
	router.Post("/api/check", e.CheckRoom)
	router.Handle("/api/ws", e.WebsocketHandler(upgrader, auth))

	e.log.Infof("starting server on %s", e.addr)
	if err := http.ListenAndServe(e.addr, router); err != nil {
		return err
	}

	return nil
}

func (e *Echos) killRoomIfEmpty(id string, deletech chan bool) {
	<-deletech
	e.Rooms.Delete(id)
}
