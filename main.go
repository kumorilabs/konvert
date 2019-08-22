package main

import (
	"github.com/ryane/konvert/pkg/sources"
	log "github.com/sirupsen/logrus"
)

func main() {
	log.SetLevel(log.DebugLevel)
	// parse konvert.yaml
	// fetch helm source
	// run helm template
	hs := sources.NewHelmSource("stable/postgresql", "5.3.12")

	err := hs.Fetch()
	if err != nil {
		log.Fatal(err)
	}
}
