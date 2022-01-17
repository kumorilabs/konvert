package functions

import (
	"fmt"

	"github.com/pkg/errors"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

const (
	fnKonvertName = "konvert"
	fnKonvertKind = "Konvert"
)

type KonvertProcessor struct{}

func (p *KonvertProcessor) Process(resourceList *framework.ResourceList) error {
	// if a function config is not provided by the framework,
	// look for one in the input items
	// this will only work for the Konvert kind, not ConfigMaps
	if resourceList.FunctionConfig == nil {
		for _, item := range resourceList.Items {
			if item.GetKind() == fnKonvertKind && item.GetApiVersion() == fnConfigAPIVersion {
				resourceList.FunctionConfig = item
				break
			}
		}
	}
	return runFn(&KonvertFunction{}, resourceList)
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
}

func (f *KonvertFunction) Name() string {
	return fnKonvertName
}

func (f *KonvertFunction) SetResourceMeta(meta kyaml.ResourceMeta) {
	f.ResourceMeta = meta
}

func (f *KonvertFunction) Config(rn *kyaml.RNode) error {
	return loadConfig(f, rn, fnKonvertKind)
}

func (f *KonvertFunction) Filter(nodes []*kyaml.RNode) ([]*kyaml.RNode, error) {
	// for each chart instance (repo, version, release?):
	//   remove previously rendered chart nodes
	//   render chart nodes
	//   run functions against rendered chart nodes
	//   add rendered chart nodes

	annotationKonvertChartValue := fmt.Sprintf("%s,%s", f.Repo, f.Chart)

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
			ReleaseName: f.ResourceMeta.Name,
			Repo:        f.Repo,
			Chart:       f.Chart,
			Version:     f.Version,
			Values:      f.Values,
			Namespace:   f.Namespace,
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
