package sources

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// HelmSource represents a helm chart
type HelmSource struct {
	Name    string
	Version string
	log     *log.Entry
}

// NewHelmSource creates a new Helm Source
func NewHelmSource(name, version string) *HelmSource {
	return &HelmSource{
		Name:    name,
		Version: version,
		log: log.WithFields(log.Fields{
			"pkg":    "sources",
			"source": "helm",
		}),
	}
}

// Fetch downloads the source
func (h *HelmSource) Fetch() error {
	cachedir := cacheDir()
	h.log.Debugf("cachedir: %s", cachedir)

	if err := os.MkdirAll(cachedir, os.ModePerm); err != nil {
		return errors.Wrapf(err, "error creating cache dir %s", cachedir)
	}

	cmd := h.fetchCommand(cachedir)
	out, err := cmd.CombinedOutput()

	if err != nil {
		return errors.Wrapf(err, "error running fetch command: %q", string(out))
	}

	return nil
}

// Generate creates raw manifests
func (h *HelmSource) Generate() error { return nil }

// Kustomize creates customizations
func (h *HelmSource) Kustomize() error { return nil }

func (h *HelmSource) fetchCommand(wd string) *exec.Cmd {
	args := []string{"fetch", "--untar"}
	if h.Version != "" {
		args = append(
			args,
			"--version", h.Version,
			"--destination", fmt.Sprintf("%s-%s", h.Name, h.Version),
		)
	} else {
		args = append(args, "--destination", h.Name)
	}
	args = append(args, h.Name)
	cmd := exec.Command("helm", args...)
	cmd.Dir = wd

	h.log.
		WithField("cmd", strings.Join(append([]string{"helm"}, args...), " ")).
		Debug("Built helm fetch command")

	return cmd
}

func cacheDir() string {
	return filepath.Join(os.Getenv("HOME"), ".cache", "konvert", "helm")
}
