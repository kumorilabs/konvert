package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"

	yaml "gopkg.in/yaml.v2"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/helm/pkg/manifest"
	"k8s.io/helm/pkg/releaseutil"
	"sigs.k8s.io/kustomize/k8sdeps/kunstruct"
	"sigs.k8s.io/kustomize/pkg/resmap"
	"sigs.k8s.io/kustomize/pkg/resource"
)

const (
	// HookAnno is the label name for a hook
	HookAnno = "helm.sh/hook"
	// HookReleaseTestSuccess is the value for a test-success hook
	HookReleaseTestSuccess = "test-success"
	// HookReleaseTestFailure is the value for a test-failure hook
	HookReleaseTestFailure = "test-failure"
)

func main() {
	data, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		panic(err)
	}

	unparsed := releaseutil.SplitManifests(string(data))
	manifests := manifest.SplitManifests(unparsed)

	var resourceMap = resmap.ResMap{}
	for _, m := range manifests {
		resList, err := newResources([]byte(m.Content))
		if err != nil {
			panic(err)
		}

		for _, r := range resList {
			resourceMap[r.Id()] = r
		}
	}

	for _, res := range resourceMap {
		if isHelmTestResource(res) {
			// skip helm tests
			continue
		}

		removeHelmLabels(res.Map())
		stripReleaseName(res.Map())

		output, err := yaml.Marshal(res.Map())
		if err != nil {
			panic(err)
		}

		filename, err := resourceFileName(res)
		if err != nil {
			panic(err)
		}

		err = ioutil.WriteFile(filename, output, 0644)
		if err != nil {
			panic(err)
		}
	}
}

var helmLabels = []string{"chart", "release", "heritage"}

func isHelmTestResource(res *resource.Resource) bool {
	for k, v := range res.GetAnnotations() {
		if k == HookAnno {
			if v == HookReleaseTestSuccess || v == HookReleaseTestFailure {
				return true
			}
		}

	}
	return false
}

func removeHelmLabels(resMap map[string]interface{}) {
	for key, val := range resMap {
		switch valtype := val.(type) {
		case map[string]interface{}:
			if key == "labels" || key == "matchLabels" || key == "selector" {
				for _, helmLabel := range helmLabels {
					delete(valtype, helmLabel)
				}
			}
			removeHelmLabels(valtype)
		case []interface{}:
			for i := range valtype {
				itemtype, ok := valtype[i].(map[string]interface{})
				if ok {
					removeHelmLabels(itemtype)
				}
			}
		}

	}
}

func stripReleaseName(resMap map[string]interface{}) {
	for key, val := range resMap {
		switch valtype := val.(type) {
		case map[string]interface{}:
			stripReleaseName(valtype)
		case []interface{}:
			for i := range valtype {
				itemtype, ok := valtype[i].(map[string]interface{})
				if ok {
					stripReleaseName(itemtype)
				}
			}
		case string:
			resMap[key] = strings.Replace(valtype, "RELEASE-NAME-", "", -1)
		}
	}
}

func resourceFileName(res *resource.Resource) (string, error) {
	kind, err := res.GetFieldValue("kind")
	if err != nil {
		return "", err
	}

	name, err := res.GetFieldValue("metadata.name")
	if err != nil {
		return "", err
	}

	return strings.ToLower(fmt.Sprintf("%s-%s.yaml", name, kind)), nil
}

func printResource(res *resource.Resource) {
	output, err := yaml.Marshal(res.Map())
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	fmt.Println("---")
	fmt.Println(string(output))
}

func newResources(in []byte) ([]*resource.Resource, error) {
	decoder := k8syaml.NewYAMLOrJSONDecoder(bytes.NewReader(in), 1024)
	rf := resource.NewFactory(kunstruct.NewKunstructuredFactoryImpl())

	var result []*resource.Resource
	var err error
	for err == nil || isEmptyYamlError(err) {
		var out map[string]interface{}
		err = decoder.Decode(&out)
		if err == nil {
			// ignore empty chunks
			if len(out) == 0 {
				continue
			}

			if list, ok := isList(out); ok {
				for _, i := range list {
					if item, ok := i.(map[string]interface{}); ok {
						result = append(result, rf.FromMap(item))
					}
				}
			} else {
				result = append(result, rf.FromMap(out))
			}
		}
	}
	if err != io.EOF {
		return nil, err
	}
	return result, nil
}

func isEmptyYamlError(err error) bool {
	return strings.Contains(err.Error(), "is missing in 'null'")
}

func isList(res map[string]interface{}) ([]interface{}, bool) {
	itemList, ok := res["items"]
	if !ok {
		return nil, false
	}

	items, ok := itemList.([]interface{})
	return items, ok
}
