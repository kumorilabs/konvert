package main

import (
	"github.com/ryane/konvert/pkg/konverter"
	"github.com/ryane/konvert/pkg/sources"
	log "github.com/sirupsen/logrus"
)

func main() {
	log.SetLevel(log.DebugLevel)
	// parse konvert.yaml
	// fetch helm source
	// run helm template
	hs := sources.NewHelmSource("stable/postgresql", "5.3.12")

	outputDirectory := "./output"
	converter := konverter.New(hs, outputDirectory)
	if err := converter.Run(); err != nil {
		log.Fatal(err)
	}
}
