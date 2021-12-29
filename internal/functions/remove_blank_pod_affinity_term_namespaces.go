package functions

import (
	"fmt"

	"github.com/pkg/errors"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
	"sigs.k8s.io/yaml"
)

const (
	fnRemoveBlankPodAffinityTermNamespacesName = "remove-blank-affinities"
	fnRemoveBlankPodAffinityTermNamespacesKind = "RemoveBlankPodAffinityTermNamespaces"
)

type RemoveBlankPodAffinityTermNamespacesProcessor struct{}

func (p *RemoveBlankPodAffinityTermNamespacesProcessor) Process(resourceList *framework.ResourceList) error {
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

func (p *RemoveBlankPodAffinityTermNamespacesProcessor) run(resourceList *framework.ResourceList) error {
	var fn RemoveBlankPodAffinityTermNamespacesFunction
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

type RemoveBlankPodAffinityTermNamespacesFunction struct {
	kyaml.ResourceMeta `json:",inline" yaml:",inline"`
}

func (f *RemoveBlankPodAffinityTermNamespacesFunction) Config(rn *kyaml.RNode) error {
	switch {
	case validGVK(rn, "v1", "ConfigMap"):
	case validGVK(rn, fnConfigAPIVersion, fnRemoveBlankPodAffinityTermNamespacesKind):
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

func (f *RemoveBlankPodAffinityTermNamespacesFunction) Run(items []*kyaml.RNode) ([]*kyaml.RNode, error) {
	// TODO: refactor to a kyaml.Filter and then just FilterAll
	for _, item := range items {
		affinities := []string{"podAffinity", "podAntiAffinity"}
		for _, afftype := range affinities {
			affinity, err := item.Pipe(
				kyaml.Lookup("spec", "template", "spec", "affinity", afftype),
			)
			if err != nil {
				return items, errors.Wrap(err, "unable to lookup affinity")
			}

			if affinity != nil {
				termNodes, err := affinity.Pipe(
					kyaml.Lookup("preferredDuringSchedulingIgnoredDuringExecution"),
				)
				if err != nil {
					return items, errors.Wrap(err, "unable to lookup pod affinity term (preferred)")
				}
				terms, err := termNodes.Elements()
				if err != nil {
					return items, errors.Wrap(err, "unable to get term node elements")
				}
				for _, weightedTerm := range terms {
					term, err := weightedTerm.Pipe(kyaml.Lookup("podAffinityTerm"))
					if err != nil {
						return items, errors.Wrap(err, "unable to lookup podAffinityTerm on weighted pod affinity term")
					}
					if term == nil {
						continue
					}
					if err := removeBlankNamespacesFromPodAffnityTerm(term); err != nil {
						return items, err
					}
				}

				termNodes, err = affinity.Pipe(
					kyaml.Lookup("requiredDuringSchedulingIgnoredDuringExecution"),
				)
				if err != nil {
					return items, errors.Wrap(err, "unable to lookup pod affinity term (required)")
				}
				terms, err = termNodes.Elements()
				if err != nil {
					return items, errors.Wrap(err, "unable to get term node elements")
				}
				for _, term := range terms {
					if err := removeBlankNamespacesFromPodAffnityTerm(term); err != nil {
						return items, err
					}
				}
			}
		}
	}
	return items, nil
}

func removeBlankNamespacesFromPodAffnityTerm(term *kyaml.RNode) error {
	namespaces, err := term.Pipe(
		kyaml.Lookup("namespaces"),
	)
	if err != nil {
		return errors.Wrap(err, "unable to lookup namespaces on pod affinity term")
	}
	if namespaces != nil {
		err = namespaces.PipeE(kyaml.ElementSetter{
			Element: nil,
			Values:  []string{""},
		})
		if err != nil {
			return errors.Wrap(err, "unable to remove empty namespace from pod affinity term")
		}
		nselems, err := namespaces.Elements()
		if err != nil {
			return errors.Wrap(err, "unable to get namespaces from pod affinity term")
		}
		if len(nselems) == 0 {
			err = term.PipeE(kyaml.Clear("namespaces"))
			if err != nil {
				return errors.Wrap(err, "unable to remove namespaces from pod affinity term")
			}
		}
	}
	return nil
}
