package cmd

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/fn/framework/command"
	"sigs.k8s.io/kustomize/kyaml/kio/filters"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

func mustServiceAccount(name string) *kyaml.RNode {
	a := `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: %s
  annotations:
    config.kubernetes.io/path: base/serviceaccount-%s.yaml
`
	yamlstr := fmt.Sprintf(a, name, name)
	return kyaml.MustParse(yamlstr)
}

type KonvertFunction struct {
	Spec KonvertSpec `yaml:"spec,omitempty"`
}

func (f *KonvertFunction) Config(rn *kyaml.RNode) error {
	yamlstr, err := rn.String()
	if err != nil {
		return errors.Wrap(err, "unable to get yaml from rnode")
	}
	if err := yaml.Unmarshal([]byte(yamlstr), f); err != nil {
		return errors.Wrap(err, "unable to unmarshal konvert config")
	}
	return nil
}

func (f *KonvertFunction) Run(items []*kyaml.RNode) ([]*kyaml.RNode, error) {
	for _, item := range items {
		err := item.PipeE(kyaml.SetAnnotation("managed-by", "konvert"))
		if err != nil {
			return items, errors.Wrap(err, "unable to run konvert filter")
		}
	}
	items = append(items, mustServiceAccount("testing"))

	items, err := filters.MergeFilter{}.Filter(items)
	if err != nil {
		return items, errors.Wrap(err, "unable to merge items")
	}
	return items, nil
}

type KonvertSpec struct {
	Name string `yaml:"name,omitempty"`
}

type KonvertProcessor struct{}

func newFnCommand(args []string) *cobra.Command {
	kp := KonvertProcessor{}
	cmd := command.Build(&kp, command.StandaloneEnabled, false)
	cmd.Use = "fn"
	cmd.Short = "konvert kpt function"
	cmd.Long = "konvert kpt function"
	return cmd
}

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
