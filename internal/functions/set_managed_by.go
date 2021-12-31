package functions

import (
	"fmt"

	"github.com/pkg/errors"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/kio"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
	"sigs.k8s.io/yaml"
)

const (
	fnSetManagedByName = "managed-by"
	fnSetManagedByKind = "SetManagedBy"
	defaultManagedBy   = "konvert"
)

type SetManagedByProcessor struct{}

func (p *SetManagedByProcessor) Process(resourceList *framework.ResourceList) error {
	err := p.run(resourceList)
	if err != nil {
		result := &framework.Result{
			Message:  fmt.Sprintf("error running %s: %v", fnSetManagedByName, err.Error()),
			Severity: framework.Error,
		}
		resourceList.Results = append(resourceList.Results, result)
	}
	return err
}

func (p *SetManagedByProcessor) run(resourceList *framework.ResourceList) error {
	var fn SetManagedByFunction
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

type SetManagedByFunction struct {
	kyaml.ResourceMeta `json:",inline" yaml:",inline"`
	Value              string `json:"value,omitempty" yaml:"value,omitempty"`
}

func (f *SetManagedByFunction) Config(rn *kyaml.RNode) error {
	switch {
	case validGVK(rn, "v1", "ConfigMap"):
		f.Value = rn.GetDataMap()["value"]
	case validGVK(rn, fnConfigAPIVersion, fnSetManagedByKind):
		yamlstr, err := rn.String()
		if err != nil {
			return errors.Wrap(err, "unable to get yaml from rnode")
		}
		if err := yaml.Unmarshal([]byte(yamlstr), f); err != nil {
			return errors.Wrap(err, "unable to unmarshal function config")
		}
	default:
		return fmt.Errorf("`functionConfig` must be a `ConfigMap` or `%s`", fnSetManagedByKind)
	}

	return nil
}

func (f *SetManagedByFunction) Run(items []*kyaml.RNode) ([]*kyaml.RNode, error) {
	if f.Value == "" {
		f.Value = defaultManagedBy
	}

	return kio.FilterAll(
		kyaml.SetLabel("app.kubernetes.io/managed-by", f.Value),
	).Filter(items)
}
