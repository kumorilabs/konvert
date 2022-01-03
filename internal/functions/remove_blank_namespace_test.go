package functions

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/kio"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestRemoveBlankNamespaceFunctionConfig(t *testing.T) {
	var tests = []struct {
		name          string
		input         string
		expectedError string
	}{
		{
			name: "configmap",
			input: `apiVersion: v1
kind: ConfigMap
metadata:
  name: test
`,
		},
		{
			name: "function-config",
			input: `apiVersion: konvert.kumorilabs.io/v1alpha1
kind: RemoveBlankNamespace
metadata:
  name: fnconfig
`,
		},
		{
			name: "invalid-gvk",
			input: `apiVersion: v1
kind: Secret
metadata:
  name: bad-gvk
`,
			expectedError: "`functionConfig` must be a `ConfigMap` or `RemoveBlankNamespace`",
		},
		{
			name: "bad-yaml-spec",
			input: `apiVersion: konvert.kumorilabs.io/v1alpha1
kind: RemoveBlankNamespace
metadata:
  name: fnconfig
spec: |
   this is not yaml
`,
			expectedError: "error unmarshaling JSON",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var fn RemoveBlankNamespaceFunction

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
		})
	}
}

func TestRemoveBlankNamespaceFilter(t *testing.T) {
	var tests = []struct {
		name   string
		input  string
		output string
	}{
		{
			name: "single-resource",
			input: `apiVersion: v1
kind: Service
metadata:
  name: mysql
  namespace:
  labels:
    app.kubernetes.io/name: mysql
spec:
  type: ClusterIP
  ports:
  - name: mysql
    port: 3306
    protocol: TCP
    targetPort: mysql
  selector:
    app.kubernetes.io/name: mysql
`,
			output: `apiVersion: v1
kind: Service
metadata:
  name: mysql
  labels:
    app.kubernetes.io/name: mysql
spec:
  type: ClusterIP
  ports:
  - name: mysql
    port: 3306
    protocol: TCP
    targetPort: mysql
  selector:
    app.kubernetes.io/name: mysql
`,
		},
		{
			name: "multiple-resources",
			input: `apiVersion: v1
kind: Service
metadata:
  name: mysql
  namespace:
  labels:
    app.kubernetes.io/name: mysql
spec:
  type: ClusterIP
  ports:
  - name: mysql
    port: 3306
    protocol: TCP
    targetPort: mysql
  selector:
    app.kubernetes.io/name: mysql
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: mysql
  namespace:
  labels:
    app.kubernetes.io/name: mysql
data:
  my.cnf: |2-

    [mysqld]
`,
			output: `apiVersion: v1
kind: Service
metadata:
  name: mysql
  labels:
    app.kubernetes.io/name: mysql
spec:
  type: ClusterIP
  ports:
  - name: mysql
    port: 3306
    protocol: TCP
    targetPort: mysql
  selector:
    app.kubernetes.io/name: mysql
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: mysql
  labels:
    app.kubernetes.io/name: mysql
data:
  my.cnf: |2-

    [mysqld]
`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var fn RemoveBlankNamespaceFunction

			input, err := kio.ParseAll(test.input)
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			output, err := fn.Filter(input)
			if !assert.NoError(t, err, test.name) {
				t.FailNow()
			}

			outputstr, err := kio.StringAll(output)
			if !assert.NoError(t, err, test.name) {
				t.FailNow()
			}

			assert.Equal(t, test.output, outputstr, test.name)
		})
	}
}
