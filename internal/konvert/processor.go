package konvert

import (
	"github.com/pkg/errors"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
)

type Processor struct{}

func (p *Processor) Process(resourceList *framework.ResourceList) error {
	err := p.process(resourceList)
	// TODO: results
	return err
}

func (p *Processor) process(resourceList *framework.ResourceList) error {
	var konvert function
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
