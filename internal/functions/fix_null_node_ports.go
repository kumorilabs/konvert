package functions

import (
	"github.com/pkg/errors"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

// hack to remove `nodePort: null` in service ports
// see https://github.com/GoogleContainerTools/kpt/issues/2321

const (
	fnFixNullNodePortsName = "fix-null-node-ports"
	fnFixNullNodePortsKind = "FixNullNodePorts"
)

type FixNullNodePortsProcessor struct{}

func (p *FixNullNodePortsProcessor) Process(resourceList *framework.ResourceList) error {
	return runFn(&FixNullNodePortsFunction{}, resourceList)
}

type FixNullNodePortsFunction struct {
	kyaml.ResourceMeta `json:",inline" yaml:",inline"`
}

func (f *FixNullNodePortsFunction) Name() string {
	return fnFixNullNodePortsName
}

func (f *FixNullNodePortsFunction) SetResourceMeta(meta kyaml.ResourceMeta) {
	f.ResourceMeta = meta
}

func (f *FixNullNodePortsFunction) Config(rn *kyaml.RNode) error {
	return loadConfig(f, rn, fnFixNullNodePortsKind)
}

func (f *FixNullNodePortsFunction) Filter(items []*kyaml.RNode) ([]*kyaml.RNode, error) {
	for _, item := range items {
		if item.GetKind() == "Service" {
			portseq, err := item.Pipe(
				kyaml.Lookup("spec", "ports"),
			)
			if err != nil {
				return items, errors.Wrap(err, "unable to lookup ports in Service")
			}

			ports, err := portseq.Elements()
			if err != nil {
				return items, errors.Wrap(err, "unable to get port elements")
			}

			for _, servicePort := range ports {
				nodePort, err := kyaml.Get("nodePort").Filter(servicePort)
				if err != nil {
					return items, errors.Wrap(err, "unable to lookup node port")
				}
				if nodePort != nil {
					if nodePort.IsNilOrEmpty() {
						_, err = kyaml.Clear("nodePort").Filter(servicePort)
						if err != nil {
							return items, errors.Wrap(err, "unable to remove null node port")
						}
					}
				}
			}
		}
	}
	return items, nil
}
