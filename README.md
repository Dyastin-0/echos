# Echos

#### How to Use

```go
package main

import (
	"github.com/Dyastin-0/echos"
)

func main() {
	echos.StartSTUN() // if you want a dedicated STUN server

	// pass a *websocket.Upgrader to handle CheckOrigin
	// and func (r *http.Request) bool to handle auth
    // uses the `addr` flag
	echos.Start(echos.UnsafeUpgrader(), echos.UnSafeAuth)
}
```
or use the handlers

```go
package main

import (
	"net/http"

	"github.com/Dyastin-0/echos"
	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
)

var upgrader = &websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Handle origin
		return true
	},
}

func auth(r *http.Request) bool {
	// Handle auth
	return true
}

func main() {
	router := chi.NewRouter()

	// Use the handlers
	router.Post("/create", echos.CreateRoom)
	router.Post("/check", echos.CheckRoom)
	router.Handle("/ws", echos.WebsocketHandler(upgrader, auth))

	if err := http.ListenAndServe(":8080", router); err != nil {
		panic(err)
	}
}

```

##### Flags

```
-addr=:42069
```

```
-stunAddr=stun.your.domain:3478 -domain=your.domain.com -secure
```

##### Build and Run

```bash
make install # will build the binary and move necessary files to the specified path
```
