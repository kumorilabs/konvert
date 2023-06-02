package functions

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"
	helmrelease "helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/releaseutil"
	"helm.sh/helm/v3/pkg/repo"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

const (
	fnRenderHelmChartName = "render-helm-chart"
	fnRenderHelmChartKind = "RenderHelmChart"
)

type RenderHelmChartProcessor struct{}

func (p *RenderHelmChartProcessor) Process(resourceList *framework.ResourceList) error {
	return runFn(&RenderHelmChartFunction{}, resourceList)
}

type RenderHelmChartFunction struct {
	kyaml.ResourceMeta `json:",inline" yaml:",inline"`
	ReleaseName        string                 `json:"releaseName,omitempty" yaml:"releaseName,omitempty"`
	Repo               string                 `json:"repo,omitempty" yaml:"repo,omitempty"`
	Chart              string                 `json:"chart,omitempty" yaml:"chart,omitempty"`
	Version            string                 `json:"version,omitempty" yaml:"version,omitempty"`
	Values             map[string]interface{} `json:"values,omitempty" yaml:"values,omitempty"`
	Namespace          string                 `json:"namespace,omitempty" yaml:"namespace,omitempty"`
	SkipHooks          bool                   `json:"skipHooks,omitempty" yaml:"skipHooks,omityempty"`
	SkipTests          bool                   `json:"skipTests,omitempty" yaml:"skipTests,omityempty"`
	KubeVersion        string                 `json:"kubeVersion,omitempty" yaml:"kubeVersion,omitempty"`
}

func (f *RenderHelmChartFunction) Name() string {
	return fnRenderHelmChartName
}

func (f *RenderHelmChartFunction) SetResourceMeta(meta kyaml.ResourceMeta) {
	f.ResourceMeta = meta
}

func (f *RenderHelmChartFunction) Config(rn *kyaml.RNode) error {
	return loadConfig(f, rn, fnRenderHelmChartKind)
}

func (f *RenderHelmChartFunction) Filter(items []*kyaml.RNode) ([]*kyaml.RNode, error) {
	if f.Repo == "" {
		return items, fmt.Errorf("repo cannot be empty")
	}
	if f.Chart == "" {
		return items, fmt.Errorf("chart cannot be empty")
	}
	if f.Version == "" {
		return items, fmt.Errorf("version cannot be empty")
	}

	tmpdir, err := os.MkdirTemp("", "konvert-helm-")
	if err != nil {
		return items, errors.Wrap(err, "unable to create temp directory for helm config and cache")
	}
	configPath := filepath.Join(tmpdir, ".config")
	cachePath := filepath.Join(tmpdir, ".cache")
	dataPath := filepath.Join(tmpdir, ".data")

	settings := cli.New()
	settings.RegistryConfig = filepath.Join(configPath, "registry.json")
	settings.RepositoryConfig = filepath.Join(configPath, "repositories.yaml")
	settings.RepositoryCache = filepath.Join(cachePath, "repository")

	for k, v := range map[string]string{
		"HELM_CONFIG_HOME": configPath,
		"HELM_CACHE_HOME":  cachePath,
		"HELM_DATA_HOME":   dataPath,
	} {
		if err := os.Setenv(k, v); err != nil {
			return items, errors.Wrapf(err, "unable to set environment variable %q", k)
		}
	}

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

	tmpDir, err := os.MkdirTemp("", "konvert")
	if err != nil {
		return nil, errors.Wrap(err, "unable to create temp directory")
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			log.WithError(err).
				WithField("directory", tmpDir).
				Error("unable to remove temporary directory")
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

	releaseName := f.ReleaseName
	if releaseName == "" {
		releaseName = f.Chart
	}

	cfg := new(action.Configuration)
	client := action.NewInstall(cfg)
	client.DryRun = true
	client.ReleaseName = releaseName
	client.Replace = true
	client.ClientOnly = true
	client.IncludeCRDs = true
	if f.Namespace != "" {
		client.Namespace = f.Namespace
	}
	if f.KubeVersion != "" {
		ver, err := chartutil.ParseKubeVersion(f.KubeVersion)
		if err != nil {
			return nil, errors.Wrap(err, "unable to parse kubeVersion")
		}
		log.WithFields(
			log.Fields{"kubeVersion": f.KubeVersion, "ver": ver},
		).Info("parsed version")
		client.KubeVersion = ver
	}

	release, err := client.Run(chart, f.Values)
	if err != nil {
		return nil, errors.Wrap(err, "unable to run helm install action")
	}

	manifests := releaseutil.SplitManifests(release.Manifest)

	if !f.SkipHooks {
		isTestHook := func(h *helmrelease.Hook) bool {
			for _, e := range h.Events {
				if e == helmrelease.HookTest {
					return true
				}
			}
			return false
		}

		for _, hook := range release.Hooks {
			if f.SkipTests && isTestHook(hook) {
				continue
			}
			log.WithFields(
				log.Fields{"kind": hook.Kind, "name": hook.Name, "path": hook.Path, "weight": hook.Weight},
			).Debug("adding hook")
			manifests[hook.Path] = fmt.Sprintf("---\n# Source: %s\n%s\n", hook.Path, hook.Manifest)
		}
	}

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
		if val, ok := item.GetAnnotations()[annotationKonvertChart]; ok {
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
