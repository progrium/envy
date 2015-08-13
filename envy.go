package main

import (
	"os"

	"github.com/progrium/envy/cmd"
)

func main() {
	if envy.ClientMode() {
		envy.RunClient(os.Args[1:])
		return
	}

	envy.SetupLogging()
	envy.CheckAdminCmd()
	envy.CheckSystemCmd()
	envy.Cmd.Execute()
}
