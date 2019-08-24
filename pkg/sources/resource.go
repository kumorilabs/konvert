package sources

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// Resource is an unstructured Kubernetes resource
type Resource struct {
	unstructured.Unstructured
}

// New creates a new resources from a map
func New(obj map[string]interface{}) Resource {
	return Resource{unstructured.Unstructured{Object: obj}}
}

// ToList expands a list of resources
func (r *Resource) ToList() ([]Resource, error) {
	if !r.IsList() {
		return []Resource{*r}, nil
	}

	resources := []Resource{}
	err := r.EachListItem(func(o runtime.Object) error {
		u := o.(*unstructured.Unstructured)
		if len(u.Object) > 0 {
			resources = append(resources, New(u.Object))
		}
		return nil
	})

	return resources, err
}
