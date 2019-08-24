package konverter

import (
	"github.com/pkg/errors"
	"github.com/ryane/konvert/pkg/sources"
	log "github.com/sirupsen/logrus"
)

// Konverter converts sources into Kustomize bases
type Konverter struct {
	source          sources.Source
	outputDirectory string
}

// New creates a new Konverter
func New(source sources.Source, outdir string) *Konverter {
	return &Konverter{source, outdir}
}

// Run runs konverts the source
func (k *Konverter) Run() error {
	if err := k.source.Fetch(); err != nil {
		return errors.Wrap(err, "error fetching source")
	}

	resources, err := k.source.Generate()
	if err != nil {
		return errors.Wrap(err, "error generating source")
	}

	for _, res := range resources {
		log.Debugf("res: %s", res.GetKind())
	}

	return nil
}
