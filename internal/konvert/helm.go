package konvert

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/releaseutil"
	"helm.sh/helm/v3/pkg/repo"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

func (f *function) Render() ([]*kyaml.RNode, error) {
	configPath := filepath.Join(os.Getenv("HOME"), ".config", "konvert", "helmlib")
	cachePath := filepath.Join(os.Getenv("HOME"), ".cache", "konvert", "helmlib")

	settings := cli.New()
	settings.RegistryConfig = filepath.Join(configPath, "registry.json")
	settings.RepositoryConfig = filepath.Join(configPath, "repositories.yaml")
	settings.RepositoryCache = filepath.Join(cachePath, "repository")

	getters := getter.All(settings)
	c := downloader.ChartDownloader{
		Out:              os.Stderr,
		Getters:          getters,
		RepositoryConfig: settings.RepositoryConfig,
		RepositoryCache:  settings.RepositoryCache,
	}

	chartURL, err := repo.FindChartInRepoURL(f.Spec.Repo,
		f.Spec.Chart,
		f.Spec.Version,
		"", "", "",
		getters,
	)
	if err != nil {
		return nil, errors.Wrap(err, "unable to resolve chart url")
	}

	tmpDir, err := ioutil.TempDir("", "konvert")
	if err != nil {
		return nil, errors.Wrap(err, "unable to create temp directory")
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			// TODO: log?
			fmt.Printf("unable to remove temporary directory %s: %s", tmpDir, err)
		}
	}()

	archive, _, err := c.DownloadTo(chartURL, "", tmpDir)
	if err != nil {
		return nil, errors.Wrap(err, "unable to download chart")
	}

	chart, err := loader.Load(archive)
	if err != nil {
		return nil, errors.Wrap(err, "unable to load chart")
	}

	err = chartutil.SaveDir(chart, cachePath)
	if err != nil {
		return nil, errors.Wrap(err, "unable to save chart")
	}

	cfg := new(action.Configuration)
	client := action.NewInstall(cfg)
	client.DryRun = true
	client.ReleaseName = f.Spec.Chart
	client.Replace = true
	client.ClientOnly = true
	client.IncludeCRDs = true

	release, err := client.Run(chart, f.Spec.Values)
	if err != nil {
		return nil, errors.Wrap(err, "unable to run helm install action")
	}

	manifests := releaseutil.SplitManifests(release.Manifest)

	var nodes []*kyaml.RNode
	for _, manifest := range manifests {
		node, err := kyaml.Parse(manifest)
		if err != nil {
			return nodes, errors.Wrap(err, "unable to parse manifest")
		}
		nodes = append(nodes, node)
	}

	return nodes, nil
}
