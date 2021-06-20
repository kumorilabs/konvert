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

func (f *function) Run(items []*kyaml.RNode) ([]*kyaml.RNode, error) {
	nodes, err := f.Render()
	if err != nil {
		return items, errors.Wrap(err, "unable to render chart")
	}
	items = append(items, nodes...)

	for _, item := range items {
		err := item.PipeE(kyaml.SetAnnotation("kumorilabs.io/generated-by", "konvert"))
		if err != nil {
			return items, errors.Wrap(err, "unable to run konvert filter")
		}
	}

	for _, item := range items {
		err := item.PipeE(kyaml.SetLabel("app.kubernetes.io/managed-by", "konvert"))
		if err != nil {
			return items, errors.Wrap(err, "unable to run konvert filter")
		}
	}

	items, err = filters.MergeFilter{}.Filter(items)
	if err != nil {
		return items, errors.Wrap(err, "unable to merge items")
	}
	return items, nil
}
