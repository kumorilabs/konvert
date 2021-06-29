package functions

import (
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

const (
	fnConfigGroup      = "konvert.kumorilabs.io"
	fnConfigVersion    = "v1alpha1"
	fnConfigAPIVersion = fnConfigGroup + "/" + fnConfigVersion
)

func validGVK(rn *kyaml.RNode, apiVersion, kind string) bool {
	meta, err := rn.GetMeta()
	if err != nil {
		return false
	}
	if meta.APIVersion != apiVersion || meta.Kind != kind {
		return false
	}
	return true
}
