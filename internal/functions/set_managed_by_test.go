package functions

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/kio"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestSetManagedByFunctionConfig(t *testing.T) {
	var tests = []struct {
		name          string
		input         string
		expectedValue string
		expectedError string
	}{
		{
			name: "configmap",
			input: `apiVersion: v1
kind: ConfigMap
metadata:
  name: test
data:
  value: "managed-by-this"
`,
			expectedValue: "managed-by-this",
		},
		{
			name: "empty-configmap",
			input: `apiVersion: v1
kind: ConfigMap
metadata:
  name: test
`,
			expectedValue: "",
		},
		{
			name: "function-config",
			input: `apiVersion: konvert.kumorilabs.io/v1alpha1
kind: SetManagedBy
metadata:
  name: managed-by
spec:
  value: "managed-by-this"
`,
			expectedValue: "managed-by-this",
		},
		{
			name: "empty-function-config",
			input: `apiVersion: konvert.kumorilabs.io/v1alpha1
kind: SetManagedBy
metadata:
  name: managed-by
`,
			expectedValue: "",
		},
		{
			name: "invalid-gvk",
			input: `apiVersion: v1
kind: Secret
metadata:
  name: bad-gvk
`,
			expectedError: "`functionConfig` must be a `ConfigMap` or `SetManagedBy`",
		},
		{
			name: "bad-yaml-spec",
			input: `apiVersion: konvert.kumorilabs.io/v1alpha1
kind: SetManagedBy
metadata:
  name: managed-by
spec: |
   this is not yaml
`,
			expectedError: "error unmarshaling JSON",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var fn SetManagedByFunction

			input, err := kyaml.Parse(test.input)
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

			assert.Equal(t, test.expectedValue, fn.Value, test.name)
		})
	}
}

func TestSetManagedByFilter(t *testing.T) {
	var tests = []struct {
		name               string
		input              string
		managedByValue     string
		expectedLabelValue string
	}{
		{
			name: "no-labels",
			input: `
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
`,
			expectedLabelValue: defaultManagedBy,
		},
		{
			name: "not-default",
			input: `
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
`,
			managedByValue:     "managed-by-this",
			expectedLabelValue: "managed-by-this",
		},
		{
			name: "existing-label",
			input: `
apiVersion: v1
kind: Service
metadata:
  name: test
  labels:
    app.kubernetes.io/managed-by: something-else
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
  labels:
    app.kubernetes.io/managed-by: something-else
data:
  env: test
  logLevel: debug
`,
			managedByValue:     "managed-by-this",
			expectedLabelValue: "managed-by-this",
		},
	}

	const managedByLabel = "app.kubernetes.io/managed-by"

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var fn SetManagedByFunction
			fn.Value = test.managedByValue

			input, err := kio.ParseAll(test.input)
			assert.NoError(t, err, test.name)

			output, err := fn.Filter(input)
			assert.NoError(t, err, test.name)

			for _, onode := range output {
				labels := onode.GetLabels()
				labelval, ok := labels[managedByLabel]
				if !ok {
					assert.Fail(t, "resource is missing managed-by label", onode.GetName(), test.name)
					t.FailNow()
				}
				assert.Equal(t, test.expectedLabelValue, labelval, test.name)
			}
		})
	}

}
