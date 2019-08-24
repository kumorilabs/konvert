package konverter

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/ryane/konvert/pkg/sources"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// Konverter converts sources into Kustomize bases
type Konverter struct {
	source          sources.Source
	outputDirectory string
	log             *log.Entry
}

// New creates a new Konverter
func New(source sources.Source, outdir string) *Konverter {
	return &Konverter{
		source:          source,
		outputDirectory: outdir,
		log: log.WithFields(log.Fields{
			"pkg": "konverter",
		}),
	}
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

	return k.writeResources(resources)
}

func (k *Konverter) writeResources(resources []sources.Resource) error {
	if err := os.MkdirAll(k.outputDirectory, os.ModePerm); err != nil {
		return errors.Wrapf(
			err,
			"error creating output dir %s",
			k.outputDirectory,
		)
	}

	for _, res := range resources {
		filename := filepath.Join(k.outputDirectory, resourceFileName(res))

		data, err := yaml.Marshal(res.Object)
		if err != nil {
			return errors.Wrapf(
				err,
				"error marshaling resource %q into yaml",
				res.ID(),
			)
		}

		err = ioutil.WriteFile(filename, data, 0644)
		if err != nil {
			return errors.Wrapf(
				err,
				"error writing resource %q to %s",
				res.ID(),
				filename,
			)
		}
	}
	return nil
}

func resourceFileName(res sources.Resource) string {
	return strings.ToLower(
		fmt.Sprintf("%s-%s.yaml", res.GetKind(), res.GetName()),
	)
}
