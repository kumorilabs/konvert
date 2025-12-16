package functions

import (
	"path/filepath"

	"github.com/kumorilabs/konvert/internal/kube"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

const (
	fnKonvertName = "konvert"
	fnKonvertKind = "Konvert"
)

type KonvertProcessor struct{}

func IsKonvertFile(item *kyaml.RNode) bool {
	return item.GetKind() == fnKonvertKind && item.GetApiVersion() == fnConfigAPIVersion
}

func (p *KonvertProcessor) Process(resourceList *framework.ResourceList) error {
	fnconfigs := p.functionConfigs(resourceList)
	for _, fnconfig := range fnconfigs {
		resourceList.FunctionConfig = fnconfig
		err := runFn(&KonvertFunction{}, resourceList)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *KonvertProcessor) functionConfigs(resourceList *framework.ResourceList) []*kyaml.RNode {
	// if a function config is not provided by the framework,
	// look for them in the input items
	// this will only work for the Konvert kind, not ConfigMaps
	// if a function config is provided by the framework AND one or more
	// function configs are in the input items, we still only process the
	// fnconfig provided by the framework b/c we are assuming the consumer
	// intentionally wants to only process the specified function config
	if resourceList.FunctionConfig != nil {
		return []*kyaml.RNode{resourceList.FunctionConfig}
	}

	var fnconfigs []*kyaml.RNode
	for _, item := range resourceList.Items {
		if IsKonvertFile(item) {
			fnconfigs = append(fnconfigs, item)
		}
	}
	return fnconfigs
}

func Konvert(filePath string) *KonvertFunction {
	return &KonvertFunction{
		filePath: filePath,
	}
}

type KonvertFunction struct {
	kyaml.ResourceMeta `json:",inline" yaml:",inline"`
	Repo               string                 `yaml:"repo,omitempty"`
	Chart              string                 `yaml:"chart,omitempty"`
	Version            string                 `yaml:"version,omitempty"`
	Namespace          string                 `yaml:"namespace,omitempty"`
	Path               string                 `yaml:"path,omitempty"`
	Pattern            string                 `yaml:"pattern,omitempty"`
	Kustomize          bool                   `yaml:"kustomize,omitempty"`
	Values             map[string]interface{} `json:"values,omitempty"`
	SkipHooks          bool                   `json:"skipHooks,omitempty" yaml:"skipHooks,omityempty"`
	SkipTests          bool                   `json:"skipTests,omitempty" yaml:"skipTests,omityempty"`
	SkipCRDs           bool                   `json:"skipCRDs,omitempty" yaml:"skipCRDs,omitempty"`
	KubeVersion        string                 `json:"kubeVersion,omitempty" yaml:"kubeVersion,omitempty"`
	filePath           string
}

func (f *KonvertFunction) Name() string {
	return fnKonvertName
}

func (f *KonvertFunction) SetResourceMeta(meta kyaml.ResourceMeta) {
	f.ResourceMeta = meta
}

func (f *KonvertFunction) Config(rn *kyaml.RNode) error {
	fnlog := log.WithField("fn", f.Name())
	err := loadConfig(f, rn, fnKonvertKind)
	if err != nil {
		return err
	}

	fnconfigPath := rn.GetAnnotations()[kioutil.PathAnnotation]
	baseDir := filepath.Dir(fnconfigPath)

	if !isDefaultPath(f.Path) {
		f.Path = filepath.Join(baseDir, f.Path)
	}

	fnlog.WithFields(log.Fields{
		"fnconfig-path": fnconfigPath,
		"filePath":      f.filePath,
		"path":          f.Path,
	}).Debug("configuring function")

	if f.KubeVersion == "" {
		kubeVersion, err := kube.TryDiscoverKubeVersion()
		if err != nil {
			fnlog.WithError(err).Debug("unable to discover kubernetes version")
		}
		fnlog.WithField("kubernetes-version", kubeVersion).Debug("discovered kubernetes version")
		if kubeVersion != "" {
			f.KubeVersion = kubeVersion
		}
	}

	return nil
}

func (f *KonvertFunction) Filter(nodes []*kyaml.RNode) ([]*kyaml.RNode, error) {
	log := log.WithFields(
		log.Fields{"fn": f.Name(), "path": f.filePath},
	)
	// for each chart instance (repo, version, release?):
	//   remove previously rendered chart nodes
	//   render chart nodes
	//   run functions against rendered chart nodes
	//   add rendered chart nodes
	log.Debug("running")

	annotationKonvertChartValue := konvertChartAnnotationValue(f.Repo, f.Chart)

	removeByAnnotations := RemoveByAnnotationsFunction{
		Annotations: map[string]string{
			annotationKonvertChart: annotationKonvertChartValue,
		},
	}

	nodes, err := removeByAnnotations.Filter(nodes)
	if err != nil {
		return nodes, errors.Wrap(err, "unable to run remove-by-annotations function")
	}

	runKonvert := func() ([]*kyaml.RNode, error) {
		var items []*kyaml.RNode
		renderHelmChart := RenderHelmChartFunction{
			ReleaseName:   f.ResourceMeta.Name,
			Repo:          f.Repo,
			Chart:         f.Chart,
			Version:       f.Version,
			KubeVersion:   f.KubeVersion,
			Values:        f.Values,
			Namespace:     f.Namespace,
			SkipHooks:     f.SkipHooks,
			SkipTests:     f.SkipTests,
			SkipCRDs:      f.SkipCRDs,
			BaseDirectory: filepath.Dir(f.filePath),
		}
		items, err := renderHelmChart.Filter(items)
		if err != nil {
			return items, err
		}

		// run pre-configured functions on rendered helm chart resources

		removeBlankNamespace := RemoveBlankNamespaceFunction{}
		items, err = removeBlankNamespace.Filter(items)
		if err != nil {
			return items, errors.Wrap(err, "unable to run remove-blank-namespace function")
		}

		setManagedBy := SetManagedByFunction{}
		items, err = setManagedBy.Filter(items)
		if err != nil {
			return items, errors.Wrap(err, "unable to run managed-by function")
		}

		setKonvertAnnotations := SetKonvertAnnotationsFunction{
			Repo:  f.Repo,
			Chart: f.Chart,
		}
		items, err = setKonvertAnnotations.Filter(items)
		if err != nil {
			return items, errors.Wrap(err, "unable to run konvert-annotations function")
		}

		fixNullNodePorts := FixNullNodePortsFunction{}
		items, err = fixNullNodePorts.Filter(items)
		if err != nil {
			return items, errors.Wrap(err, "unable to run fix-null-node-ports function")
		}

		removeBlankAffinities := RemoveBlankAffinitiesFunction{}
		items, err = removeBlankAffinities.Filter(items)
		if err != nil {
			return items, errors.Wrap(err, "unable to run remove-blank-affinities function")
		}

		removeBlankPodAffinityTermNamespaces := RemoveBlankPodAffinityTermNamespacesFunction{}
		items, err = removeBlankPodAffinityTermNamespaces.Filter(items)
		if err != nil {
			return items, errors.Wrap(err, "unable to run remove-blank-pod-affinity-term-namespaces function")
		}

		setPathAnnotation := SetPathAnnotationFunction{
			Path:    f.Path,
			Pattern: f.Pattern,
		}
		items, err = setPathAnnotation.Filter(items)
		if err != nil {
			return items, errors.Wrap(err, "unable to run path-annotation function")
		}

		return items, nil
	}

	items, err := runKonvert()
	if err != nil {
		return nodes, err
	}

	// append newly rendered chart nodes
	nodes = append(nodes, items...)

	if f.Kustomize {
		kustomizer := KustomizerFunction{
			Path:                    f.Path,
			Namespace:               f.Namespace,
			ResourceAnnotationName:  annotationKonvertChart,
			ResourceAnnotationValue: annotationKonvertChartValue,
		}
		nodes, err = kustomizer.Filter(nodes)
		if err != nil {
			return nodes, errors.Wrap(err, "unable to run kustomizer function")
		}
	}

	return nodes, nil
}
