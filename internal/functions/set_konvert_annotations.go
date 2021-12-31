package functions

import (
	"fmt"

	"github.com/pkg/errors"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/kio"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
	"sigs.k8s.io/yaml"
)

const (
	fnSetKonvertAnnotationsName  = "konvert-annotations"
	fnSetKonvertAnnotationsKind  = "SetKonvertAnnotations"
	annotationKonvertGeneratedBy = fnConfigGroup + "/generated-by"
	annotationKonvertChart       = fnConfigGroup + "/chart"
	AnnotationKonvertChart       = annotationKonvertChart
	defaultGeneratedBy           = "konvert"
)

type SetKonvertAnnotationsProcessor struct{}

func (p *SetKonvertAnnotationsProcessor) Process(resourceList *framework.ResourceList) error {
	err := p.run(resourceList)
	if err != nil {
		result := &framework.Result{
			Message:  fmt.Sprintf("error running %s: %v", fnSetKonvertAnnotationsName, err.Error()),
			Severity: framework.Error,
		}
		resourceList.Results = append(resourceList.Results, result)
	}
	return err
}

func (p *SetKonvertAnnotationsProcessor) run(resourceList *framework.ResourceList) error {
	var fn SetKonvertAnnotationsFunction
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

type SetKonvertAnnotationsFunction struct {
	kyaml.ResourceMeta `json:",inline" yaml:",inline"`
	Chart              string `json:"chart,omitempty" yaml:"chart,omitempty"`
	Repo               string `json:"repo,omitempty" yaml:"repo,omitempty"`
}

func (f *SetKonvertAnnotationsFunction) Config(rn *kyaml.RNode) error {
	switch {
	case validGVK(rn, "v1", "ConfigMap"):
	case validGVK(rn, fnConfigAPIVersion, fnSetKonvertAnnotationsKind):
		yamlstr, err := rn.String()
		if err != nil {
			return errors.Wrap(err, "unable to get yaml from rnode")
		}
		if err := yaml.Unmarshal([]byte(yamlstr), f); err != nil {
			return errors.Wrap(err, "unable to unmarshal function config")
		}
	default:
		return fmt.Errorf("`functionConfig` must be a `ConfigMap` or `%s`", fnSetKonvertAnnotationsKind)
	}

	return nil
}

func (f *SetKonvertAnnotationsFunction) Run(items []*kyaml.RNode) ([]*kyaml.RNode, error) {
	if f.Repo == "" {
		return items, fmt.Errorf("repo cannot be empty")
	}
	if f.Chart == "" {
		return items, fmt.Errorf("chart cannot be empty")
	}

	items, err := kio.FilterAll(
		kyaml.SetAnnotation(annotationKonvertGeneratedBy, defaultGeneratedBy),
	).Filter(items)
	if err != nil {
		return items, errors.Wrapf(err, "unable to set annotation %s", annotationKonvertGeneratedBy)
	}

	items, err = kio.FilterAll(
		kyaml.SetAnnotation(annotationKonvertChart, fmt.Sprintf("%s,%s", f.Repo, f.Chart)),
	).Filter(items)
	if err != nil {
		return items, errors.Wrapf(err, "unable to set annotation %s", annotationKonvertChart)
	}

	return items, nil
}
