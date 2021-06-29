package functions

import (
	"fmt"

	"github.com/pkg/errors"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
	"sigs.k8s.io/yaml"
)

const (
	fnRemoveBlankNamespaceName = "remove-blank-namespace"
	fnRemoveBlankNamespaceKind = "RemoveBlankNamespace"
)

type RemoveBlankNamespaceProcessor struct{}

func (p *RemoveBlankNamespaceProcessor) Process(resourceList *framework.ResourceList) error {
	err := p.run(resourceList)
	if err != nil {
		resourceList.Result = &framework.Result{
			Name: fnRemoveBlankNamespaceName,
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

func (p *RemoveBlankNamespaceProcessor) run(resourceList *framework.ResourceList) error {
	var fn RemoveBlankNamespaceFunction
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

type RemoveBlankNamespaceFunction struct {
	kyaml.ResourceMeta `json:",inline" yaml:",inline"`
}

func (f *RemoveBlankNamespaceFunction) Config(rn *kyaml.RNode) error {
	switch {
	case validGVK(rn, "v1", "ConfigMap"):
	case validGVK(rn, fnConfigAPIVersion, fnRemoveBlankNamespaceKind):
		yamlstr, err := rn.String()
		if err != nil {
			return errors.Wrap(err, "unable to get yaml from rnode")
		}
		if err := yaml.Unmarshal([]byte(yamlstr), f); err != nil {
			return errors.Wrap(err, "unable to unmarshal function config")
		}
	default:
		return fmt.Errorf("`functionConfig` must be a `ConfigMap` or `%s`", fnRemoveBlankNamespaceKind)
	}

	return nil
}

func (f *RemoveBlankNamespaceFunction) Run(items []*kyaml.RNode) ([]*kyaml.RNode, error) {
	for _, item := range items {
		mdnode, err := item.Pipe(kyaml.Lookup("metadata"))
		if err != nil {
			return items, errors.Wrap(err, "unable to get metadata rnode from rnode")
		}
		if item.GetNamespace() == "" {
			err := mdnode.PipeE(kyaml.Clear("namespace"))
			if err != nil {
				return items, errors.Wrap(err, "unable to clear namespace")
			}
		}
	}
	return items, nil
}
