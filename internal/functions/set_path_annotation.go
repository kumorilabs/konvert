package functions

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/kio"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
	"sigs.k8s.io/yaml"
)

const (
	fnSetPathAnnotationName = "path-annotation"
	fnSetPathAnnotationKind = "SetPathAnnotation"
)

type SetPathAnnotationProcessor struct{}

func (p *SetPathAnnotationProcessor) Process(resourceList *framework.ResourceList) error {
	err := p.run(resourceList)
	if err != nil {
		result := &framework.Result{
			Message:  fmt.Sprintf("error running %s: %v", fnSetPathAnnotationName, err.Error()),
			Severity: framework.Error,
		}
		resourceList.Results = append(resourceList.Results, result)
	}
	return err
}

func (p *SetPathAnnotationProcessor) run(resourceList *framework.ResourceList) error {
	var fn SetPathAnnotationFunction
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

type SetPathAnnotationFunction struct {
	kyaml.ResourceMeta `json:",inline" yaml:",inline"`
	Path               string `json:"path,omitempty" yaml:"path,omitempty"`
	Pattern            string `json:"pattern,omitempty" yaml:"pattern,omitempty"`
}

func (f *SetPathAnnotationFunction) Config(rn *kyaml.RNode) error {
	switch {
	case validGVK(rn, "v1", "ConfigMap"):
	case validGVK(rn, fnConfigAPIVersion, fnSetPathAnnotationKind):
		yamlstr, err := rn.String()
		if err != nil {
			return errors.Wrap(err, "unable to get yaml from rnode")
		}
		if err := yaml.Unmarshal([]byte(yamlstr), f); err != nil {
			return errors.Wrap(err, "unable to unmarshal function config")
		}
	default:
		return fmt.Errorf("`functionConfig` must be a `ConfigMap` or `%s`", fnSetPathAnnotationKind)
	}

	return nil
}

func (f *SetPathAnnotationFunction) Run(items []*kyaml.RNode) ([]*kyaml.RNode, error) {
	if f.Path == "" {
		f.Path = "."
	}
	if f.Pattern == "" {
		f.Pattern = "%s-%s.yaml"
	}

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
			"config.kubernetes.io/path",
			filepath.Join(f.path, fmt.Sprintf(f.pattern, kind, name)),
		),
	)
	if err != nil {
		return node, errors.Wrap(err, "unable to set path annotation")
	}
	return node, nil
}
