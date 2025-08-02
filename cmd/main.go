package main

import (
	"flag"

	"github.com/Dyastin-0/echos"
)

func main() {
	addr := flag.String("addr", ":8080", "set echos addr")
	stunAddr := flag.String("stunAddr", "stun.l.google.com:19302", "set stun server")

	echos.StartSTUN()

	s := echos.New(*addr, *stunAddr)

	err := s.Start(echos.UnsafeUpgrader(), echos.UnSafeAuth)
	if err != nil {
		panic(err)
	}
}
