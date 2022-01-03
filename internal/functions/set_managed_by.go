package functions

import (
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/kio"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

const (
	fnSetManagedByName = "managed-by"
	fnSetManagedByKind = "SetManagedBy"
	defaultManagedBy   = "konvert"
)

type SetManagedByProcessor struct{}

func (p *SetManagedByProcessor) Process(resourceList *framework.ResourceList) error {
	return runFn(&SetManagedByFunction{}, resourceList)
}

type SetManagedByFunction struct {
	kyaml.ResourceMeta `json:",inline" yaml:",inline"`
	Value              string `json:"value,omitempty" yaml:"value,omitempty"`
}

func (f *SetManagedByFunction) Name() string {
	return fnSetManagedByName
}

func (f *SetManagedByFunction) Config(rn *kyaml.RNode) error {
	return loadConfig(f, rn, fnSetManagedByKind)
}

func (f *SetManagedByFunction) Filter(items []*kyaml.RNode) ([]*kyaml.RNode, error) {
	if f.Value == "" {
		f.Value = defaultManagedBy
	}

	return kio.FilterAll(
		kyaml.SetLabel("app.kubernetes.io/managed-by", f.Value),
	).Filter(items)
}
