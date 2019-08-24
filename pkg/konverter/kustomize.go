package konverter

import (
	"io/ioutil"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

// we will generate kustomization.yaml from scratch to avoid taking a dependency
// on kustomize. our requirements are simple for now

const (
	kustomizeAPIVersion = "kustomize.config.k8s.io/v1beta1"
	kustomizeKind       = "Kustomization"
)

// Kustomization holds the information needed to generate customized k8s api
// resources
type Kustomization struct {
	Kind       string   `json:"kind,omitempty" yaml:"kind,omitempty"`
	APIVersion string   `json:"apiVersion,omitempty" yaml:"apiversion,omitempty"`
	Resources  []string `json:"resources,omitempty" yaml:"resources,omitempty"`
}

// NewKustomization creates a new Kustomization
func NewKustomization() *Kustomization {
	return &Kustomization{
		Kind:       kustomizeKind,
		APIVersion: kustomizeAPIVersion,
		Resources:  []string{},
	}
}

// AddResource adds a resource filenames to a Kustomization
func (k *Kustomization) AddResource(filename string) {
	k.Resources = append(k.Resources, filename)
}

// Save persists a kustomization to a file as yaml
func (k *Kustomization) Save(filename string) error {
	data, err := yaml.Marshal(k)
	if err != nil {
		return errors.Wrap(err, "error marshaling kustomization to yaml")
	}

	if err = ioutil.WriteFile(filename, data, 0644); err != nil {
		return errors.Wrapf(
			err,
			"error writing kustomization to %q",
			filename,
		)
	}

	return nil
}
