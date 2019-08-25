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
	konfig                 *Konfig
	source                 sources.Source
	kustomizationDirectory string
	log                    *log.Entry
}

// New creates a new Konverter
func New(konfig *Konfig) *Konverter {
	return &Konverter{
		konfig: konfig,
		log: log.WithFields(log.Fields{
			"pkg": "konverter",
		}),
	}
}

// Run runs konverts the source
func (k *Konverter) Run() error {
	if err := k.getSource(); err != nil {
		return errors.Wrap(err, "error getting source")
	}

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
	if err := os.MkdirAll(k.kustomizationDirectory, os.ModePerm); err != nil {
		return errors.Wrapf(
			err,
			"error creating output dir %s",
			k.kustomizationDirectory,
		)
	}

	kustfile := NewKustomization()
	kustfilename := filepath.Join(k.kustomizationDirectory, "kustomization.yaml")

	for _, res := range resources {
		resfile := resourceFileName(res)
		outfile := filepath.Join(k.kustomizationDirectory, resfile)

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

func (k *Konverter) getSource() error {
	t := strings.ToLower(k.konfig.Source.Type)
	switch t {
	case "helm":
		s, err := getHelmSource(k.konfig.Source.Config)
		if err != nil {
			return err
		}
		k.source = s
		k.kustomizationDirectory = filepath.Join(
			k.konfig.konvertDirectory,
			s.Name(),
		)
	default:
		return fmt.Errorf("unsupported source type: %s", k.konfig.Source.Type)
	}

	return nil
}

func getHelmSource(config map[string]interface{}) (sources.Source, error) {
	var helmConfig *helmSourceConfig
	if err := unmarshalConfig(config, &helmConfig); err != nil {
		return nil, errors.Wrap(err, "error unmarshaling helm source config")
	}

	// TODO: validate helmConfig
	return sources.NewHelmSource(helmConfig.Name, helmConfig.Version), nil
}

func unmarshalConfig(config, configOut interface{}) error {
	b, err := yaml.Marshal(config)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(b, configOut)
}

func resourceFileName(res sources.Resource) string {
	return strings.ToLower(
		fmt.Sprintf("%s-%s.yaml", res.GetKind(), res.GetName()),
	)
}
