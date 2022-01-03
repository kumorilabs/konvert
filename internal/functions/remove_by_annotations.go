package functions

import (
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

const (
	fnRemoveByAnnotationsName = "remove-by-annotation"
	fnRemoveByAnnotationsKind = "RemoveByAnnotations"
)

type RemoveByAnnotationsProcessor struct{}

func (p *RemoveByAnnotationsProcessor) Process(resourceList *framework.ResourceList) error {
	return runFn(&RemoveByAnnotationsFunction{}, resourceList)
}

type RemoveByAnnotationsFunction struct {
	kyaml.ResourceMeta `json:",inline" yaml:",inline"`
	Annotations        map[string]string `json:"annotations,omitempty" yaml:"annotations,omitempty"`
}

func (f *RemoveByAnnotationsFunction) Name() string {
	return fnRemoveByAnnotationsName
}

func (f *RemoveByAnnotationsFunction) Config(rn *kyaml.RNode) error {
	return loadConfig(f, rn, fnRemoveByAnnotationsKind)
}

// any resources that have a matching annotation (k=v) will be removed
func (f *RemoveByAnnotationsFunction) Filter(items []*kyaml.RNode) ([]*kyaml.RNode, error) {
	if len(f.Annotations) == 0 {
		return items, nil
	}

	var filtered []*kyaml.RNode
	for _, item := range items {
		var matched bool
		for key, val := range f.Annotations {
			if v, ok := item.GetAnnotations()[key]; ok {
				matched = v == val
			}
			if matched {
				break
			}
		}
		if matched {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered, nil
}
