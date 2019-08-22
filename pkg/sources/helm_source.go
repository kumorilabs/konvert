package sources

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

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

	fetchArgs := []string{"fetch", "--untar"}
	if h.Version != "" {
		fetchArgs = append(fetchArgs, "--version", h.Version)
		fetchArgs = append(fetchArgs, "--destination", fmt.Sprintf("%s-%s", h.Name, h.Version))
	} else {
		fetchArgs = append(fetchArgs, "--destination", h.Name)
	}
	fetchArgs = append(fetchArgs, h.Name)

	h.log.Debugf("Running helm: %v", fetchArgs)
	cmd := h.command(cachedir, fetchArgs...)
	out, err := cmd.CombinedOutput()

	if err != nil {
		return errors.Wrapf(err, "error running helm %v: %q", fetchArgs, string(out))
	}

	return nil
}

// Generate creates raw manifests
func (h *HelmSource) Generate() error { return nil }

// Kustomize creates customizations
func (h *HelmSource) Kustomize() error { return nil }

func (h *HelmSource) command(wd string, args ...string) *exec.Cmd {
	cmd := exec.Command("helm", args...)
	cmd.Dir = wd
	return cmd
}

func cacheDir() string {
	return filepath.Join(os.Getenv("HOME"), ".cache", "konvert", "helm")
}
