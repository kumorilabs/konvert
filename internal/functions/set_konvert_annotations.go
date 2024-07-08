package functions

import (
	"fmt"

	"github.com/pkg/errors"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/kio"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

const (
	fnSetKonvertAnnotationsName  = "konvert-annotations"
	fnSetKonvertAnnotationsKind  = "SetKonvertAnnotations"
	annotationKonvertGeneratedBy = fnConfigGroup + "/generated-by"
	annotationKonvertChart       = fnConfigGroup + "/chart"
	defaultGeneratedBy           = "konvert"
)

type SetKonvertAnnotationsProcessor struct{}

func (p *SetKonvertAnnotationsProcessor) Process(resourceList *framework.ResourceList) error {
	return runFn(&SetManagedByFunction{}, resourceList)
}

type SetKonvertAnnotationsFunction struct {
	kyaml.ResourceMeta `json:",inline" yaml:",inline"`
	Chart              string `json:"chart,omitempty" yaml:"chart,omitempty"`
	Repo               string `json:"repo,omitempty" yaml:"repo,omitempty"`
}

func (f *SetKonvertAnnotationsFunction) Name() string {
	return fnSetKonvertAnnotationsName
}

func (f *SetKonvertAnnotationsFunction) SetResourceMeta(meta kyaml.ResourceMeta) {
	f.ResourceMeta = meta
}

func (f *SetKonvertAnnotationsFunction) Config(rn *kyaml.RNode) error {
	return loadConfig(f, rn, fnSetKonvertAnnotationsKind)
}

func (f *SetKonvertAnnotationsFunction) Filter(items []*kyaml.RNode) ([]*kyaml.RNode, error) {
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
		kyaml.SetAnnotation(annotationKonvertChart, konvertChartAnnotationValue(f.Repo, f.Chart)),
	).Filter(items)
	if err != nil {
		return items, errors.Wrapf(err, "unable to set annotation %s", annotationKonvertChart)
	}

	return items, nil
}

func konvertChartAnnotationValue(repo, chart string) string {
	if repo != "" {
		return fmt.Sprintf("%s,%s", repo, chart)
	}
	return chart
}
