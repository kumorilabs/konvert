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
	fnConfigGroup      = "konvert.kumorilabs.io"
	fnConfigVersion    = "v1alpha1"
	fnConfigAPIVersion = fnConfigGroup + "/" + fnConfigVersion
)

var podSpecPaths = [][]string{
	// e.g. Deployment, ReplicaSet, DaemonSet, Job, StatefulSet
	{"spec", "template", "spec"},
	// e.g. CronJob
	{"spec", "jobTemplate", "spec", "template", "spec"},
	// e.g. Pod
	{"spec"},
	// e.g. PodTemplate
	{"template", "spec"},
}

type konvertFunction interface {
	kio.Filter
	Config(*kyaml.RNode) error
	Name() string
}

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

func runFn(fn konvertFunction, resourceList *framework.ResourceList) error {
	err := fn.Config(resourceList.FunctionConfig)
	if err != nil {
		return errors.Wrap(err, "failed to configure function")
	}

	resourceList.Items, err = fn.Filter(resourceList.Items)
	if err != nil {
		resourceList.Results = framework.Results{
			&framework.Result{
				Message:  fmt.Sprintf("error running %s: %v", fn.Name(), err.Error()),
				Severity: framework.Error,
			},
		}
		return resourceList.Results
	}

	return nil
}

func unmarshalConfig(fn konvertFunction, rn *kyaml.RNode, field string) error {
	spec := rn.Field(field)
	if spec == nil {
		return nil
	}

	yamlstr, err := spec.Value.String()
	if err != nil {
		return errors.Wrap(err, "unable to get yaml from spec rnode")
	}
	if err := yaml.Unmarshal([]byte(yamlstr), fn); err != nil {
		return errors.Wrap(err, "unable to unmarshal function config spec")
	}
	return nil
}

func loadConfig(fn konvertFunction, rn *kyaml.RNode, kind string) error {
	switch {
	case validGVK(rn, "v1", "ConfigMap"):
		return unmarshalConfig(fn, rn, "data")
	case validGVK(rn, fnConfigAPIVersion, kind):
		return unmarshalConfig(fn, rn, "spec")
	default:
		return fmt.Errorf("`functionConfig` must be a `ConfigMap` or `%s`", kind)
	}
}
