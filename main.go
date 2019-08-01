package main

import (
	"github.com/ryane/konvert/pkg/fetcher"
	log "github.com/sirupsen/logrus"
)

func main() {
	log.SetLevel(log.DebugLevel)
	// parse konvert.yaml
	// fetch helm source
	// run helm template
	hf := fetcher.NewHelmFetcher("stable/postgresql", "5.3.12")

	err := hf.Fetch()
	if err != nil {
		log.Fatal(err)
	}
}
