package konvert

import (
	"fmt"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"sigs.k8s.io/kustomize/kyaml/kio/filters"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

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
	for _, item := range items {
		err := item.PipeE(kyaml.SetAnnotation("managed-by", "konvert"))
		if err != nil {
			return items, errors.Wrap(err, "unable to run konvert filter")
		}
	}
	items = append(items, mustServiceAccount("testing"))

	items, err := filters.MergeFilter{}.Filter(items)
	if err != nil {
		return items, errors.Wrap(err, "unable to merge items")
	}
	return items, nil
}

type spec struct {
	Repo    string                 `yaml:"repo,omitempty"`
	Chart   string                 `yaml:"chart,omitempty"`
	Version string                 `yaml:"version,omitempty"`
	Values  map[string]interface{} `json:"values,omitempty"`
}

func mustServiceAccount(name string) *kyaml.RNode {
	a := `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: %s
  annotations:
    config.kubernetes.io/path: base/serviceaccount-%s.yaml
`
	yamlstr := fmt.Sprintf(a, name, name)
	return kyaml.MustParse(yamlstr)
}
