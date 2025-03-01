package echos

import (
	"flag"
	"net/http"
	"os"
	"text/template"

	"github.com/gorilla/websocket"
	"github.com/pion/logging"
)

var (
	addr          = flag.String("addr", ":8080", "http service address")
	stunAddr      = flag.String("stunAddr", "stun.l.google.com:19302", "stun server address")
	domain        = flag.String("domain", "localhost:8080", "http service domain")
	secure        = flag.Bool("secure", false, "ws secure")
	indexTemplate = &template.Template{}
	Rooms         map[string]*Room
	log           = logging.NewDefaultLoggerFactory().NewLogger("sfu-ws")
)

func Start(upgrader *websocket.Upgrader, auth authFunc) {
	flag.Parse()

	Rooms = make(map[string]*Room)

	indexHTML, err := os.ReadFile("static/src/index.html")
	if err != nil {
		panic(err)
	}
	indexTemplate = template.Must(template.New("").Parse(string(indexHTML)))

	fs := http.FileServer(http.Dir("static/"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.HandleFunc("/websocket", websocketHandler(upgrader, auth))

	http.HandleFunc("/meeting", func(w http.ResponseWriter, r *http.Request) {
		protocol := "ws"
		if *secure {
			protocol = "wss"
		}
		if err = indexTemplate.Execute(w, protocol+"://"+*domain+"/websocket?"+r.URL.RawQuery); err != nil {
			log.Errorf("Failed to parse index template: %v", err)
		}
	})

	log.Infof("Starting server on %s", *addr)
	if err = http.ListenAndServe(*addr, nil); err != nil {
		log.Errorf("Failed to start HTTP server: %v", err)
	}
}
