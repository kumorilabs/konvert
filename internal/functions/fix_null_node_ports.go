package functions

import (
	"fmt"

	"github.com/pkg/errors"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
	"sigs.k8s.io/yaml"
)

// hack to remove `nodePort: null` in service ports
// see https://github.com/GoogleContainerTools/kpt/issues/2321

const (
	fnFixNullNodePortsName = "fix-null-node-ports"
	fnFixNullNodePortsKind = "FixNullNodePorts"
)

type FixNullNodePortsProcessor struct{}

func (p *FixNullNodePortsProcessor) Process(resourceList *framework.ResourceList) error {
	err := p.run(resourceList)
	if err != nil {
		result := &framework.Result{
			Message:  fmt.Sprintf("error running %s: %v", fnFixNullNodePortsName, err.Error()),
			Severity: framework.Error,
		}
		resourceList.Results = append(resourceList.Results, result)
	}
	return err
}

func (p *FixNullNodePortsProcessor) run(resourceList *framework.ResourceList) error {
	var fn FixNullNodePortsFunction
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

type FixNullNodePortsFunction struct {
	kyaml.ResourceMeta `json:",inline" yaml:",inline"`
}

func (f *FixNullNodePortsFunction) Config(rn *kyaml.RNode) error {
	switch {
	case validGVK(rn, "v1", "ConfigMap"):
	case validGVK(rn, fnConfigAPIVersion, fnFixNullNodePortsKind):
		yamlstr, err := rn.String()
		if err != nil {
			return errors.Wrap(err, "unable to get yaml from rnode")
		}
		if err := yaml.Unmarshal([]byte(yamlstr), f); err != nil {
			return errors.Wrap(err, "unable to unmarshal function config")
		}
	default:
		return fmt.Errorf("`functionConfig` must be a `ConfigMap` or `%s`", fnFixNullNodePortsKind)
	}

	return nil
}

func (f *FixNullNodePortsFunction) Run(items []*kyaml.RNode) ([]*kyaml.RNode, error) {
	for _, item := range items {
		if item.GetKind() == "Service" {
			err := item.PipeE(
				kyaml.Lookup("spec", "ports", "[nodePort=null]"),
				kyaml.Clear("nodePort"),
			)
			if err != nil {
				return items, errors.Wrap(err, "unable to run remove null nodePort from Service")
			}
		}
	}
	return items, nil
}
