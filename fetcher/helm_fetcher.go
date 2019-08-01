package fetcher

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"
)

type HelmFetcher struct {
	SourceURL string
	Name      string
	Version   string
}

func (h *HelmFetcher) Fetch() error {
	cachedir := cacheDir()
	log.Printf("cachedir: %s", cachedir)
	if err := os.MkdirAll(cachedir, os.ModePerm); err != nil {
		return errors.Wrapf(err, "error creating cache dir %s", cachedir)
	}

	fetchArgs := []string{"fetch", "--untar"}
	fetchArgs = append(fetchArgs, "stable/"+h.Name)

	cmd := h.command(cachedir, fetchArgs...)
	out, err := cmd.Output()
	if err != nil {
		return errors.Wrapf(err, "error running %v: %q", cmd)
	}

	log.Println(string(out))

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
