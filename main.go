package main

import (
	"os"

	"github.com/kumorilabs/konvert/cmd"
	log "github.com/sirupsen/logrus"
)

func main() {
	log.SetLevel(log.DebugLevel)
	cmd.Execute(os.Args[1:])
}
