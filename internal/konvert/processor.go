package konvert

import (
	"github.com/pkg/errors"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
)

type KonvertProcessor struct{}

func (p *KonvertProcessor) Process(resourceList *framework.ResourceList) error {
	err := p.process(resourceList)
	// TODO: results
	return err
}

func (p *KonvertProcessor) process(resourceList *framework.ResourceList) error {
	var konvert KonvertFunction
	err := konvert.Config(resourceList.FunctionConfig)
	if err != nil {
		return errors.Wrap(err, "unable to configure konvert")
	}
	resourceList.Items, err = konvert.Run(resourceList.Items)
	if err != nil {
		return errors.Wrap(err, "unable to run konvert")
	}
	return nil
}
