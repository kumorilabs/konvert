package functions

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/kio"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestRemoveByAnnotationsFunctionConfig(t *testing.T) {
	var tests = []struct {
		name                string
		input               string
		expectedAnnotations map[string]string
		expectedError       string
	}{
		{
			name: "configmap",
			input: `apiVersion: v1
kind: ConfigMap
metadata:
  name: test
data:
  annotations:
    anno1: val
    anno2: val2
`,
			expectedAnnotations: map[string]string{"anno1": "val", "anno2": "val2"},
		},
		{
			name: "function-config",
			input: `apiVersion: konvert.kumorilabs.io/v1alpha1
kind: RemoveByAnnotations
metadata:
  name: fnconfig
spec:
  annotations:
    anno1: val
    anno2: val2
`,
			expectedAnnotations: map[string]string{"anno1": "val", "anno2": "val2"},
		},
		{
			name: "empty-configmap",
			input: `apiVersion: v1
kind: ConfigMap
metadata:
  name: test
`,
		},
		{
			name: "empty-function-config",
			input: `apiVersion: konvert.kumorilabs.io/v1alpha1
kind: RemoveByAnnotations
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
			expectedError: "`functionConfig` must be a `ConfigMap` or `RemoveByAnnotations`",
		},
		{
			name: "bad-yaml-spec",
			input: `apiVersion: konvert.kumorilabs.io/v1alpha1
kind: RemoveByAnnotations
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
			var fn RemoveByAnnotationsFunction

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

			assert.Equal(t, test.expectedAnnotations, fn.Annotations, test.name)
		})
	}
}

func TestRemoveByAnnotationsFilter(t *testing.T) {
	var tests = []struct {
		name        string
		input       string
		output      string
		annotations map[string]string
	}{
		{
			name: "single-annotation",
			annotations: map[string]string{
				"to-be-removed": "yes",
			},
			input: `apiVersion: v1
kind: Service
metadata:
  name: mysql
  labels:
    app.kubernetes.io/name: mysql
  annotations:
    to-be-removed: "yes"
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
  annotations:
    some-other-annotation: "whatever"
data:
  my.cnf: |2-

    [mysqld]
`,
			output: `apiVersion: v1
kind: ConfigMap
metadata:
  name: mysql
  labels:
    app.kubernetes.io/name: mysql
  annotations:
    some-other-annotation: "whatever"
data:
  my.cnf: |2-

    [mysqld]
`,
		},
		{
			name: "multiple-annotations",
			annotations: map[string]string{
				"to-be-removed": "yes",
				"confirm":       "please",
			},
			input: `apiVersion: v1
kind: Service
metadata:
  name: mysql
  labels:
    app.kubernetes.io/name: mysql
  annotations:
    confirm: "please"
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
  annotations:
    to-be-removed: "yes"
data:
  my.cnf: |2-

    [mysqld]
`,
			output: ``,
		},
		{
			name: "no-annotations",
			input: `apiVersion: v1
kind: Service
metadata:
  name: mysql
  labels:
    app.kubernetes.io/name: mysql
  annotations:
    confirm: "please"
    to-be-removed: "yes"
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
  annotations:
    to-be-removed: "yes"
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
  annotations:
    confirm: "please"
    to-be-removed: "yes"
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
  annotations:
    to-be-removed: "yes"
data:
  my.cnf: |2-

    [mysqld]
`,
		},
		{
			name: "no-matching",
			annotations: map[string]string{
				"no-resources": "match",
			},
			input: `apiVersion: v1
kind: Service
metadata:
  name: mysql
  labels:
    app.kubernetes.io/name: mysql
  annotations:
    confirm: "please"
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
  annotations:
    to-be-removed: "yes"
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
  annotations:
    confirm: "please"
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
  annotations:
    to-be-removed: "yes"
data:
  my.cnf: |2-

    [mysqld]
`,
		},
		{
			name: "all-matching",
			annotations: map[string]string{
				"to-be-removed": "yes",
			},
			input: `apiVersion: v1
kind: Service
metadata:
  name: mysql
  labels:
    app.kubernetes.io/name: mysql
  annotations:
    to-be-removed: "yes"
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
  annotations:
    to-be-removed: "yes"
data:
  my.cnf: |2-

    [mysqld]
`,
			output: ``,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var fn RemoveByAnnotationsFunction
			fn.Annotations = test.annotations

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
