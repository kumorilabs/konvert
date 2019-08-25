package sources

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"
)

// HelmSource represents a helm chart
type HelmSource struct {
	repo      string
	chartName string
	version   string
	log       *log.Entry
}

// NewHelmSource creates a new Helm Source
func NewHelmSource(name, version string) *HelmSource {
	nameParts := strings.Split(name, "/")
	// TODO: verify valid name

	return &HelmSource{
		repo:      nameParts[0],
		chartName: nameParts[1],
		version:   version,
		log: log.WithFields(log.Fields{
			"pkg":    "sources",
			"source": "helm",
		}),
	}
}

// Fetch downloads the source
func (h *HelmSource) Fetch() error {
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
func (h *HelmSource) Generate() ([]Resource, error) {
	cmd := h.templateCommand()
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

// Kustomize creates customizations
func (h *HelmSource) Kustomize() error { return nil }

// Name returns the name of this source
func (h *HelmSource) Name() string {
	return h.chartName
}

func (h *HelmSource) templateCommand() *exec.Cmd {
	args := []string{
		"template",
		"--name", h.Name(),
		filepath.Join(h.chartDir(), h.Name()),
	}

	cmd := exec.Command("helm", args...)

	h.log.
		WithField("cmd", strings.Join(append([]string{"helm"}, args...), " ")).
		Debug("Built helm template command")

	return cmd
}

func (h *HelmSource) fetchCommand() *exec.Cmd {
	args := []string{"fetch", "--untar"}
	if h.version != "" {
		args = append(args, "--version", h.version)
	}
	args = append(
		args,
		"--destination", h.chartDir(),
		h.repo+"/"+h.Name(),
	)
	cmd := exec.Command("helm", args...)

	h.log.
		WithField("cmd", strings.Join(append([]string{"helm"}, args...), " ")).
		Debug("Built helm fetch command")

	return cmd
}

func (h *HelmSource) decode(in io.Reader) ([]Resource, error) {
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

func (h *HelmSource) chartDir() string {
	var d string
	if h.version != "" {
		d = fmt.Sprintf("%s-%s", h.Name(), h.version)
	} else {
		d = h.Name()
	}

	return filepath.Join(
		cacheDir(),
		h.repo,
		d,
	)
}

func cacheDir() string {
	return filepath.Join(os.Getenv("HOME"), ".cache", "konvert", "helm")
}
