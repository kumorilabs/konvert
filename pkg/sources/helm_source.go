package sources

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"
)

// helmSource represents a helm chart
type helmSource struct {
	Repo      string                 `json:"repo,omitempty" yaml:"repo,omitempty"`
	ChartName string                 `json:"name,omitempty" yaml:"name,omitempty"`
	Version   string                 `json:"version,omitempty" yaml:"version,omitempty"`
	Namespace namespace              `json:"namespace,omitempty" yaml:"namespace,omitempty"`
	Values    map[string]interface{} `json:"values,omitempty" yaml:"values,omitempty"`
	log       *log.Entry
}

type namespace struct {
	Name   string            `json:"name,omitempty" yaml:"name,omitempty"`
	Create bool              `json:"create,omitempty" yaml:"create,omitempty"`
	Labels map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
}

// NewHelmSourceFromConfig creates a new Helm Source from a config map
func NewHelmSourceFromConfig(config map[string]interface{}) (Source, error) {
	var source *helmSource

	// marshal map into a byte slice
	b, err := yaml.Marshal(config)
	if err != nil {
		return nil, errors.Wrap(err, "error marshaling helm source config")
	}

	// unmarshal into a helmSource
	err = yaml.Unmarshal(b, &source)
	if err != nil {
		return nil, errors.Wrap(err, "error unmarshaling helm source config")
	}

	if strings.Contains(source.ChartName, "/") {
		nameParts := strings.Split(source.ChartName, "/")
		if len(nameParts) > 1 {
			source.Repo = nameParts[0]
			source.ChartName = nameParts[1]
		}
	}

	source.log = log.WithFields(log.Fields{
		"pkg":    "sources",
		"source": "helm",
	})

	return source, nil
}

// Fetch downloads the source
func (h *helmSource) Fetch() error {
	chartYaml := filepath.Join(h.chartDir(), h.Name(), "Chart.yaml")
	if _, err := os.Stat(chartYaml); err == nil {
		// chart is already extracted, return
		h.log.Debug("found chart in cache")
		return nil
	}

	cachedir := cacheDir()
	if err := os.MkdirAll(cachedir, os.ModePerm); err != nil {
		return errors.Wrapf(err, "error creating cache dir %s", cachedir)
	}

	cmd := h.fetchCommand()
	out, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "error running fetch command: %q", string(out))
	}

	return nil
}

// Generate creates raw manifests
func (h *helmSource) Generate() ([]Resource, error) {
	var valuesFile string

	if len(h.Values) > 0 {
		tmpfile, err := ioutil.TempFile("", fmt.Sprintf("konvert-%s-*", h.Name()))
		if err != nil {
			return nil, errors.Wrap(err, "error creating values file")
		}

		defer func() {
			_ = os.Remove(tmpfile.Name())
		}()

		b, err := yaml.Marshal(h.Values)
		if err != nil {
			return nil, errors.Wrap(err, "error marshaling values")
		}

		if _, err := tmpfile.Write(b); err != nil {
			return nil, errors.Wrap(err, "error writing values file")
		}

		valuesFile = tmpfile.Name()
	}

	cmd := h.templateCommand(valuesFile)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, errors.Wrapf(err, "error running template command: %q", string(out))
	}

	resources, err := h.decode(bytes.NewReader(out))
	if err != nil {
		return resources, errors.Wrap(err, "error decoding template output")
	}

	h.log.Debugf("found %d resources", len(resources))

	return resources, nil
}

// Name returns the name of this source
func (h *helmSource) Name() string {
	return h.ChartName
}

func (h *helmSource) templateCommand(valuesFile string) *exec.Cmd {
	args := []string{
		"template",
		"--name", h.Name(),
		"--namespace", h.Namespace.Name,
	}

	if valuesFile != "" {
		args = append(args, "-f", valuesFile)
	}

	args = append(args, filepath.Join(h.chartDir(), h.Name()))

	cmd := exec.Command("helm", args...)

	h.log.
		WithField("cmd", strings.Join(append([]string{"helm"}, args...), " ")).
		Debug("Built helm template command")

	return cmd
}

func (h *helmSource) fetchCommand() *exec.Cmd {
	args := []string{"fetch", "--untar"}
	if h.Version != "" {
		args = append(args, "--version", h.Version)
	}
	args = append(
		args,
		"--destination", h.chartDir(),
		h.Repo+"/"+h.Name(),
	)
	cmd := exec.Command("helm", args...)

	h.log.
		WithField("cmd", strings.Join(append([]string{"helm"}, args...), " ")).
		Debug("Built helm fetch command")

	return cmd
}

func (h *helmSource) decode(in io.Reader) ([]Resource, error) {
	var (
		result []Resource
		err    error
	)

	decoder := k8syaml.NewYAMLOrJSONDecoder(in, 1024)

	for err == nil {
		var out Resource
		err = decoder.Decode(&out)
		if err == nil && len(out.Object) > 0 {
			if out.IsList() {
				items, err := out.ToList()
				if err != nil {
					return nil, errors.Wrap(err, "failed to explode list")
				}
				result = append(result, items...)
			} else {
				result = append(result, out)
			}
		}
	}
	if err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode input")
	}

	return result, nil
}

func (h *helmSource) chartDir() string {
	var d string
	if h.Version != "" {
		d = fmt.Sprintf("%s-%s", h.Name(), h.Version)
	} else {
		d = h.Name()
	}

	return filepath.Join(
		cacheDir(),
		h.Repo,
		d,
	)
}

func cacheDir() string {
	return filepath.Join(os.Getenv("HOME"), ".cache", "konvert", "helm")
}
