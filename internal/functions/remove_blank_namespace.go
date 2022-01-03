package functions

import (
	"github.com/pkg/errors"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

const (
	fnRemoveBlankNamespaceName = "remove-blank-namespace"
	fnRemoveBlankNamespaceKind = "RemoveBlankNamespace"
)

type RemoveBlankNamespaceProcessor struct{}

func (p *RemoveBlankNamespaceProcessor) Process(resourceList *framework.ResourceList) error {
	return runFn(&RemoveBlankNamespaceFunction{}, resourceList)
}

type RemoveBlankNamespaceFunction struct {
	kyaml.ResourceMeta `json:",inline" yaml:",inline"`
}

func (f *RemoveBlankNamespaceFunction) Name() string {
	return fnRemoveBlankNamespaceName
}
func (f *RemoveBlankNamespaceFunction) Config(rn *kyaml.RNode) error {
	return loadConfig(f, rn, fnRemoveBlankNamespaceKind)
}

func (f *RemoveBlankNamespaceFunction) Filter(items []*kyaml.RNode) ([]*kyaml.RNode, error) {
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
