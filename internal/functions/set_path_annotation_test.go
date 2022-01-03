package functions

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestSetPathAnnotationFunctionFilter(t *testing.T) {
	inputyaml := `
apiVersion: v1
kind: Service
metadata:
  name: test
spec:
  ports:
  - name: http
    port: 8080
  selector:
    app: name
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: test-cm
data:
  env: test
  logLevel: debug
`
	var fn SetPathAnnotationFunction

	input, err := kio.ParseAll(inputyaml)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	output, err := fn.Filter(input)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	for _, onode := range output {
		annotations := onode.GetAnnotations()
		if _, ok := annotations[kioutil.PathAnnotation]; !ok {
			assert.Fail(t, "resource s is missing path annotation", onode.GetName())
		}
	}
}

func TestSetPathAnnotationFunctionConfig(t *testing.T) {
	var tests = []struct {
		name            string
		input           string
		expectedPath    string
		expectedPattern string
		expectedError   string
	}{
		{
			name: "configmap",
			input: `apiVersion: v1
kind: ConfigMap
metadata:
  name: test
data:
  path: "testpath"
  pattern: "%s--%s.yaml"
`,
			expectedPath:    "testpath",
			expectedPattern: "%s--%s.yaml",
		},
		{
			name: "empty-configmap",
			input: `apiVersion: v1
kind: ConfigMap
metadata:
  name: test
`,
			expectedPath:    "",
			expectedPattern: "",
		},
		{
			name: "function-config",
			input: `apiVersion: konvert.kumorilabs.io/v1alpha1
kind: SetPathAnnotation
metadata:
  name: path-annotation
spec:
  path: "testpath"
  pattern: "%s--%s.yaml"
`,
			expectedPath:    "testpath",
			expectedPattern: "%s--%s.yaml",
		},
		{
			name: "empty-function-config",
			input: `apiVersion: konvert.kumorilabs.io/v1alpha1
kind: SetPathAnnotation
metadata:
  name: path-annotation
`,
			expectedPath:    "",
			expectedPattern: "",
		},
		{
			name: "invalid-gvk",
			input: `apiVersion: v1
kind: Secret
metadata:
  name: bad-gvk
`,
			expectedError: "`functionConfig` must be a `ConfigMap` or `SetPathAnnotation`",
		},
		{
			name: "bad-yaml-spec",
			input: `apiVersion: konvert.kumorilabs.io/v1alpha1
kind: SetPathAnnotation
metadata:
  name: path-annotation
spec: |
   this is not yaml
`,
			expectedError: "error unmarshaling JSON",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var fn SetPathAnnotationFunction

			input, err := yaml.Parse(test.input)
			if !assert.NoError(t, err, test.name) {
				t.FailNow()
			}

			err = fn.Config(input)

			if test.expectedError != "" {
				if !assert.NotNil(t, err, test.name) {
					t.FailNow()
				}
				if !assert.Contains(t, err.Error(), test.expectedError, test.name) {
					t.FailNow()
				}
			} else {
				if !assert.NoError(t, err, test.name) {
					t.FailNow()
				}
			}

			assert.Equal(t, test.expectedPath, fn.Path, test.name)
			assert.Equal(t, test.expectedPattern, fn.Pattern, test.name)
		})
	}
}

func TestPathAnnotationSetter(t *testing.T) {
	var tests = []struct {
		name               string
		path               string
		pattern            string
		expectedAnnotation string
		resultCount        int
	}{
		{
			name:               "simple-pattern",
			path:               ".",
			pattern:            "%s-%s.yaml",
			expectedAnnotation: "service-test.yaml",
			resultCount:        0,
		},
		{
			name:               "another-pattern",
			path:               ".",
			pattern:            "%s_%s.yaml",
			expectedAnnotation: "service_test.yaml",
			resultCount:        0,
		},
		{
			name:               "with-path",
			path:               "upstream",
			pattern:            "%s-%s.yaml",
			expectedAnnotation: "upstream/service-test.yaml",
			resultCount:        0,
		},
		{
			name:               "with-prefix",
			path:               ".",
			pattern:            "base/%s-%s.yaml",
			expectedAnnotation: "base/service-test.yaml",
			resultCount:        0,
		},
		{
			name:               "with-path-and-prefix",
			path:               "upstream",
			pattern:            "base/%s-%s.yaml",
			expectedAnnotation: "upstream/base/service-test.yaml",
			resultCount:        0,
		},
		{
			name:               "with-empty-path",
			path:               "",
			pattern:            "%s-%s.yaml",
			expectedAnnotation: "service-test.yaml",
			resultCount:        0,
		},
		{
			name:               "with-empty-pattern",
			path:               ".",
			pattern:            "",
			expectedAnnotation: "service-test.yaml",
			resultCount:        0,
		},
	}

	const resyaml = `
apiVersion: v1
kind: Service
metadata:
  name: test
spec:
  ports:
  - name: http
    port: 8080
  selector:
    app: name
`

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fn := PathAnnotation(test.path, test.pattern)
			inputNode, err := yaml.Parse(resyaml)
			if !assert.NoError(t, err, test.name) {
				t.FailNow()
			}
			output, err := fn.Filter(inputNode)
			if !assert.NoError(t, err, test.name) {
				t.FailNow()
			}

			assert.Equal(t, 0, test.resultCount)

			annotations := output.GetAnnotations()
			assert.Contains(t, annotations, kioutil.PathAnnotation, test.name)
			assert.Equal(t, test.expectedAnnotation, annotations[kioutil.PathAnnotation], test.name)
		})
	}
}
