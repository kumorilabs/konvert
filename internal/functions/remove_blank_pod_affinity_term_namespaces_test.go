package functions

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/kio"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestRemoveBlankPodAffinityTermNamespacesFunctionConfig(t *testing.T) {
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
kind: RemoveBlankPodAffinityTermNamespaces
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
			expectedError: "`functionConfig` must be a `ConfigMap` or `RemoveBlankPodAffinityTermNamespaces`",
		},
		{
			name: "bad-yaml-spec",
			input: `apiVersion: konvert.kumorilabs.io/v1alpha1
kind: RemoveBlankPodAffinityTermNamespaces
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
			var fn RemoveBlankPodAffinityTermNamespacesFunction

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

func TestRemoveBlankPodAffinityTermNamespacesFilter(t *testing.T) {
	var tests = []struct {
		name   string
		input  string
		output string
	}{
		{
			name: "pod-anti-affinity-preferred",
			input: `apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: mysql
spec:
  template:
    spec:
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
          - podAffinityTerm:
              labelSelector:
                matchLabels:
                  app.kubernetes.io/name: mysql
              namespaces:
              - ""
              topologyKey: kubernetes.io/hostname
            weight: 1
      containers:
      - name: mysql
        image: mysql
`,
			output: `apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: mysql
spec:
  template:
    spec:
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
          - podAffinityTerm:
              labelSelector:
                matchLabels:
                  app.kubernetes.io/name: mysql
              topologyKey: kubernetes.io/hostname
            weight: 1
      containers:
      - name: mysql
        image: mysql
`,
		},
		{
			name: "pod-anti-affinity-required",
			input: `apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: mysql
spec:
  template:
    spec:
      affinity:
        podAntiAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
          - labelSelector:
              matchLabels:
                app.kubernetes.io/name: mysql
            namespaces:
            - ""
            topologyKey: kubernetes.io/hostname
      containers:
      - name: mysql
        image: mysql
`,
			output: `apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: mysql
spec:
  template:
    spec:
      affinity:
        podAntiAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
          - labelSelector:
              matchLabels:
                app.kubernetes.io/name: mysql
            topologyKey: kubernetes.io/hostname
      containers:
      - name: mysql
        image: mysql
`,
		},
		{
			name: "pod-affinity-preferred",
			input: `apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: mysql
spec:
  template:
    spec:
      affinity:
        podAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
          - podAffinityTerm:
              labelSelector:
                matchLabels:
                  app.kubernetes.io/name: mysql
              namespaces:
              - ""
              topologyKey: kubernetes.io/hostname
            weight: 1
      containers:
      - name: mysql
        image: mysql
`,
			output: `apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: mysql
spec:
  template:
    spec:
      affinity:
        podAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
          - podAffinityTerm:
              labelSelector:
                matchLabels:
                  app.kubernetes.io/name: mysql
              topologyKey: kubernetes.io/hostname
            weight: 1
      containers:
      - name: mysql
        image: mysql
`,
		},
		{
			name: "pod-affinity-required",
			input: `apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: mysql
spec:
  template:
    spec:
      affinity:
        podAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
          - labelSelector:
              matchLabels:
                app.kubernetes.io/name: mysql
            namespaces:
            - ""
            topologyKey: kubernetes.io/hostname
      containers:
      - name: mysql
        image: mysql
`,
			output: `apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: mysql
spec:
  template:
    spec:
      affinity:
        podAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
          - labelSelector:
              matchLabels:
                app.kubernetes.io/name: mysql
            topologyKey: kubernetes.io/hostname
      containers:
      - name: mysql
        image: mysql
`,
		},
		{
			name: "multiple-terms",
			input: `apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: mysql
spec:
  template:
    spec:
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
          - podAffinityTerm:
              labelSelector:
                matchLabels:
                  app.kubernetes.io/name: mysql
              namespaces:
              - ""
              topologyKey: kubernetes.io/hostname
            weight: 1
          - podAffinityTerm:
              labelSelector:
                matchLabels:
                  app.kubernetes.io/name: mysql2
              namespaces:
              - ""
              topologyKey: kubernetes.io/hostname
            weight: 2
      containers:
      - name: mysql
        image: mysql
`,
			output: `apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: mysql
spec:
  template:
    spec:
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
          - podAffinityTerm:
              labelSelector:
                matchLabels:
                  app.kubernetes.io/name: mysql
              topologyKey: kubernetes.io/hostname
            weight: 1
          - podAffinityTerm:
              labelSelector:
                matchLabels:
                  app.kubernetes.io/name: mysql2
              topologyKey: kubernetes.io/hostname
            weight: 2
      containers:
      - name: mysql
        image: mysql
`,
		},
		{
			name: "multiple-namespaces",
			input: `apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: mysql
spec:
  template:
    spec:
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
          - podAffinityTerm:
              labelSelector:
                matchLabels:
                  app.kubernetes.io/name: mysql
              namespaces:
              - ""
              - my-namespace
              topologyKey: kubernetes.io/hostname
            weight: 1
      containers:
      - name: mysql
        image: mysql
`,
			output: `apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: mysql
spec:
  template:
    spec:
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
          - podAffinityTerm:
              labelSelector:
                matchLabels:
                  app.kubernetes.io/name: mysql
              namespaces:
              - my-namespace
              topologyKey: kubernetes.io/hostname
            weight: 1
      containers:
      - name: mysql
        image: mysql
`,
		},
		{
			name: "pod",
			input: `apiVersion: v1
kind: Pod
metadata:
  name: nginx
spec:
  affinity:
    podAntiAffinity:
      preferredDuringSchedulingIgnoredDuringExecution:
      - podAffinityTerm:
          labelSelector:
            matchLabels:
              app.kubernetes.io/name: nginx
          namespaces:
          - ""
          topologyKey: kubernetes.io/hostname
        weight: 1
  containers:
  - name: nginx
    image: nginx
`,
			output: `apiVersion: v1
kind: Pod
metadata:
  name: nginx
spec:
  affinity:
    podAntiAffinity:
      preferredDuringSchedulingIgnoredDuringExecution:
      - podAffinityTerm:
          labelSelector:
            matchLabels:
              app.kubernetes.io/name: nginx
          topologyKey: kubernetes.io/hostname
        weight: 1
  containers:
  - name: nginx
    image: nginx
`,
		},
		{
			name: "cronjob",
			input: `apiVersion: batch/v1
kind: CronJob
metadata:
  name: hello
spec:
  schedule: "*/1 * * * *"
  jobTemplate:
    spec:
      template:
        spec:
          affinity:
            podAntiAffinity:
              preferredDuringSchedulingIgnoredDuringExecution:
              - podAffinityTerm:
                  labelSelector:
                    matchLabels:
                      app.kubernetes.io/name: hello
                  namespaces:
                  - ""
                  topologyKey: kubernetes.io/hostname
                weight: 1
            nodeAffinity:
          containers:
          - name: hello
            image: busybox
            command:
            - /bin/sh
            - -c
            - date; echo Hello from the Kubernetes cluster
          restartPolicy: OnFailure
`,
			output: `apiVersion: batch/v1
kind: CronJob
metadata:
  name: hello
spec:
  schedule: "*/1 * * * *"
  jobTemplate:
    spec:
      template:
        spec:
          affinity:
            podAntiAffinity:
              preferredDuringSchedulingIgnoredDuringExecution:
              - podAffinityTerm:
                  labelSelector:
                    matchLabels:
                      app.kubernetes.io/name: hello
                  topologyKey: kubernetes.io/hostname
                weight: 1
            nodeAffinity:
          containers:
          - name: hello
            image: busybox
            command:
            - /bin/sh
            - -c
            - date; echo Hello from the Kubernetes cluster
          restartPolicy: OnFailure
`,
		},
		{
			name: "multiple-resources",
			input: `apiVersion: batch/v1
kind: CronJob
metadata:
  name: hello
spec:
  schedule: "*/1 * * * *"
  jobTemplate:
    spec:
      template:
        spec:
          affinity:
            podAntiAffinity:
              preferredDuringSchedulingIgnoredDuringExecution:
              - podAffinityTerm:
                  labelSelector:
                    matchLabels:
                      app.kubernetes.io/name: hello
                  namespaces:
                  - ""
                  topologyKey: kubernetes.io/hostname
                weight: 1
            nodeAffinity:
          containers:
          - name: hello
            image: busybox
            command:
            - /bin/sh
            - -c
            - date; echo Hello from the Kubernetes cluster
          restartPolicy: OnFailure
---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: mysql
spec:
  template:
    spec:
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
          - podAffinityTerm:
              labelSelector:
                matchLabels:
                  app.kubernetes.io/name: mysql
              namespaces:
              - ""
              topologyKey: kubernetes.io/hostname
            weight: 1
      containers:
      - name: mysql
        image: mysql
---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: mysql2
spec:
  template:
    spec:
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
          - podAffinityTerm:
              labelSelector:
                matchLabels:
                  app.kubernetes.io/name: mysql
              namespaces:
              - ""
              - my-namespace
              topologyKey: kubernetes.io/hostname
            weight: 1
      containers:
      - name: mysql
        image: mysql
---
apiVersion: v1
kind: Pod
metadata:
  name: nginx
spec:
  affinity:
    podAntiAffinity:
      preferredDuringSchedulingIgnoredDuringExecution:
      - podAffinityTerm:
          labelSelector:
            matchLabels:
              app.kubernetes.io/name: nginx
          namespaces:
          - ""
          topologyKey: kubernetes.io/hostname
        weight: 1
  containers:
  - name: nginx
    image: nginx
`,
			output: `apiVersion: batch/v1
kind: CronJob
metadata:
  name: hello
spec:
  schedule: "*/1 * * * *"
  jobTemplate:
    spec:
      template:
        spec:
          affinity:
            podAntiAffinity:
              preferredDuringSchedulingIgnoredDuringExecution:
              - podAffinityTerm:
                  labelSelector:
                    matchLabels:
                      app.kubernetes.io/name: hello
                  topologyKey: kubernetes.io/hostname
                weight: 1
            nodeAffinity:
          containers:
          - name: hello
            image: busybox
            command:
            - /bin/sh
            - -c
            - date; echo Hello from the Kubernetes cluster
          restartPolicy: OnFailure
---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: mysql
spec:
  template:
    spec:
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
          - podAffinityTerm:
              labelSelector:
                matchLabels:
                  app.kubernetes.io/name: mysql
              topologyKey: kubernetes.io/hostname
            weight: 1
      containers:
      - name: mysql
        image: mysql
---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: mysql2
spec:
  template:
    spec:
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
          - podAffinityTerm:
              labelSelector:
                matchLabels:
                  app.kubernetes.io/name: mysql
              namespaces:
              - my-namespace
              topologyKey: kubernetes.io/hostname
            weight: 1
      containers:
      - name: mysql
        image: mysql
---
apiVersion: v1
kind: Pod
metadata:
  name: nginx
spec:
  affinity:
    podAntiAffinity:
      preferredDuringSchedulingIgnoredDuringExecution:
      - podAffinityTerm:
          labelSelector:
            matchLabels:
              app.kubernetes.io/name: nginx
          topologyKey: kubernetes.io/hostname
        weight: 1
  containers:
  - name: nginx
    image: nginx
`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var fn RemoveBlankPodAffinityTermNamespacesFunction

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
