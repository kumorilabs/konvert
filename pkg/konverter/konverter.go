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

// TODO: these need to be configurable
var transformers = []Transformer{
	NewRemoveLabelsTransformer(
		RemoveLabelsTransformerConfig{Names: []string{
			"chart",
			"heritage",
			"release",
		}},
	),
}

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
	var (
		resources []sources.Resource
		err       error
	)

	if err = k.getSource(); err != nil {
		return errors.Wrap(err, "error getting source")
	}

	if err = k.source.Fetch(); err != nil {
		return errors.Wrap(err, "error fetching source")
	}

	resources, err = k.source.Generate()
	if err != nil {
		return errors.Wrap(err, "error generating source")
	}

	for _, t := range transformers {
		resources, err = t.Transform(resources)
		if err != nil {
			return errors.Wrapf(err, "error running %s transformer", t.Name())
		}
	}

	if err := k.writeResources(resources); err != nil {
		return errors.Wrap(err, "error writing konverted resources")
	}

	return k.writeRootResources()
}

func (k *Konverter) writeRootResources() error {
	kustfile := NewKustomization()
	kustfilename := filepath.Join(k.konfig.konvertDirectory, "kustomization.yaml")

	if k.source.CreateNamespace() {
		namespaceName := k.source.Namespace()
		namespaceLabels := k.source.NamespaceLabels()

		resobj := map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Namespace",
			"metadata": map[string]interface{}{
				"name":   namespaceName,
				"labels": namespaceLabels,
			},
		}

		data, err := yaml.Marshal(resobj)
		if err != nil {
			return errors.Wrapf(
				err,
				"error marshaling namespace %q into yaml",
				namespaceName,
			)
		}
		// TODO: check if exists?
		nsfile := "namespace.yaml"
		err = ioutil.WriteFile(nsfile, data, 0644)
		if err != nil {
			return errors.Wrapf(
				err,
				"error writing namespace to %s",
				nsfile,
			)
		}
		kustfile.AddResource(nsfile)
	}

	kustfile.AddResource(k.source.Name())
	// TODO: check if exists?
	kustfile.Save(kustfilename)
	return nil
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
		s, err := sources.NewHelmSourceFromConfig(k.konfig.Source.Config)
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

func resourceFileName(res sources.Resource) string {
	return strings.ToLower(
		fmt.Sprintf("%s-%s.yaml", res.GetKind(), res.GetName()),
	)
}
