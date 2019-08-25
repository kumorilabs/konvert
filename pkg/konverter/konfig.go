package konverter

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

const konvertFileName = "konvert.yaml"

// Konfig contains the configuration for konvert
type Konfig struct {
	// TODO: typemeta?
	Source           *konvertSource `json:"source,omitempty" yaml:"source,omitempty"`
	konvertDirectory string
}

type konvertSource struct {
	Type   string                 `json:"type,omitempty" yaml:"type,omitempty"`
	Config map[string]interface{} `json:"config,omitempty" yaml:"config,omitempty"`
}

type helmSourceConfig struct {
	Name    string                 `json:"name,omitempty" yaml:"name,omitempty"`
	Version string                 `json:"version,omitempty" yaml:"version,omitempty"`
	Values  map[string]interface{} `json:"values,omitempty" yaml:"values,omitempty"`
}

// LoadConfig loads Konfig from a yaml file
func LoadConfig() (*Konfig, error) {
	workingDir, err := os.Getwd()
	if err != nil {
		return nil, errors.Wrap(err, "error getting working directory")
	}

	konvertFile := filepath.Join(workingDir, konvertFileName)
	data, err := ioutil.ReadFile(konvertFile)
	if err != nil {
		return nil, errors.Wrapf(err, "error reading %s", konvertFile)
	}

	var konfig *Konfig
	if err := yaml.Unmarshal(data, &konfig); err != nil {
		return nil, errors.Wrapf(
			err,
			"error unmarshaling config from %s",
			konvertFile,
		)
	}

	konfig.konvertDirectory = workingDir
	return konfig, nil
}
