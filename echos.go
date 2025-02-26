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
	indexTemplate = &template.Template{}
	Rooms         map[string]*Room
	log           = logging.NewDefaultLoggerFactory().NewLogger("sfu-ws")
)

func Start(upgrader *websocket.Upgrader, auth authFunc) {
	flag.Parse()

	Rooms = make(map[string]*Room)

	indexHTML, err := os.ReadFile("index.html")
	if err != nil {
		panic(err)
	}
	indexTemplate = template.Must(template.New("").Parse(string(indexHTML)))

	http.HandleFunc("/websocket", websocketHandler(upgrader, auth))

	http.HandleFunc("/meeting", func(w http.ResponseWriter, r *http.Request) {
		if err = indexTemplate.Execute(w, "wss://"+r.Host+"/websocket?"+r.URL.RawQuery); err != nil {
			log.Errorf("Failed to parse index template: %v", err)
		}
	})

	if err = http.ListenAndServe(*addr, nil); err != nil {
		log.Errorf("Failed to start http server: %v", err)
	}
}
