package main

import (
	"log"
	"os"

	"github.com/progrium/envy/cmd"
)

func main() {
	log.SetFlags(0)

	if envy.DetectSocket() {
		envy.SocketClient(os.Args[1:])
		return
	}

	exitCh := make(chan int)
	go func() {
		os.Exit(<-exitCh)
	}()

	envy.RunCmd(os.Args[1:], os.Stdin, os.Stdout, os.Stderr, exitCh, nil)
}
