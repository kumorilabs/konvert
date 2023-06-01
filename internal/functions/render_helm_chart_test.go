package functions

import (
	"bytes"
	"fmt"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestRenderHelmChartFunctionConfig(t *testing.T) {
	var tests = []struct {
		name                string
		input               string
		expectedReleaseName string
		expectedRepo        string
		expectedChart       string
		expectedVersion     string
		expectedNamespace   string
		expectedValues      map[string]interface{}
		expectedError       string
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
  version: 8.6.2
  namespace: mysql
  releaseName: db01
  values:
    architecture: standalone
    image:
      pullPolicy: Always
      debug: true
`,
			expectedReleaseName: "db01",
			expectedRepo:        "https://charts.bitnami.com/bitnami",
			expectedChart:       "mysql",
			expectedVersion:     "8.6.2",
			expectedNamespace:   "mysql",
			expectedValues: map[string]interface{}{
				"image": map[string]interface{}{
					"pullPolicy": "Always",
					"debug":      true,
				},
				"architecture": "standalone",
			},
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
			name: "function-config",
			input: `apiVersion: konvert.kumorilabs.io/v1alpha1
kind: RenderHelmChart
metadata:
  name: fnconfig
spec:
  releaseName: db01
  chart: mysql
  repo: https://charts.bitnami.com/bitnami
  version: 8.6.2
  namespace: mysql
  values:
    architecture: standalone
    image:
      pullPolicy: Always
      debug: true
`,
			expectedReleaseName: "db01",
			expectedRepo:        "https://charts.bitnami.com/bitnami",
			expectedChart:       "mysql",
			expectedVersion:     "8.6.2",
			expectedNamespace:   "mysql",
			expectedValues: map[string]interface{}{
				"image": map[string]interface{}{
					"pullPolicy": "Always",
					"debug":      true,
				},
				"architecture": "standalone",
			},
		},
		{
			name: "empty-function-config",
			input: `apiVersion: konvert.kumorilabs.io/v1alpha1
kind: RenderHelmChart
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
			expectedError: "`functionConfig` must be a `ConfigMap` or `RenderHelmChart`",
		},
		{
			name: "bad-yaml-spec",
			input: `apiVersion: konvert.kumorilabs.io/v1alpha1
kind: RenderHelmChart
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
			var fn RenderHelmChartFunction

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

			assert.Equal(t, test.expectedReleaseName, fn.ReleaseName, test.name)
			assert.Equal(t, test.expectedRepo, fn.Repo, test.name)
			assert.Equal(t, test.expectedChart, fn.Chart, test.name)
			assert.Equal(t, test.expectedVersion, fn.Version, test.name)
			assert.Equal(t, test.expectedNamespace, fn.Namespace, test.name)
			assert.Equal(t, test.expectedValues, fn.Values, test.name)
		})
	}
}

func TestRenderHelmChartFilter(t *testing.T) {
	var tests = []struct {
		name          string
		releaseName   string
		repo          string
		chart         string
		version       string
		namespace     string
		values        map[string]interface{}
		skipHooks     bool
		skipTests     bool
		expectedError string
	}{
		{
			name:        "mysql",
			releaseName: "db01",
			repo:        "https://charts.bitnami.com/bitnami",
			chart:       "mysql",
			version:     "9.10.1",
			namespace:   "mysql",
			values: map[string]interface{}{
				"auth": map[string]interface{}{
					"rootPassword": "password",
					"username":     "admin",
					"password":     "password",
				},
			},
		},
		{
			name:      "cluster-autoscaler",
			repo:      "https://kubernetes.github.io/autoscaler",
			chart:     "cluster-autoscaler",
			version:   "9.11.0",
			namespace: "cas",
		},
		{
			name:      "ingress-nginx",
			repo:      "https://kubernetes.github.io/ingress-nginx",
			chart:     "ingress-nginx",
			version:   "4.0.16",
			namespace: "ingress",
		},
		{
			name:      "ingress-nginx-no-hooks",
			repo:      "https://kubernetes.github.io/ingress-nginx",
			chart:     "ingress-nginx",
			version:   "4.0.16",
			namespace: "ingress",
			skipHooks: true,
		},
		{
			name:      "kong",
			repo:      "https://charts.konghq.com",
			chart:     "kong",
			version:   "2.6.4",
			namespace: "kong",
			skipTests: true,
		},
		{
			name:          "mysql",
			chart:         "mysql",
			version:       "8.6.2",
			namespace:     "mysql",
			expectedError: "repo cannot be empty",
		},
		{
			name:          "mysql",
			repo:          "https://charts.bitnami.com/bitnami",
			version:       "8.6.2",
			namespace:     "mysql",
			expectedError: "chart cannot be empty",
		},
		{
			name:          "mysql",
			repo:          "https://charts.bitnami.com/bitnami",
			chart:         "mysql",
			namespace:     "mysql",
			expectedError: "version cannot be empty",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var fn RenderHelmChartFunction
			fn.ReleaseName = test.releaseName
			fn.Repo = test.repo
			fn.Chart = test.chart
			fn.Version = test.version
			fn.Namespace = test.namespace
			fn.Values = test.values
			fn.SkipHooks = test.skipHooks
			fn.SkipTests = test.skipTests

			output, err := fn.Filter([]*kyaml.RNode{})
			if test.expectedError != "" {
				require.NotNil(t, err, test.name)
				assert.Contains(t, err.Error(), test.expectedError, test.name)
				return
			}

			if !assert.NoError(t, err, test.name) {
				t.FailNow()
			}

			var (
				outputbuf  bytes.Buffer
				fixturebuf bytes.Buffer
			)

			err = kio.Pipeline{
				Inputs: []kio.Reader{&kio.PackageBuffer{Nodes: output}},
				Outputs: []kio.Writer{
					kio.ByteWriter{
						Writer: &outputbuf,
						Sort:   true,
						ClearAnnotations: []string{
							kioutil.PathAnnotation,
							//lint:ignore SA1019 explicitly clearing legacy annotations that may have been added by framework
							kioutil.LegacyPathAnnotation, //nolint:staticcheck
						},
					},
				},
			}.Execute()
			if !assert.NoError(t, err, test.name) {
				t.FailNow()
			}

			err = kio.Pipeline{
				Inputs: []kio.Reader{
					kio.LocalPackageReader{
						PackagePath: fmt.Sprintf("./fixtures/render_helm_chart/%s-%s", test.name, test.version),
					},
				},
				Outputs: []kio.Writer{
					kio.ByteWriter{
						Writer: &fixturebuf,
						Sort:   true,
						ClearAnnotations: []string{
							kioutil.PathAnnotation,
							//lint:ignore SA1019 explicitly clearing legacy annotations that may have been added by framework
							kioutil.LegacyPathAnnotation, //nolint:staticcheck
						},
					},
				},
			}.Execute()
			if !assert.NoError(t, err, test.name) {
				t.FailNow()
			}

			// try to get a consistent sort
			// there is probably a cleaner way to do this but :shrug:
			outnodes, err := kio.ParseAll(outputbuf.String())
			if !assert.NoError(t, err, test.name) {
				t.FailNow()
			}
			fixnodes, err := kio.ParseAll(fixturebuf.String())
			if !assert.NoError(t, err, test.name) {
				t.FailNow()
			}

			resid := func(node *kyaml.RNode) string {
				return fmt.Sprintf("%s/%s~%s/%s", node.GetApiVersion(), node.GetKind(), node.GetNamespace(), node.GetName())
			}
			sort.Slice(outnodes, func(i, j int) bool {
				return resid(outnodes[i]) < resid(outnodes[j])
			})
			sort.Slice(fixnodes, func(i, j int) bool {
				return resid(fixnodes[i]) < resid(fixnodes[j])
			})

			outstr, err := kio.StringAll(outnodes)
			if !assert.NoError(t, err, test.name) {
				t.FailNow()
			}
			fixstr, err := kio.StringAll(fixnodes)
			if !assert.NoError(t, err, test.name) {
				t.FailNow()
			}
			assert.Equal(t, fixstr, outstr, test.name)
		})
	}
}
