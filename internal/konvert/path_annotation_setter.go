package konvert

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

type PathAnnotationSetter struct {
	pattern string
}

func PathAnnotation(pattern string) PathAnnotationSetter {
	return PathAnnotationSetter{pattern}
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
			fmt.Sprintf(f.pattern, kind, name),
		),
	)
	if err != nil {
		return node, errors.Wrap(err, "unable to set path annotation")
	}
	return node, nil
}
