package functions

import (
	"fmt"
	"path/filepath"
	"sort"

	"github.com/pkg/errors"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

const (
	fnKustomizeConfigGroup      = "kustomize.config.k8s.io"
	fnKustomizeConfigVersion    = "v1beta1"
	fnKustomizeConfigAPIVersion = fnKustomizeConfigGroup + "/" + fnKustomizeConfigVersion
	fnKustomizeConfigKind       = "Kustomization"
	fnKustomizerName            = "kustomizer"
	fnKustomizerKind            = "Kustomizer"
)

type kustomizationFilter struct{}

func (f kustomizationFilter) Filter(nodes []*kyaml.RNode) ([]*kyaml.RNode, error) {
	// there really only should ever be one...
	var kustomizationNodes []*kyaml.RNode
	for _, node := range nodes {
		meta, err := node.GetMeta()
		if err != nil {
			return nodes, errors.Wrap(err, "unable to get meta from rnode")
		}
		if meta.APIVersion == fnKustomizeConfigAPIVersion && meta.Kind == fnKustomizeConfigKind {
			kustomizationNodes = append(kustomizationNodes, node)
		}
	}
	return kustomizationNodes, nil
}

type KustomizerProcessor struct{}

func (p *KustomizerProcessor) Process(resourceList *framework.ResourceList) error {
	return runFn(&KustomizerFunction{}, resourceList)
}

type KustomizerFunction struct {
	kyaml.ResourceMeta      `json:",inline" yaml:",inline"`
	Path                    string `json:"path,omitempty" yaml:"path,omitempty"`
	Namespace               string `json:"namespace,omitempty" yaml:"namespace,omitempty"`
	ResourceAnnotationName  string `json:"resource_annotation_name,omitempty" yaml:"resource_annotation_name,omitempty"`
	ResourceAnnotationValue string `json:"resource_annotation_value,omitempty" yaml:"resource_annotation_value,omitempty"`
}

func (f *KustomizerFunction) Name() string {
	return fnKustomizerName
}

func (f *KustomizerFunction) Config(rn *kyaml.RNode) error {
	return loadConfig(f, rn, fnKustomizerKind)
}

func (f *KustomizerFunction) Filter(items []*kyaml.RNode) ([]*kyaml.RNode, error) {
	if f.ResourceAnnotationName == "" {
		return items, fmt.Errorf("resource annotation name cannot be empty")
	}
	if f.ResourceAnnotationValue == "" {
		return items, fmt.Errorf("resource annotation value cannot be empty")
	}

	resourceComment := fmt.Sprintf("%s: %s", f.ResourceAnnotationName, f.ResourceAnnotationValue)
	knodes, err := kustomizationFilter{}.Filter(items)
	if err != nil {
		return items, errors.Wrap(err, "unable to run kustomization filter")
	}
	var kustnode *kyaml.RNode
	if len(knodes) > 0 {
		kustnode = knodes[0]
	} else {
		kustnode = f.buildKustomizationNode()
		items = append(items, kustnode)
	}

	if f.Namespace != "" {
		err := kustnode.PipeE(
			kyaml.SetField("namespace", kyaml.NewScalarRNode(f.Namespace)),
		)
		if err != nil {
			return items, errors.Wrap(err, "unable to set kustomization namespace")
		}
	} else {
		err := kustnode.PipeE(
			kyaml.Clear("namespace"),
		)
		if err != nil {
			return items, errors.Wrap(err, "unable to clear kustomization namespace")
		}
	}

	resources, err := kustnode.Pipe(kyaml.LookupCreate(kyaml.SequenceNode, "resources"))
	if err != nil {
		return items, errors.Wrap(err, "unable to get kustomization resources node")
	}
	reselems, err := resources.Elements()
	if err != nil {
		return items, errors.Wrap(err, "unable to read element from kustomization resources")
	}
	var preservedResources []*kyaml.Node
	for _, e := range reselems {
		if e.YNode().LineComment != "# "+resourceComment {
			preservedResources = append(preservedResources, e.YNode())
		}
	}

	if err := kustnode.PipeE(kyaml.Clear("resources")); err != nil {
		return items, errors.Wrap(err, "unable to clear resources field")
	}

	resources, err = kustnode.Pipe(kyaml.LookupCreate(kyaml.SequenceNode, "resources"))
	if err != nil {
		return items, errors.Wrap(err, "unable to get kustomization resources node")
	}

	for _, res := range preservedResources {
		err = resources.PipeE(
			kyaml.Append(res),
		)
		if err != nil {
			return items, errors.Wrap(err, "unable to append to kustomization resources")
		}
	}

	// this assumes that items are already annotated with
	// f.ResourceAnnotationName=f.ResourceAnnotationValue
	// and Path
	var kustresources []string
	for _, node := range items {
		// make sure we never add the kustnode to the resource list
		if node == kustnode {
			continue
		}
		path := node.GetAnnotations()[kioutil.PathAnnotation]
		resannotationvalue := node.GetAnnotations()[f.ResourceAnnotationName]
		if path != "" && f.ResourceAnnotationValue == resannotationvalue {
			kustresources = append(kustresources, path)
		}
	}

	sort.Strings(kustresources)
	for _, res := range kustresources {
		err = resources.PipeE(kyaml.Append(
			&kyaml.Node{
				Kind:        kyaml.ScalarNode,
				Value:       res,
				LineComment: resourceComment,
			},
		))
	}

	return items, nil
}

func (f KustomizerFunction) buildKustomizationNode() *kyaml.RNode {
	template := `
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
metadata:
  name: %s
  annotations:
    %s: %s
`

	kustomizeYaml := fmt.Sprintf(
		template,
		"kustomization",
		kioutil.PathAnnotation,
		filepath.Join(f.Path, "kustomization.yaml"),
	)

	return kyaml.MustParse(kustomizeYaml)
}
