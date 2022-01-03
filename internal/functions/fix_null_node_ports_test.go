package functions

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/kio"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestFixNullNodePortsFunctionConfig(t *testing.T) {
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
kind: FixNullNodePorts
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
			expectedError: "`functionConfig` must be a `ConfigMap` or `FixNullNodePorts`",
		},
		{
			name: "bad-yaml-spec",
			input: `apiVersion: konvert.kumorilabs.io/v1alpha1
kind: FixNullNodePorts
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
			var fn FixNullNodePortsFunction

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

func TestFixNullNodePortsFilter(t *testing.T) {
	var tests = []struct {
		name   string
		input  string
		output string
	}{
		{
			name: "null-node-port",
			input: `apiVersion: v1
kind: Service
metadata:
  name: test
spec:
  ports:
  - name: http
    port: 8080
    nodePort: null
  selector:
    app: name
`,
			output: `apiVersion: v1
kind: Service
metadata:
  name: test
spec:
  ports:
  - name: http
    port: 8080
  selector:
    app: name
`,
		},
		{
			name: "multiple-null-node-ports",
			input: `apiVersion: v1
kind: Service
metadata:
  name: test
spec:
  ports:
  - name: http
    port: 8080
    nodePort: null
  - name: metrics
    port: 9090
    nodePort: null
  selector:
    app: name
`,
			output: `apiVersion: v1
kind: Service
metadata:
  name: test
spec:
  ports:
  - name: http
    port: 8080
  - name: metrics
    port: 9090
  selector:
    app: name
`,
		},
		{
			name: "no-null-node-port",
			input: `apiVersion: v1
kind: Service
metadata:
  name: test
spec:
  ports:
  - name: http
    port: 8080
  selector:
    app: name
`,
			output: `apiVersion: v1
kind: Service
metadata:
  name: test
spec:
  ports:
  - name: http
    port: 8080
  selector:
    app: name
`,
		},
		{
			name: "no-ports",
			input: `apiVersion: v1
kind: Service
metadata:
  name: test
spec:
  selector:
    app: name
`,
			output: `apiVersion: v1
kind: Service
metadata:
  name: test
spec:
  selector:
    app: name
`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var fn FixNullNodePortsFunction

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
