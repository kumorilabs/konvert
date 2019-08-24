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
	// TODO: delete output directory if it already exists
	// TODO: prompt? add -f force flag?
	if err := os.MkdirAll(k.outputDirectory, os.ModePerm); err != nil {
		return errors.Wrapf(
			err,
			"error creating output dir %s",
			k.outputDirectory,
		)
	}

	kustfile := NewKustomization()
	kustfilename := filepath.Join(k.outputDirectory, "kustomization.yaml")

	for _, res := range resources {
		resfile := resourceFileName(res)
		outfile := filepath.Join(k.outputDirectory, resfile)

		data, err := yaml.Marshal(res.Object)
		if err != nil {
			return errors.Wrapf(
				err,
				"error marshaling resource %q into yaml",
				res.ID(),
			)
		}

		err = ioutil.WriteFile(outfile, data, 0644)
		if err != nil {
			return errors.Wrapf(
				err,
				"error writing resource %q to %s",
				res.ID(),
				outfile,
			)
		}
		kustfile.AddResource(resfile)
	}

	if err := kustfile.Save(kustfilename); err != nil {
		return err
	}

	return nil
}

func resourceFileName(res sources.Resource) string {
	return strings.ToLower(
		fmt.Sprintf("%s-%s.yaml", res.GetKind(), res.GetName()),
	)
}
