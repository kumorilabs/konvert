package functions

import (
	"github.com/pkg/errors"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/kio"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

const (
	fnRemoveBlankAffinitiesName = "remove-blank-affinities"
	fnRemoveBlankAffinitiesKind = "RemoveBlankAffinities"
)

type RemoveBlankAffinitiesProcessor struct{}

func (p *RemoveBlankAffinitiesProcessor) Process(resourceList *framework.ResourceList) error {
	return runFn(&RemoveBlankAffinitiesFunction{}, resourceList)
}

type RemoveBlankAffinitiesFunction struct {
	kyaml.ResourceMeta `json:",inline" yaml:",inline"`
}

func (f *RemoveBlankAffinitiesFunction) Name() string {
	return fnRemoveBlankAffinitiesName
}

func (f *RemoveBlankAffinitiesFunction) SetResourceMeta(meta kyaml.ResourceMeta) {
	f.ResourceMeta = meta
}

func (f *RemoveBlankAffinitiesFunction) Config(rn *kyaml.RNode) error {
	return loadConfig(f, rn, fnRemoveBlankAffinitiesKind)
}

func (f *RemoveBlankAffinitiesFunction) Filter(items []*kyaml.RNode) ([]*kyaml.RNode, error) {
	return kio.FilterAll(&RemoveBlankAffinity{}).Filter(items)
}

type RemoveBlankAffinity struct{}

func (f *RemoveBlankAffinity) Filter(node *kyaml.RNode) (*kyaml.RNode, error) {
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

	if affinity != nil {
		affinities := []string{"podAffinity", "nodeAffinity", "podAntiAffinity"}
		for _, affinityField := range affinities {
			field := affinity.Field(affinityField)
			if field.IsNilOrEmpty() {
				err = affinity.PipeE(kyaml.Clear(affinityField))
				if err != nil {
					return node, errors.Wrapf(err, "unable to clear field %q", affinityField)
				}
			}
		}
		if affinity.IsNilOrEmpty() {
			err = podSpec.PipeE(kyaml.Clear("affinity"))
			if err != nil {
				return node, errors.Wrapf(err, "unable to clear field %q", "affinity")
			}
		}
	}

	return node, nil
}
