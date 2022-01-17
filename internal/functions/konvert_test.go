package functions

import (
	"bytes"
	"fmt"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestKonvertProcess(t *testing.T) {
	// this tests functionConfig discovery
	// - provided by the framework
	// - included in input
	// - or not found at all (error)
	var tests = []struct {
		name           string
		input          string
		functionConfig string
		expectedError  string
	}{
		{
			name:  "has-function-config",
			input: "",
			functionConfig: `apiVersion: konvert.kumorilabs.io/v1alpha1
kind: Konvert
metadata:
  name: fnconfig
spec:
  chart: mysql
  repo: https://charts.bitnami.com/bitnami
  version: 8.6.2
  namespace: mysql
  path: "upstream"
  pattern: "%s_%s.yaml"
  kustomize: true
  values:
    architecture: standalone
    image:
      pullPolicy: Always
      debug: true
`,
		},
		{
			name: "has-function-config-in-input",
			input: `apiVersion: v1
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
---
apiVersion: konvert.kumorilabs.io/v1alpha1
kind: Konvert
metadata:
  name: fnconfig
spec:
  chart: mysql
  repo: https://charts.bitnami.com/bitnami
  version: 8.6.2
  namespace: mysql
  path: "upstream"
  pattern: "%s_%s.yaml"
  kustomize: true
  values:
    architecture: standalone
    image:
      pullPolicy: Always
      debug: true
`,
			functionConfig: "",
		},
		{
			name: "no-function-config",
			input: `apiVersion: v1
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
			functionConfig: "",
			expectedError:  "failed to configure function",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			items, err := kio.ParseAll(test.input)
			if !assert.NoError(t, err, test.name) {
				t.FailNow()
			}

			var fnconfig *kyaml.RNode
			if test.functionConfig != "" {
				fnconfig, err = kyaml.Parse(test.functionConfig)
				if !assert.NoError(t, err, test.name) {
					t.FailNow()
				}
			}

			reslist := &framework.ResourceList{
				Items:          items,
				FunctionConfig: fnconfig,
			}
			processor := &KonvertProcessor{}
			err = processor.Process(reslist)

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

func TestKonvertFunctionConfig(t *testing.T) {
	var tests = []struct {
		name              string
		input             string
		expectedRepo      string
		expectedChart     string
		expectedVersion   string
		expectedNamespace string
		expectedPath      string
		expectedPattern   string
		expectedKustomize bool
		expectedValues    map[string]interface{}
		expectedError     string
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
  path: "upstream"
  pattern: "%s_%s.yaml"
  kustomize: true
  values:
    architecture: standalone
    image:
      pullPolicy: Always
      debug: true
`,
			expectedRepo:      "https://charts.bitnami.com/bitnami",
			expectedChart:     "mysql",
			expectedVersion:   "8.6.2",
			expectedNamespace: "mysql",
			expectedPath:      "upstream",
			expectedPattern:   "%s_%s.yaml",
			expectedKustomize: true,
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
kind: Konvert
metadata:
  name: fnconfig
spec:
  chart: mysql
  repo: https://charts.bitnami.com/bitnami
  version: 8.6.2
  namespace: mysql
  path: "upstream"
  pattern: "%s_%s.yaml"
  kustomize: true
  values:
    architecture: standalone
    image:
      pullPolicy: Always
      debug: true
`,
			expectedRepo:      "https://charts.bitnami.com/bitnami",
			expectedChart:     "mysql",
			expectedVersion:   "8.6.2",
			expectedNamespace: "mysql",
			expectedPath:      "upstream",
			expectedPattern:   "%s_%s.yaml",
			expectedKustomize: true,
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
kind: Konvert
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
			expectedError: "`functionConfig` must be a `ConfigMap` or `Konvert`",
		},
		{
			name: "bad-yaml-spec",
			input: `apiVersion: konvert.kumorilabs.io/v1alpha1
kind: Konvert
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
			var fn KonvertFunction

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
			assert.Equal(t, test.expectedVersion, fn.Version, test.name)
			assert.Equal(t, test.expectedNamespace, fn.Namespace, test.name)
			assert.Equal(t, test.expectedPath, fn.Path, test.name)
			assert.Equal(t, test.expectedPattern, fn.Pattern, test.name)
			assert.Equal(t, test.expectedKustomize, fn.Kustomize, test.name)
			assert.Equal(t, test.expectedValues, fn.Values, test.name)
		})
	}
}

func TestKonvertFilter(t *testing.T) {
	var tests = []struct {
		name          string
		repo          string
		chart         string
		version       string
		path          string
		namespace     string
		kustomize     bool
		values        map[string]interface{}
		releaseName   string
		expectedError string
	}{
		{
			name:        "mysql",
			releaseName: "db01",
			repo:        "https://charts.bitnami.com/bitnami",
			chart:       "mysql",
			version:     "8.6.2",
			namespace:   "",
			kustomize:   true,
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
			version:   "4.0.13",
			path:      "upstream",
			kustomize: true,
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
			var fn KonvertFunction
			fn.ResourceMeta.Name = test.releaseName
			fn.Repo = test.repo
			fn.Chart = test.chart
			fn.Version = test.version
			fn.Namespace = test.namespace
			fn.Values = test.values
			fn.Path = test.path
			fn.Kustomize = test.kustomize

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
				outputbuf        bytes.Buffer
				fixturebuf       bytes.Buffer
				clearAnnotations = []string{
					kioutil.LegacyPathAnnotation,  //nolint:staticcheck
					kioutil.LegacyIndexAnnotation, //nolint:staticcheck
					kioutil.LegacyIdAnnotation,    //nolint:staticcheck
					kioutil.PathAnnotation,
				}
			)

			err = kio.Pipeline{
				Inputs: []kio.Reader{&kio.PackageBuffer{Nodes: output}},
				Outputs: []kio.Writer{
					kio.ByteWriter{
						Writer:           &outputbuf,
						Sort:             true,
						ClearAnnotations: clearAnnotations,
					},
				},
			}.Execute()
			if !assert.NoError(t, err, test.name) {
				t.FailNow()
			}

			err = kio.Pipeline{
				Inputs: []kio.Reader{
					kio.LocalPackageReader{
						PackagePath: fmt.Sprintf("./fixtures/konvert/%s-%s", test.chart, test.version),
					},
				},
				Outputs: []kio.Writer{
					kio.ByteWriter{
						Writer:           &fixturebuf,
						Sort:             true,
						ClearAnnotations: clearAnnotations,
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
				return fmt.Sprintf(
					"%s~%s/%s/%s",
					node.GetApiVersion(),
					node.GetKind(),
					node.GetNamespace(),
					node.GetName(),
				)
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
