package functions

import (
	"testing"

	"github.com/stretchr/testify/assert"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestValidGVK(t *testing.T) {
	var tests = []struct {
		name       string
		input      string
		apiVersion string
		kind       string
		valid      bool
	}{
		{
			name: "valid-configmap",
			input: `apiVersion: v1
kind: ConfigMap
metadata:
  name: config
`,
			apiVersion: "v1",
			kind:       "ConfigMap",
			valid:      true,
		},
		{
			name: "kind-no-match",
			input: `apiVersion: v1
kind: ConfigMap
metadata:
  name: config
`,
			apiVersion: "v1",
			kind:       "Service",
			valid:      false,
		},
		{
			name: "apiversion-no-match",
			input: `apiVersion: v1
kind: ConfigMap
metadata:
  name: config
`,
			apiVersion: "apps/v1",
			kind:       "ConfigMap",
			valid:      false,
		},
		{
			name:       "missing-meta",
			input:      `apiVersion: v1`,
			apiVersion: "v1",
			kind:       "ConfigMap",
			valid:      false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			node, err := kyaml.Parse(test.input)
			if !assert.NoError(t, err, test.name) {
				t.FailNow()
			}
			actual := validGVK(node, test.apiVersion, test.kind)
			assert.Equal(t, test.valid, actual, test.name)
		})
	}
}
