package konvert

import (
	"fmt"

	"github.com/kumorilabs/konvert/internal/functions"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

// TODO: this is just another function

const (
	fnConfigGroup          = "konvert.kumorilabs.io"
	annotationKonvertChart = fnConfigGroup + "/chart"
)

type spec struct {
	Repo      string                 `yaml:"repo,omitempty"`
	Chart     string                 `yaml:"chart,omitempty"`
	Version   string                 `yaml:"version,omitempty"`
	Namespace string                 `yaml:"namespace,omitempty"`
	Path      string                 `yaml:"path,omitempty"`
	Pattern   string                 `yaml:"pattern,omitempty"`
	Kustomize bool                   `yaml:"kustomize,omitempty"`
	Values    map[string]interface{} `json:"values,omitempty"`
}

type function struct {
	kyaml.ResourceMeta `json:",inline" yaml:",inline"`
	Spec               spec `yaml:"spec,omitempty"`
}

func (f *function) Config(rn *kyaml.RNode) error {
	yamlstr, err := rn.String()
	if err != nil {
		return errors.Wrap(err, "unable to get yaml from rnode")
	}
	if err := yaml.Unmarshal([]byte(yamlstr), f); err != nil {
		return errors.Wrap(err, "unable to unmarshal konvert config")
	}
	return nil
}

func (f *function) Run(nodes []*kyaml.RNode) ([]*kyaml.RNode, error) {
	// for each chart instance (repo, version, release?):
	//   remove previously rendered chart nodes
	//   render chart nodes
	//   run functions against rendered chart nodes
	//   add rendered chart nodes

	annotationKonvertChartValue := fmt.Sprintf("%s,%s", f.Spec.Repo, f.Spec.Chart)

	removeByAnnotations := functions.RemoveByAnnotationsFunction{
		Annotations: map[string]string{
			annotationKonvertChart: annotationKonvertChartValue,
		},
	}

	nodes, err := removeByAnnotations.Run(nodes)
	if err != nil {
		return nodes, errors.Wrap(err, "unable to run remove-by-annotations function")
	}

	runKonvert := func() ([]*kyaml.RNode, error) {
		var items []*kyaml.RNode
		renderHelmChart := functions.RenderHelmChartFunction{
			Repo:      f.Spec.Repo,
			Chart:     f.Spec.Chart,
			Version:   f.Spec.Version,
			Values:    f.Spec.Values,
			Namespace: f.Spec.Namespace,
		}
		items, err := renderHelmChart.Run(items)
		if err != nil {
			return items, err
		}

		// run pre-configured functions on rendered helm chart resources

		removeBlankNamespace := functions.RemoveBlankNamespaceFunction{}
		items, err = removeBlankNamespace.Run(items)
		if err != nil {
			return items, errors.Wrap(err, "unable to run remove-blank-namespace function")
		}

		setManagedBy := functions.SetManagedByFunction{}
		items, err = setManagedBy.Run(items)
		if err != nil {
			return items, errors.Wrap(err, "unable to run managed-by function")
		}

		setKonvertAnnotations := functions.SetKonvertAnnotationsFunction{
			Repo:  f.Spec.Repo,
			Chart: f.Spec.Chart,
		}
		items, err = setKonvertAnnotations.Run(items)
		if err != nil {
			return items, errors.Wrap(err, "unable to run konvert-annotations function")
		}

		fixNullNodePorts := functions.FixNullNodePortsFunction{}
		items, err = fixNullNodePorts.Run(items)
		if err != nil {
			return items, errors.Wrap(err, "unable to run fix-null-node-ports function")
		}

		removeBlankAffinities := functions.RemoveBlankAffinitiesFunction{}
		items, err = removeBlankAffinities.Run(items)
		if err != nil {
			return items, errors.Wrap(err, "unable to run remove-blank-affinities function")
		}

		removeBlankPodAffinityTermNamespaces := functions.RemoveBlankPodAffinityTermNamespacesFunction{}
		items, err = removeBlankPodAffinityTermNamespaces.Run(items)
		if err != nil {
			return items, errors.Wrap(err, "unable to run remove-blank-pod-affinity-term-namespaces function")
		}

		setPathAnnotation := functions.SetPathAnnotationFunction{
			Path:    f.Spec.Path,
			Pattern: f.Spec.Pattern,
		}
		items, err = setPathAnnotation.Run(items)
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

	if f.Spec.Kustomize {
		kustomizer := functions.KustomizerFunction{
			Path:                    f.Spec.Path,
			Namespace:               f.Spec.Namespace,
			ResourceAnnotationName:  annotationKonvertChart,
			ResourceAnnotationValue: annotationKonvertChartValue,
		}
		nodes, err = kustomizer.Run(nodes)
		if err != nil {
			return nodes, errors.Wrap(err, "unable to run kustomizer function")
		}
	}

	return nodes, nil
}
