package main

import (
	"github.com/Dyastin-0/echos"
)

func main() {
	echos.StartSTUN()
	echos.Start(echos.UnsafeUpgrader(), echos.UnSafeAuth)
}
