package fetcher

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type HelmFetcher struct {
	Name    string
	Version string
	log     *log.Entry
}

func NewHelmFetcher(name, version string) *HelmFetcher {
	return &HelmFetcher{
		Name:    name,
		Version: version,
		log: log.WithFields(log.Fields{
			"pkg":     "fetcher",
			"fetcher": "helm",
		}),
	}
}

func (h *HelmFetcher) Fetch() error {
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

func (h *HelmFetcher) command(wd string, args ...string) *exec.Cmd {
	cmd := exec.Command("helm", args...)
	cmd.Dir = wd
	return cmd
}

func cacheDir() string {
	return filepath.Join(os.Getenv("HOME"), ".cache", "konvert")
}
