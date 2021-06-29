package functions

import (
	"fmt"

	"github.com/pkg/errors"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
	"sigs.k8s.io/yaml"
)

const (
	fnRemoveBlankAffinitiesName = "remove-blank-affinities"
	fnRemoveBlankAffinitiesKind = "RemoveBlankAffinities"
)

type RemoveBlankAffinitiesProcessor struct{}

func (p *RemoveBlankAffinitiesProcessor) Process(resourceList *framework.ResourceList) error {
	err := p.run(resourceList)
	if err != nil {
		resourceList.Result = &framework.Result{
			Name: fnSetKonvertAnnotationsName,
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

func (p *RemoveBlankAffinitiesProcessor) run(resourceList *framework.ResourceList) error {
	var fn RemoveBlankAffinitiesFunction
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

type RemoveBlankAffinitiesFunction struct {
	kyaml.ResourceMeta `json:",inline" yaml:",inline"`
}

func (f *RemoveBlankAffinitiesFunction) Config(rn *kyaml.RNode) error {
	switch {
	case validGVK(rn, "v1", "ConfigMap"):
	case validGVK(rn, fnConfigAPIVersion, fnRemoveBlankAffinitiesKind):
		yamlstr, err := rn.String()
		if err != nil {
			return errors.Wrap(err, "unable to get yaml from rnode")
		}
		if err := yaml.Unmarshal([]byte(yamlstr), f); err != nil {
			return errors.Wrap(err, "unable to unmarshal function config")
		}
	default:
		return fmt.Errorf("`functionConfig` must be a `ConfigMap` or `%s`", fnRemoveBlankAffinitiesKind)
	}

	return nil
}

func (f *RemoveBlankAffinitiesFunction) Run(items []*kyaml.RNode) ([]*kyaml.RNode, error) {
	// TODO: refactor to a kyaml.Filter and then just FilterAll
	for _, item := range items {
		affinity, err := item.Pipe(
			kyaml.Lookup("spec", "template", "spec", "affinity"),
		)
		if err != nil {
			return items, errors.Wrap(err, "unable to lookup affinity")
		}

		if affinity != nil {
			affinities := []string{"podAffinity", "nodeAffinity", "podAntiAffinity"}
			for _, affinityField := range affinities {
				field := affinity.Field(affinityField)
				if field.IsNilOrEmpty() {
					err = affinity.PipeE(kyaml.Clear(affinityField))
					if err != nil {
						return items, errors.Wrapf(err, "unable to clear field %q", affinityField)
					}
				}
			}
		}
	}
	return items, nil
}
