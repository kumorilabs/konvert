package functions

import (
	"fmt"

	"github.com/pkg/errors"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
	"sigs.k8s.io/yaml"
)

const (
	fnRemoveByAnnotationsName = "remove-by-annotation"
	fnRemoveByAnnotationsKind = "RemoveByAnnotations"
)

type RemoveByAnnotationsProcessor struct{}

func (p *RemoveByAnnotationsProcessor) Process(resourceList *framework.ResourceList) error {
	err := p.run(resourceList)
	if err != nil {
		resourceList.Result = &framework.Result{
			Name: fnRemoveByAnnotationsName,
			Items: []framework.ResultItem{
				{
					Message:  err.Error(),
					Severity: framework.Error,
				},
			},
		}
	}
	return err
}

func (p *RemoveByAnnotationsProcessor) run(resourceList *framework.ResourceList) error {
	var fn RemoveByAnnotationsFunction
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

type RemoveByAnnotationsFunction struct {
	kyaml.ResourceMeta `json:",inline" yaml:",inline"`
	Annotations        map[string]string `json:"annotations,omitempty" yaml:"annotations,omitempty"`
}

func (f *RemoveByAnnotationsFunction) Config(rn *kyaml.RNode) error {
	switch {
	case validGVK(rn, "v1", "ConfigMap"):
		f.Annotations = rn.GetDataMap()
	case validGVK(rn, fnConfigAPIVersion, fnRemoveByAnnotationsKind):
		yamlstr, err := rn.String()
		if err != nil {
			return errors.Wrap(err, "unable to get yaml from rnode")
		}
		if err := yaml.Unmarshal([]byte(yamlstr), f); err != nil {
			return errors.Wrap(err, "unable to unmarshal function config")
		}
	default:
		return fmt.Errorf("`functionConfig` must be a `ConfigMap` or `%s`", fnRemoveByAnnotationsKind)
	}

	return nil
}

func (f *RemoveByAnnotationsFunction) Run(items []*kyaml.RNode) ([]*kyaml.RNode, error) {
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
			if !matched {
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
