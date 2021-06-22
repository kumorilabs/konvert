package konvert

import (
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"sigs.k8s.io/kustomize/kyaml/kio/filters"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

type spec struct {
	Repo    string                 `yaml:"repo,omitempty"`
	Chart   string                 `yaml:"chart,omitempty"`
	Version string                 `yaml:"version,omitempty"`
	Path    string                 `yaml:"path,omitempty"`
	Pattern string                 `yaml:"pattern,omitempty"`
	Values  map[string]interface{} `json:"values,omitempty"`
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
	// render helm chart into rnodes
	items, err := f.Render()
	if err != nil {
		return nodes, errors.Wrap(err, "unable to render chart")
	}

	// run pre-configured filters on rendered helm chart resources
	// set generated-by annotation
	// TODO: probably not useful in the end
	for _, item := range items {
		err := item.PipeE(kyaml.SetAnnotation("kumorilabs.io/generated-by", "konvert"))
		if err != nil {
			return nodes, errors.Wrap(err, "unable to run konvert filter")
		}
	}

	// set managed-by label
	for _, item := range items {
		err := item.PipeE(kyaml.SetLabel("app.kubernetes.io/managed-by", "konvert"))
		if err != nil {
			return nodes, errors.Wrap(err, "unable to run konvert filter")
		}
	}

	// set path for resources
	for _, item := range items {
		err := item.PipeE(PathAnnotation(f.Spec.Path, f.Spec.Pattern))
		if err != nil {
			return nodes, errors.Wrap(err, "unable to run konvert filter")
		}
	}

	nodes = append(nodes, items...)

	// merge duplicate nodes
	// to overwrite previously rendered helm resources
	nodes, err = filters.MergeFilter{}.Filter(nodes)
	if err != nil {
		return nodes, errors.Wrap(err, "unable to merge nodes")
	}

	return nodes, nil
}
