package functions

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
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
	"sigs.k8s.io/yaml"
)

const (
	fnRenderHelmChartName = "render-helm-chart"
	fnRenderHelmChartKind = "RenderHelmChart"
)

type RenderHelmChartProcessor struct{}

func (p *RenderHelmChartProcessor) Process(resourceList *framework.ResourceList) error {
	err := p.run(resourceList)
	if err != nil {
		result := &framework.Result{
			Message:  fmt.Sprintf("error running %s: %v", fnRenderHelmChartName, err.Error()),
			Severity: framework.Error,
		}
		resourceList.Results = append(resourceList.Results, result)
	}
	return err
}

func (p *RenderHelmChartProcessor) run(resourceList *framework.ResourceList) error {
	var fn RenderHelmChartFunction
	err := fn.Config(resourceList.FunctionConfig)
	if err != nil {
		return errors.Wrap(err, "failed to configure function")
	}
	resourceList.Items, err = fn.Run(resourceList.Items)
	if err != nil {
		return errors.Wrap(err, "failed to run function")
	}
	return nil
}

type RenderHelmChartFunction struct {
	kyaml.ResourceMeta `json:",inline" yaml:",inline"`
	Repo               string                 `json:"repo,omitempty" yaml:"repo,omitempty"`
	Chart              string                 `json:"chart,omitempty" yaml:"chart,omitempty"`
	Version            string                 `json:"version,omitempty" yaml:"version,omitempty"`
	Values             map[string]interface{} `json:"values,omitempty" yaml:"values,omitempty"`
	Namespace          string                 `json:"namespace,omitempty" yaml:"namespace,omitempty"`
}

func (f *RenderHelmChartFunction) Config(rn *kyaml.RNode) error {
	switch {
	case validGVK(rn, "v1", "ConfigMap"):
		f.Namespace = rn.GetDataMap()["namespace"]
	case validGVK(rn, fnConfigAPIVersion, fnRenderHelmChartKind):
		yamlstr, err := rn.String()
		if err != nil {
			return errors.Wrap(err, "unable to get yaml from rnode")
		}
		if err := yaml.Unmarshal([]byte(yamlstr), f); err != nil {
			return errors.Wrap(err, "unable to unmarshal function config")
		}
	default:
		return fmt.Errorf("`functionConfig` must be a `ConfigMap` or `%s`", fnRenderHelmChartKind)
	}

	return nil
}

func (f *RenderHelmChartFunction) Run(items []*kyaml.RNode) ([]*kyaml.RNode, error) {
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

	chartURL, err := repo.FindChartInRepoURL(
		f.Repo,
		f.Chart,
		f.Version,
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
	client.ReleaseName = f.Chart
	client.Replace = true
	client.ClientOnly = true
	client.IncludeCRDs = true
	if f.Namespace != "" {
		client.Namespace = f.Namespace
	}

	release, err := client.Run(chart, f.Values)
	if err != nil {
		return nil, errors.Wrap(err, "unable to run helm install action")
	}

	manifests := releaseutil.SplitManifests(release.Manifest)

	var renderedNodes []*kyaml.RNode
	for _, manifest := range manifests {
		node, err := kyaml.Parse(manifest)
		if err != nil {
			return renderedNodes, errors.Wrap(err, "unable to parse manifest")
		}
		renderedNodes = append(renderedNodes, node)
	}

	// remove previously rendered chart nodes (preserve nodes not generated from
	// this chart)
	var nonChartNodes []*kyaml.RNode
	for _, item := range items {
		// TODO: don't hardcode annotation here
		if val, ok := item.GetAnnotations()["konvert.kumorilabs.io/chart"]; ok {
			if val == fmt.Sprintf("%s,%s", f.Repo, f.Chart) {
				continue
			}
		}
		nonChartNodes = append(nonChartNodes, item)
	}
	// append newly rendered chart nodes
	items = append(nonChartNodes, renderedNodes...)

	return items, nil
}
