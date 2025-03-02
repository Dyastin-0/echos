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
	echos.Start(echos.UnsafeUpgrader(), echos.UnSafeAuth)
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
