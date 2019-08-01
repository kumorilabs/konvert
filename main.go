package main

import (
	"fmt"
	"log"

	"github.com/ryane/konvert/fetcher"
)

func main() {
	// parse konvert.yaml
	// fetch helm source
	// run helm template
	fmt.Println("konvert")

	hf := &fetcher.HelmFetcher{
		Name:    "stable/postgresql",
		Version: "5.3.12",
	}

	err := hf.Fetch()
	if err != nil {
		log.Fatal(err)
	}
}
