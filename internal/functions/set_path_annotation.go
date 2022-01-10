package functions

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

const (
	fnSetPathAnnotationName = "path-annotation"
	fnSetPathAnnotationKind = "SetPathAnnotation"
)

type SetPathAnnotationProcessor struct{}

func (p *SetPathAnnotationProcessor) Process(resourceList *framework.ResourceList) error {
	return runFn(&SetPathAnnotationFunction{}, resourceList)
}

// not sure why we have a path and a pattern when you can include a path in the
// pattern :shrug:
type SetPathAnnotationFunction struct {
	kyaml.ResourceMeta `json:",inline" yaml:",inline"`
	Path               string `json:"path,omitempty" yaml:"path,omitempty"`
	Pattern            string `json:"pattern,omitempty" yaml:"pattern,omitempty"`
}

func (f *SetPathAnnotationFunction) Name() string {
	return fnSetPathAnnotationName
}

func (f *SetPathAnnotationFunction) SetResourceMeta(meta kyaml.ResourceMeta) {
	f.ResourceMeta = meta
}

func (f *SetPathAnnotationFunction) Config(rn *kyaml.RNode) error {
	return loadConfig(f, rn, fnSetPathAnnotationKind)
}

func (f *SetPathAnnotationFunction) Filter(items []*kyaml.RNode) ([]*kyaml.RNode, error) {
	items, err := kio.FilterAll(
		PathAnnotation(f.Path, f.Pattern),
	).Filter(items)
	if err != nil {
		return items, errors.Wrapf(err, "unable to set path annotation")
	}

	return items, nil
}

type PathAnnotationSetter struct {
	path    string
	pattern string
}

func PathAnnotation(path, pattern string) PathAnnotationSetter {
	return PathAnnotationSetter{path, pattern}
}

func (f PathAnnotationSetter) Filter(node *kyaml.RNode) (*kyaml.RNode, error) {
	if f.path == "" {
		f.path = "."
	}
	if f.pattern == "" {
		f.pattern = "%s-%s.yaml"
	}

	meta, err := node.GetMeta()
	if err != nil {
		return node, errors.Wrap(err, "unable to get meta from rnode")
	}
	kind := strings.ToLower(meta.Kind)
	name := meta.Name

	// TODO: pattern is not super useful yet
	// you can only pass a fmt string with two string values
	// which will be replaced with kind and name respectively
	// Examples:
	// %s-%s.yaml
	// %s_%s.yaml
	// base/%s-%s.yaml
	err = node.PipeE(
		kyaml.SetAnnotation(
			kioutil.PathAnnotation,
			filepath.Join(f.path, fmt.Sprintf(f.pattern, kind, name)),
		),
	)
	if err != nil {
		return node, errors.Wrap(err, "unable to set path annotation")
	}
	return node, nil
}
