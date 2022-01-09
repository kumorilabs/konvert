package functions

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/kio"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestKustomizerFunctionConfig(t *testing.T) {
	var tests = []struct {
		name            string
		input           string
		path            string
		namespace       string
		annotationName  string
		annotationValue string
		expectedError   string
	}{
		{
			name: "configmap",
			input: `apiVersion: v1
kind: ConfigMap
metadata:
  name: test
data:
  path: upstream
  namespace: mysql
  resource_annotation_name: konvert.kumorilabs.io/chart
  resource_annotation_value: https://charts.bitnami.com/bitnami,mysql
`,
			path:            "upstream",
			namespace:       "mysql",
			annotationName:  "konvert.kumorilabs.io/chart",
			annotationValue: "https://charts.bitnami.com/bitnami,mysql",
		},
		{
			name: "function-config",
			input: `apiVersion: konvert.kumorilabs.io/v1alpha1
kind: Kustomizer
metadata:
  name: fnconfig
spec:
  path: upstream
  namespace: mysql
  resource_annotation_name: konvert.kumorilabs.io/chart
  resource_annotation_value: https://charts.bitnami.com/bitnami,mysql
`,
			path:            "upstream",
			namespace:       "mysql",
			annotationName:  "konvert.kumorilabs.io/chart",
			annotationValue: "https://charts.bitnami.com/bitnami,mysql",
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
kind: Kustomizer
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
			expectedError: "`functionConfig` must be a `ConfigMap` or `Kustomizer`",
		},
		{
			name: "bad-yaml-spec",
			input: `apiVersion: konvert.kumorilabs.io/v1alpha1
kind: Kustomizer
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
			var fn KustomizerFunction

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

			assert.Equal(t, test.path, fn.Path, test.name)
			assert.Equal(t, test.namespace, fn.Namespace, test.name)
			assert.Equal(t, test.annotationName, fn.ResourceAnnotationName, test.name)
			assert.Equal(t, test.annotationValue, fn.ResourceAnnotationValue, test.name)
		})
	}
}

func TestKustomizerFilter(t *testing.T) {
	var tests = []struct {
		name            string
		path            string
		namespace       string
		annotationName  string
		annotationValue string
		input           string
		kustomization   string
		expectedError   string
	}{
		{
			name:            "default",
			annotationName:  annotationKonvertChart,
			annotationValue: "https://charts.bitnami.com/bitnami,mysql",
			input: `apiVersion: v1
kind: Service
metadata:
  name: mysql
  labels:
    app.kubernetes.io/name: mysql
  annotations:
    internal.config.kubernetes.io/path: 'service-mysql.yaml'
    konvert.kumorilabs.io/chart: 'https://charts.bitnami.com/bitnami,mysql'
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
    internal.config.kubernetes.io/path: 'configmap-mysql.yaml'
    konvert.kumorilabs.io/chart: 'https://charts.bitnami.com/bitnami,mysql'
data:
  my.cnf: |2-

    [mysqld]
`,
			kustomization: `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
metadata:
  name: kustomization
  annotations:
    config.kubernetes.io/local-config: 'true'
    internal.config.kubernetes.io/path: kustomization.yaml
    config.kubernetes.io/path: 'kustomization.yaml'
resources:
- configmap-mysql.yaml # konvert.kumorilabs.io/chart: https://charts.bitnami.com/bitnami,mysql
- service-mysql.yaml # konvert.kumorilabs.io/chart: https://charts.bitnami.com/bitnami,mysql
`,
		},
		{
			name:            "with-namespace",
			namespace:       "mysql",
			annotationName:  annotationKonvertChart,
			annotationValue: "https://charts.bitnami.com/bitnami,mysql",
			input: `apiVersion: v1
kind: Service
metadata:
  name: mysql
  labels:
    app.kubernetes.io/name: mysql
  annotations:
    internal.config.kubernetes.io/path: 'service-mysql.yaml'
    konvert.kumorilabs.io/chart: 'https://charts.bitnami.com/bitnami,mysql'
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
    internal.config.kubernetes.io/path: 'configmap-mysql.yaml'
    konvert.kumorilabs.io/chart: 'https://charts.bitnami.com/bitnami,mysql'
data:
  my.cnf: |2-

    [mysqld]
`,
			kustomization: `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
metadata:
  name: kustomization
  annotations:
    config.kubernetes.io/local-config: 'true'
    internal.config.kubernetes.io/path: kustomization.yaml
    config.kubernetes.io/path: 'kustomization.yaml'
namespace: mysql
resources:
- configmap-mysql.yaml # konvert.kumorilabs.io/chart: https://charts.bitnami.com/bitnami,mysql
- service-mysql.yaml # konvert.kumorilabs.io/chart: https://charts.bitnami.com/bitnami,mysql
`,
		},
		{
			name:            "with-existing-kustomization",
			annotationName:  annotationKonvertChart,
			annotationValue: "https://charts.bitnami.com/bitnami,mysql",
			input: `apiVersion: v1
kind: Service
metadata:
  name: mysql
  labels:
    app.kubernetes.io/name: mysql
  annotations:
    internal.config.kubernetes.io/path: 'service-mysql.yaml'
    konvert.kumorilabs.io/chart: 'https://charts.bitnami.com/bitnami,mysql'
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
    internal.config.kubernetes.io/path: 'configmap-mysql.yaml'
    konvert.kumorilabs.io/chart: 'https://charts.bitnami.com/bitnami,mysql'
data:
  my.cnf: |2-

    [mysqld]
---
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
metadata:
  name: kustomization
  annotations:
    config.kubernetes.io/local-config: 'true'
    internal.config.kubernetes.io/path: kustomization.yaml
    config.kubernetes.io/path: 'kustomization.yaml'
resources:
- some-service.yaml
- some-secret.yaml
`,
			kustomization: `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
metadata:
  name: kustomization
  annotations:
    config.kubernetes.io/local-config: 'true'
    internal.config.kubernetes.io/path: kustomization.yaml
    config.kubernetes.io/path: 'kustomization.yaml'
resources:
- some-service.yaml
- some-secret.yaml
- configmap-mysql.yaml # konvert.kumorilabs.io/chart: https://charts.bitnami.com/bitnami,mysql
- service-mysql.yaml # konvert.kumorilabs.io/chart: https://charts.bitnami.com/bitnami,mysql
`,
		},
		{
			name:            "without-path-annotation",
			annotationName:  annotationKonvertChart,
			annotationValue: "https://charts.bitnami.com/bitnami,mysql",
			input: `apiVersion: v1
kind: Service
metadata:
  name: mysql
  labels:
    app.kubernetes.io/name: mysql
  annotations:
    konvert.kumorilabs.io/chart: 'https://charts.bitnami.com/bitnami,mysql'
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
    konvert.kumorilabs.io/chart: 'https://charts.bitnami.com/bitnami,mysql'
data:
  my.cnf: |2-

    [mysqld]
`,
			kustomization: `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
metadata:
  name: kustomization
  annotations:
    config.kubernetes.io/local-config: 'true'
    internal.config.kubernetes.io/path: kustomization.yaml
    config.kubernetes.io/path: 'kustomization.yaml'
resources: []
`,
		},
		{
			name:            "without-matching-konvert-annotation",
			annotationName:  annotationKonvertChart,
			annotationValue: "https://charts.bitnami.com/bitnami,mysql",
			input: `apiVersion: v1
kind: Service
metadata:
  name: mysql
  labels:
    app.kubernetes.io/name: mysql
  annotations:
    internal.config.kubernetes.io/path: 'service-mysql.yaml'
    konvert.kumorilabs.io/chart: 'https://charts.bitnami.com/bitnami,mysql'
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
    internal.config.kubernetes.io/path: 'configmap-mysql.yaml'
data:
  my.cnf: |2-

    [mysqld]
`,
			kustomization: `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
metadata:
  name: kustomization
  annotations:
    config.kubernetes.io/local-config: 'true'
    internal.config.kubernetes.io/path: kustomization.yaml
    config.kubernetes.io/path: 'kustomization.yaml'
resources:
- service-mysql.yaml # konvert.kumorilabs.io/chart: https://charts.bitnami.com/bitnami,mysql
`,
		},
		{
			name:            "with-kustomization-path",
			path:            "upstream/base",
			annotationName:  annotationKonvertChart,
			annotationValue: "https://charts.bitnami.com/bitnami,mysql",
			input: `apiVersion: v1
kind: Service
metadata:
  name: mysql
  labels:
    app.kubernetes.io/name: mysql
  annotations:
    internal.config.kubernetes.io/path: 'service-mysql.yaml'
    konvert.kumorilabs.io/chart: 'https://charts.bitnami.com/bitnami,mysql'
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
    internal.config.kubernetes.io/path: 'configmap-mysql.yaml'
    konvert.kumorilabs.io/chart: 'https://charts.bitnami.com/bitnami,mysql'
data:
  my.cnf: |2-

    [mysqld]
`,
			kustomization: `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
metadata:
  name: kustomization
  annotations:
    config.kubernetes.io/local-config: 'true'
    internal.config.kubernetes.io/path: upstream/base/kustomization.yaml
resources:
- configmap-mysql.yaml # konvert.kumorilabs.io/chart: https://charts.bitnami.com/bitnami,mysql
- service-mysql.yaml # konvert.kumorilabs.io/chart: https://charts.bitnami.com/bitnami,mysql
---
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
metadata:
  name: kustomization
  annotations:
    config.kubernetes.io/local-config: 'true'
    internal.config.kubernetes.io/path: kustomization.yaml
resources:
- upstream/base
`,
		},
		{
			name:            "empty-resource-annotation-name",
			annotationValue: "https://charts.bitnami.com/bitnami,mysql",
			expectedError:   "resource annotation name cannot be empty",
			input: `apiVersion: v1
kind: Service
metadata:
  name: mysql
  labels:
    app.kubernetes.io/name: mysql
  annotations:
    internal.config.kubernetes.io/path: 'service-mysql.yaml'
    konvert.kumorilabs.io/chart: 'https://charts.bitnami.com/bitnami,mysql'
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
    internal.config.kubernetes.io/path: 'configmap-mysql.yaml'
    konvert.kumorilabs.io/chart: 'https://charts.bitnami.com/bitnami,mysql'
data:
  my.cnf: |2-

    [mysqld]
`,
			kustomization: ``,
		},
		{
			name:           "empty-resource-annotation-name",
			annotationName: annotationKonvertChart,
			expectedError:  "resource annotation value cannot be empty",
			input: `apiVersion: v1
kind: Service
metadata:
  name: mysql
  labels:
    app.kubernetes.io/name: mysql
  annotations:
    internal.config.kubernetes.io/path: 'service-mysql.yaml'
    konvert.kumorilabs.io/chart: 'https://charts.bitnami.com/bitnami,mysql'
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
    internal.config.kubernetes.io/path: 'configmap-mysql.yaml'
    konvert.kumorilabs.io/chart: 'https://charts.bitnami.com/bitnami,mysql'
data:
  my.cnf: |2-

    [mysqld]
`,
			kustomization: ``,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var fn KustomizerFunction
			fn.Path = test.path
			fn.Namespace = test.namespace
			fn.ResourceAnnotationName = test.annotationName
			fn.ResourceAnnotationValue = test.annotationValue

			input, err := kio.ParseAll(test.input)
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			output, err := fn.Filter(input)
			if test.expectedError != "" {
				assert.NotNil(t, err, test.name)
				assert.Contains(t, err.Error(), test.expectedError, test.name)
				return
			} else {
				if !assert.NoError(t, err, test.name) {
					t.FailNow()
				}
			}

			kustomization, err := kustomizationFilter{}.Filter(output)
			if !assert.NoError(t, err, test.name) {
				t.FailNow()
			}

			kustomizationstr, err := kio.StringAll(kustomization)
			if !assert.NoError(t, err, test.name) {
				t.FailNow()
			}

			assert.Equal(t, test.kustomization, kustomizationstr, test.name)
		})
	}
}
