package konvert

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"sigs.k8s.io/kustomize/kyaml/yaml"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

type Kustomizer struct {
	path      string
	namespace string
}

func Kustomization(path, namespace string) Kustomizer {
	return Kustomizer{path, namespace}
}

func (f Kustomizer) Filter(nodes []*kyaml.RNode) ([]*kyaml.RNode, error) {
	var resources []string

	for _, node := range nodes {
		pathstr, err := node.Pipe(yaml.GetAnnotation("config.kubernetes.io/path"))
		if err != nil {
			return nodes, errors.Wrap(err, "unable to get config path annotation")
		}

		// if we can't figure out the path, this resource will not be included
		// in the kustomization
		ynode := pathstr.YNode()
		if ynode == nil {
			continue
		}

		path := ynode.Value
		if path == "" {
			continue
		}

		// unresolvable paths abort the filter
		resourcePath, err := filepath.Rel(f.path, path)
		if err != nil {
			return nodes, errors.Wrap(err, "unable to resolve config path")
		}

		resources = append(resources, resourcePath)
	}

	kustomizationNode, err := kyaml.Parse(f.kustomizationYAML(resources))
	if err != nil {
		return nodes, errors.Wrap(err, "unable to generate kustomization yaml")
	}
	nodes = append(nodes, kustomizationNode)

	return nodes, nil
}

func (f Kustomizer) kustomizationYAML(resources []string) string {
	// namespace should be optional
	template := `
namespace: %s
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
metadata:
  name: %s
  annotations:
    config.kubernetes.io/path: %s
resources:
%s
`

	resourceList := make([]string, len(resources))
	for i, res := range resources {
		resourceList[i] = fmt.Sprintf("- %s", res)
	}

	name := "kustomization"
	if f.path != "" && f.path != "." {
		name = f.path
	}

	return fmt.Sprintf(
		template,
		f.namespace,
		name,
		filepath.Join(f.path, "kustomization.yaml"),
		strings.Join(resourceList, "\n"),
	)
}
