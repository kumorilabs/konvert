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

func (f *KustomizerFunction) SetResourceMeta(meta kyaml.ResourceMeta) {
	f.ResourceMeta = meta
}

func (f *KustomizerFunction) Config(rn *kyaml.RNode) error {
	return loadConfig(f, rn, fnKustomizerKind)
}

func (f *KustomizerFunction) kustomizationAtPath(path string, items []*kyaml.RNode) (*kyaml.RNode, bool, error) {
	kustomizations, err := kustomizationFilter{}.Filter(items)
	if err != nil {
		return nil, false, errors.Wrap(err, "unable to run kustomization filter")
	}

	path = normalizePath(path)
	var knodes []*kyaml.RNode
	for _, kustomization := range kustomizations {
		respath := kustomization.GetAnnotations()[kioutil.PathAnnotation]
		base := filepath.Dir(respath)
		if base == path {
			knodes = append(knodes, kustomization)
		}
	}

	var (
		kustnode *kyaml.RNode
		created  bool
	)
	if len(knodes) > 0 {
		kustnode = knodes[0]
	} else {
		kustnode = f.buildKustomizationNode(path)
		created = true
	}
	return kustnode, created, nil
}

func (f *KustomizerFunction) kustomizeResources(kustnode *kyaml.RNode, resitems []string) error {
	resourceComment := fmt.Sprintf("%s: %s", f.ResourceAnnotationName, f.ResourceAnnotationValue)
	sort.Strings(resitems)

	resources, err := kustnode.Pipe(kyaml.LookupCreate(kyaml.SequenceNode, "resources"))
	if err != nil {
		return errors.Wrap(err, "unable to get kustomization resources node")
	}
	reselems, err := resources.Elements()
	if err != nil {
		return errors.Wrap(err, "unable to read element from kustomization resources")
	}
	var resourceItems []*kyaml.Node
	for _, e := range reselems {
		if e.YNode().LineComment != "# "+resourceComment {
			resourceItems = append(resourceItems, e.YNode())
		}
	}

	if err := kustnode.PipeE(kyaml.Clear("resources")); err != nil {
		return errors.Wrap(err, "unable to clear resources field")
	}

	resources, err = kustnode.Pipe(kyaml.LookupCreate(kyaml.SequenceNode, "resources"))
	if err != nil {
		return errors.Wrap(err, "unable to get kustomization resources node")
	}

	for _, res := range resitems {
		resitem := &kyaml.Node{
			Kind:        kyaml.ScalarNode,
			Value:       res,
			LineComment: resourceComment,
		}
		resourceItems = append(resourceItems, resitem)
	}

	resitemmap := make(map[string]bool)
	for _, res := range resourceItems {
		if _, val := resitemmap[res.Value]; !val {
			resitemmap[res.Value] = true
			err = resources.PipeE(
				kyaml.Append(res),
			)
			if err != nil {
				return errors.Wrap(err, "unable to append to kustomization resources")
			}
		}
	}
	return nil
}

func (f *KustomizerFunction) kustomizeNamespace(kustnode *kyaml.RNode, namespace string) error {
	if f.Namespace != "" {
		err := kustnode.PipeE(
			kyaml.SetField("namespace", kyaml.NewScalarRNode(f.Namespace)),
		)
		if err != nil {
			return errors.Wrap(err, "unable to set kustomization namespace")
		}
	} else {
		err := kustnode.PipeE(
			kyaml.Clear("namespace"),
		)
		if err != nil {
			return errors.Wrap(err, "unable to clear kustomization namespace")
		}
	}
	return nil
}

func (f *KustomizerFunction) Filter(items []*kyaml.RNode) ([]*kyaml.RNode, error) {
	if f.ResourceAnnotationName == "" {
		return items, fmt.Errorf("resource annotation name cannot be empty")
	}
	if f.ResourceAnnotationValue == "" {
		return items, fmt.Errorf("resource annotation value cannot be empty")
	}

	// get or create kustomization.yaml at Path
	kustnode, created, err := f.kustomizationAtPath(f.Path, items)
	if err != nil {
		return items, err
	}
	if created {
		items = append(items, kustnode)
	}

	// set or clear namespace
	if err := f.kustomizeNamespace(kustnode, f.Namespace); err != nil {
		return items, err
	}

	// build list of kustomization resources
	// this assumes that items are already annotated with
	// f.ResourceAnnotationName=f.ResourceAnnotationValue
	// and Path
	// (see SetKonvertAnnotationsFunction, SetPathAnnotationFunction)
	var kustresources []string
	for _, node := range items {
		// make sure we never add the kustnode to the resource list
		if node == kustnode {
			continue
		}
		path := node.GetAnnotations()[kioutil.PathAnnotation]
		resannotationvalue := node.GetAnnotations()[f.ResourceAnnotationName]
		if path != "" && f.ResourceAnnotationValue == resannotationvalue {
			// The kustomization.yaml we are writing here lives in the same
			// directory as the generated resources. So, we don't want to use
			// the relative path on disk when we write the `resources` section
			// in the kustomization.yaml. We will just use the file names.
			kustresources = append(kustresources, filepath.Base(path))
		}
	}

	// set kustomization resourrces
	if err := f.kustomizeResources(kustnode, kustresources); err != nil {
		return items, err
	}

	// if we are kustomizing resources in a subdirectory (upstream, for
	// example), write a kustomization file in the parent with the subdirectory
	// as a resource
	// Example:
	// resources:
	// - upstream
	if !isDefaultPath(f.Path) {
		baseKustNode, created, err := f.kustomizationAtPath(".", items)
		if err != nil {
			return items, err
		}
		if created {
			items = append(items, baseKustNode)
		}
		if err := f.kustomizeResources(baseKustNode, []string{f.Path}); err != nil {
			return items, err
		}
	}

	return items, nil
}

func (f KustomizerFunction) buildKustomizationNode(kpath string) *kyaml.RNode {
	template := `
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
metadata:
  name: %s
  annotations:
    config.kubernetes.io/local-config: 'true'
    %s: %s
`

	kustomizeYaml := fmt.Sprintf(
		template,
		"kustomization",
		kioutil.PathAnnotation,
		filepath.Join(kpath, "kustomization.yaml"),
	)

	return kyaml.MustParse(kustomizeYaml)
}

func isDefaultPath(path string) bool {
	return path == "." || path == ""
}

func normalizePath(path string) string {
	if isDefaultPath(path) {
		return "."
	}
	return path
}
