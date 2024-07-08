package functions

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/kio"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestSetKonvertAnnotationsFunctionConfig(t *testing.T) {
	var tests = []struct {
		name          string
		input         string
		expectedChart string
		expectedRepo  string
		expectedError string
	}{
		{
			name: "configmap",
			input: `apiVersion: v1
kind: ConfigMap
metadata:
  name: test
data:
  chart: mysql
  repo: https://charts.bitnami.com/bitnami
`,
			expectedChart: "mysql",
			expectedRepo:  "https://charts.bitnami.com/bitnami",
		},
		{
			name: "empty-configmap",
			input: `apiVersion: v1
kind: ConfigMap
metadata:
  name: test
`,
			expectedRepo:  "",
			expectedChart: "",
		},
		{
			name: "function-config",
			input: `apiVersion: konvert.kumorilabs.io/v1alpha1
kind: SetKonvertAnnotations
metadata:
  name: fnconfig
spec:
  chart: mysql
  repo: https://charts.bitnami.com/bitnami
`,
			expectedChart: "mysql",
			expectedRepo:  "https://charts.bitnami.com/bitnami",
		},
		{
			name: "empty-function-config",
			input: `apiVersion: konvert.kumorilabs.io/v1alpha1
kind: SetKonvertAnnotations
metadata:
  name: fnconfig
`,
			expectedRepo:  "",
			expectedChart: "",
		},
		{
			name: "invalid-gvk",
			input: `apiVersion: v1
kind: Secret
metadata:
  name: bad-gvk
`,
			expectedError: "`functionConfig` must be a `ConfigMap` or `SetKonvertAnnotations`",
		},
		{
			name: "bad-yaml-spec",
			input: `apiVersion: konvert.kumorilabs.io/v1alpha1
kind: SetKonvertAnnotations
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
			var fn SetKonvertAnnotationsFunction

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

			assert.Equal(t, test.expectedRepo, fn.Repo, test.name)
			assert.Equal(t, test.expectedChart, fn.Chart, test.name)
		})
	}
}

func TestSetKonvertAnnotationsFilter(t *testing.T) {
	var tests = []struct {
		name                    string
		input                   string
		chart                   string
		repo                    string
		expectedChartAnnotation string
		expectedError           string
	}{
		{
			name: "simple",
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
			chart:                   "mysql",
			repo:                    "https://charts.bitnami.com/bitnami",
			expectedChartAnnotation: "https://charts.bitnami.com/bitnami,mysql",
		},
		{
			name: "missing-chart",
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
			repo:          "https://charts.bitnami.com/bitnami",
			expectedError: "chart cannot be empty",
		},
		{
			name: "local-chart-directory",
			input: `
apiVersion: v1
kind: Service
metadata:
  name: test
  metadata:
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
			chart:                   "../charts/mysql",
			expectedChartAnnotation: "../charts/mysql",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var fn SetKonvertAnnotationsFunction
			fn.Chart = test.chart
			fn.Repo = test.repo

			input, err := kio.ParseAll(test.input)
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			output, err := fn.Filter(input)

			if test.expectedError != "" {
				assert.NotNil(t, err, test.name)
				assert.Contains(t, err.Error(), test.expectedError, test.name)
				return
			}

			if !assert.NoError(t, err, test.name) {
				t.FailNow()
			}

			for _, onode := range output {
				annotations := onode.GetAnnotations()

				generatedBy, ok := annotations[annotationKonvertGeneratedBy]
				if !ok {
					assert.Fail(t, "resource is missing generated-by annotation", onode.GetName(), test.name)
				}

				chartAnno, ok := annotations[annotationKonvertChart]
				if !ok {
					assert.Fail(t, "resource is missing chart annotation", onode.GetName(), test.name)
				}

				assert.Equal(t, defaultGeneratedBy, generatedBy, test.name)
				assert.Equal(t, test.expectedChartAnnotation, chartAnno, test.name)
			}
		})
	}
}
