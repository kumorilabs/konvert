package functions

import (
	"github.com/pkg/errors"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/kio"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

const (
	fnRemoveBlankPodAffinityTermNamespacesName = "remove-blank-affinities"
	fnRemoveBlankPodAffinityTermNamespacesKind = "RemoveBlankPodAffinityTermNamespaces"
)

type RemoveBlankPodAffinityTermNamespacesProcessor struct{}

func (p *RemoveBlankPodAffinityTermNamespacesProcessor) Process(resourceList *framework.ResourceList) error {
	return runFn(&RemoveBlankPodAffinityTermNamespacesFunction{}, resourceList)
}

type RemoveBlankPodAffinityTermNamespacesFunction struct {
	kyaml.ResourceMeta `json:",inline" yaml:",inline"`
}

func (f *RemoveBlankPodAffinityTermNamespacesFunction) Name() string {
	return fnRemoveBlankPodAffinityTermNamespacesName
}

func (f *RemoveBlankPodAffinityTermNamespacesFunction) Config(rn *kyaml.RNode) error {
	return loadConfig(f, rn, fnRemoveBlankPodAffinityTermNamespacesKind)
}

func (f *RemoveBlankPodAffinityTermNamespacesFunction) Filter(items []*kyaml.RNode) ([]*kyaml.RNode, error) {
	return kio.FilterAll(&RemoveBlankPodAffinityTermNamespaces{}).Filter(items)
}

type RemoveBlankPodAffinityTermNamespaces struct{}

func (f *RemoveBlankPodAffinityTermNamespaces) Filter(node *kyaml.RNode) (*kyaml.RNode, error) {
	podSpec, err := node.Pipe(
		kyaml.LookupFirstMatch(podSpecPaths),
	)
	if err != nil {
		return node, errors.Wrap(err, "unable to lookup pod spec")
	}

	affinity, err := kyaml.Get("affinity").Filter(podSpec)
	if err != nil {
		return node, errors.Wrap(err, "unable to get affinity field")
	}

	if affinity == nil {
		return node, nil
	}

	affinities := []string{"podAffinity", "podAntiAffinity"}
	for _, afftype := range affinities {
		affinityType, err := kyaml.Get(afftype).Filter(affinity)
		if err != nil {
			return node, errors.Wrap(err, "unable to get affinity type")
		}

		if affinityType != nil {
			termNodes, err := affinityType.Pipe(
				kyaml.Lookup("preferredDuringSchedulingIgnoredDuringExecution"),
			)
			if err != nil {
				return node, errors.Wrap(err, "unable to lookup pod affinity term (preferred)")
			}
			terms, err := termNodes.Elements()
			if err != nil {
				return node, errors.Wrap(err, "unable to get term node elements")
			}
			for _, weightedTerm := range terms {
				term, err := weightedTerm.Pipe(kyaml.Lookup("podAffinityTerm"))
				if err != nil {
					return node, errors.Wrap(err, "unable to lookup podAffinityTerm on weighted pod affinity term")
				}
				if term == nil {
					continue
				}
				if err := removeBlankNamespacesFromPodAffnityTerm(term); err != nil {
					return node, err
				}
			}

			termNodes, err = affinityType.Pipe(
				kyaml.Lookup("requiredDuringSchedulingIgnoredDuringExecution"),
			)
			if err != nil {
				return node, errors.Wrap(err, "unable to lookup pod affinity term (required)")
			}
			terms, err = termNodes.Elements()
			if err != nil {
				return node, errors.Wrap(err, "unable to get term node elements")
			}
			for _, term := range terms {
				if err := removeBlankNamespacesFromPodAffnityTerm(term); err != nil {
					return node, err
				}
			}
		}
	}
	return node, nil
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
