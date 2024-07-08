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
	"helm.sh/helm/v3/pkg/registry"
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
	BaseDirectory      string
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
	fnlog := log.WithField("fn", f.Name())
	fnlog.Debug("rendering")

	if f.Chart == "" {
		return items, fmt.Errorf("chart cannot be empty")
	}

	tmpdir, err := os.MkdirTemp("", "konvert-helm-")
	if err != nil {
		return items, errors.Wrap(err, "unable to create temp directory for helm config and cache")
	}
	defer cleanupTmpDir(tmpdir, fnlog)

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
			return items, errors.Wrapf(err, "setting environment variable %q", k)
		}
	}

	var (
		chartURL string
		archive  string
	)

	if f.Repo == "" {
		log.WithField("base-directory", f.BaseDirectory).Debug("looking for local chart directory")
		if chartDir := resolveLocalChartDirectory(f.Chart, f.BaseDirectory); chartDir != "" {
			fnlog.WithField("chart", chartDir).Debug("using local chart directory")
			archive = chartDir
		}
	}

	if archive == "" {
		getters := getter.All(settings)
		regclient, err := getRegistryClient()
		if err != nil {
			return nil, errors.Wrap(err, "getting registry client")
		}
		c := downloader.ChartDownloader{
			Out:              os.Stderr,
			Getters:          getters,
			RepositoryConfig: settings.RepositoryConfig,
			RepositoryCache:  settings.RepositoryCache,
			RegistryClient:   regclient,
		}

		if f.Repo != "" {
			// if repo is specified, resolve url from repo and chart name
			fnlog.WithFields(
				log.Fields{
					"repo":    f.Repo,
					"chart":   f.Chart,
					"version": f.Version,
				},
			).Debug("resolving chart url from repo")
			chartURL, err = repo.FindChartInRepoURL(
				f.Repo,
				f.Chart,
				f.Version,
				"", "", "",
				getters,
			)
			if err != nil {
				return nil, errors.Wrap(err, "unable to resolve chart url")
			}
		} else {
			// otherwise, assume Chart is the full chart URL
			chartURL = f.Chart
		}

		fnlog.WithField("url", f.Chart).Debug("downloading chart from url")
		tmpDir, err := os.MkdirTemp("", "konvert")
		if err != nil {
			return nil, errors.Wrap(err, "unable to create temp directory")
		}
		defer cleanupTmpDir(tmpDir, fnlog)

		archive, _, err = c.DownloadTo(chartURL, f.Version, tmpDir)
		if err != nil {
			return nil, errors.Wrap(err, "unable to download chart")
		}
	}

	fnlog.WithField("archive", archive).Debug("loading chart")
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
		fnlog.WithFields(
			log.Fields{"kubeVersion": f.KubeVersion, "ver": ver},
		).Debug("parsed version")
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
			fnlog.WithFields(
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
			if val == konvertChartAnnotationValue(f.Repo, f.Chart) {
				continue
			}
		}
		nonChartNodes = append(nonChartNodes, item)
	}
	// append newly rendered chart nodes
	items = append(nonChartNodes, renderedNodes...)

	fnlog.Debug("done")
	return items, nil
}

func resolveLocalChartDirectory(chart string, workingDir string) string {
	if !filepath.IsAbs(chart) {
		chartDir, err := filepath.Abs(filepath.Join(workingDir, chart))
		if err != nil {
			return ""
		}
		chart = chartDir
	}
	if _, err := os.Stat(chart); err != nil {
		return ""
	}
	return chart
}

func cleanupTmpDir(tmpdir string, log *log.Entry) {
	if err := os.RemoveAll(tmpdir); err != nil {
		log.WithError(err).
			WithField("directory", tmpdir).
			Error("unable to remove temporary directory")
	}
}

func getRegistryClient() (*registry.Client, error) {
	return registry.NewClient(
		registry.ClientOptDebug(true),
		registry.ClientOptEnableCache(true),
	)
}
