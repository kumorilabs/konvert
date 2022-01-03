package functions

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/kio"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestRemoveBlankAffinitiesFunctionConfig(t *testing.T) {
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
kind: RemoveBlankAffinities
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
			expectedError: "`functionConfig` must be a `ConfigMap` or `RemoveBlankAffinities`",
		},
		{
			name: "bad-yaml-spec",
			input: `apiVersion: konvert.kumorilabs.io/v1alpha1
kind: RemoveBlankAffinities
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
			var fn RemoveBlankAffinitiesFunction

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

func TestRemoveBlankAffinityFilter(t *testing.T) {
	var tests = []struct {
		name   string
		input  string
		output string
	}{
		{
			name: "pod-all-empty",
			input: `apiVersion: v1
kind: Pod
metadata:
  name: nginx
spec:
  affinity:
    podAffinity:
    podAntiAffinity: {}
    nodeAffinity: null
  containers:
  - name: nginx
    image: nginx:1.14.2
    ports:
    - containerPort: 80
`,
			output: `apiVersion: v1
kind: Pod
metadata:
  name: nginx
spec:
  containers:
  - name: nginx
    image: nginx:1.14.2
    ports:
    - containerPort: 80
`,
		},
		{
			name: "pod-empty-pod-affinity",
			input: `apiVersion: v1
kind: Pod
metadata:
  name: nginx
spec:
  affinity:
    podAffinity:
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
    image: nginx:1.14.2
    ports:
    - containerPort: 80
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
    image: nginx:1.14.2
    ports:
    - containerPort: 80
`,
		},
		{
			name: "pod-empty-pod-anti-affinity",
			input: `apiVersion: v1
kind: Pod
metadata:
  name: with-node-affinity
spec:
  affinity:
    podAntiAffinity:
    nodeAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
        nodeSelectorTerms:
        - matchExpressions:
          - key: kubernetes.io/e2e-az-name
            operator: In
            values:
            - e2e-az1
            - e2e-az2
      preferredDuringSchedulingIgnoredDuringExecution:
      - weight: 1
        preference:
          matchExpressions:
          - key: another-node-label-key
            operator: In
            values:
            - another-node-label-value
  containers:
  - name: with-node-affinity
    image: k8s.gcr.io/pause:2.0
`,
			output: `apiVersion: v1
kind: Pod
metadata:
  name: with-node-affinity
spec:
  affinity:
    nodeAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
        nodeSelectorTerms:
        - matchExpressions:
          - key: kubernetes.io/e2e-az-name
            operator: In
            values:
            - e2e-az1
            - e2e-az2
      preferredDuringSchedulingIgnoredDuringExecution:
      - weight: 1
        preference:
          matchExpressions:
          - key: another-node-label-key
            operator: In
            values:
            - another-node-label-value
  containers:
  - name: with-node-affinity
    image: k8s.gcr.io/pause:2.0
`,
		},
		{
			name: "deployment",
			input: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: redis-cache
spec:
  selector:
    matchLabels:
      app: store
  replicas: 3
  template:
    metadata:
      labels:
        app: store
    spec:
      affinity:
        nodeAffinity:
        podAntiAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
          - labelSelector:
              matchExpressions:
              - key: app
                operator: In
                values:
                - store
            topologyKey: "kubernetes.io/hostname"
      containers:
      - name: redis-server
        image: redis:3.2-alpine
`,
			output: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: redis-cache
spec:
  selector:
    matchLabels:
      app: store
  replicas: 3
  template:
    metadata:
      labels:
        app: store
    spec:
      affinity:
        podAntiAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
          - labelSelector:
              matchExpressions:
              - key: app
                operator: In
                values:
                - store
            topologyKey: "kubernetes.io/hostname"
      containers:
      - name: redis-server
        image: redis:3.2-alpine
`,
		},
		{
			name: "statefulset",
			input: `apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: mysql
spec:
  template:
    spec:
      affinity:
        podAffinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
          - podAffinityTerm:
              labelSelector:
                matchLabels:
                  app.kubernetes.io/name: mysql
              topologyKey: kubernetes.io/hostname
            weight: 1
        nodeAffinity:
      containers:
      - name: mysql
        image: docker.io/bitnami/mysql
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
        image: docker.io/bitnami/mysql
`,
		},
		{
			name: "job",
			input: `apiVersion: batch/v1
kind: Job
metadata:
  name: pi
spec:
  template:
    spec:
      affinity:
        podAffinity: null
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
          - podAffinityTerm:
              labelSelector:
                matchLabels:
                  app.kubernetes.io/name: pi
              topologyKey: kubernetes.io/hostname
            weight: 1
      containers:
      - name: pi
        image: perl
        command: ["perl", "-Mbignum=bpi", "-wle", "print bpi(2000)"]
      restartPolicy: Never
  backoffLimit: 4
`,
			output: `apiVersion: batch/v1
kind: Job
metadata:
  name: pi
spec:
  template:
    spec:
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
          - podAffinityTerm:
              labelSelector:
                matchLabels:
                  app.kubernetes.io/name: pi
              topologyKey: kubernetes.io/hostname
            weight: 1
      containers:
      - name: pi
        image: perl
        command: ["perl", "-Mbignum=bpi", "-wle", "print bpi(2000)"]
      restartPolicy: Never
  backoffLimit: 4
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
            podAffinity:
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
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var fn RemoveBlankAffinity

			input, err := kyaml.Parse(test.input)
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			output, err := fn.Filter(input)
			if !assert.NoError(t, err, test.name) {
				t.FailNow()
			}

			outputstr, err := output.String()
			if !assert.NoError(t, err, test.name) {
				t.FailNow()
			}

			assert.Equal(t, test.output, outputstr, test.name)
		})
	}
}

func TestRemoveBlankAffinitiesFilter(t *testing.T) {
	var tests = []struct {
		name   string
		input  string
		output string
	}{
		{
			name: "filter-all",
			input: `apiVersion: v1
kind: Pod
metadata:
  name: nginx
spec:
  affinity:
    podAffinity:
    podAntiAffinity: {}
    nodeAffinity: null
  containers:
  - name: nginx
    image: nginx:1.14.2
    ports:
    - containerPort: 80
---
apiVersion: batch/v1
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
            nodeAffinity:
            podAntiAffinity:
              preferredDuringSchedulingIgnoredDuringExecution:
              - podAffinityTerm:
                  labelSelector:
                    matchLabels:
                      app.kubernetes.io/name: hello
                  topologyKey: kubernetes.io/hostname
                weight: 1
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
kind: Deployment
metadata:
  name: redis-cache
spec:
  selector:
    matchLabels:
      app: store
  replicas: 3
  template:
    metadata:
      labels:
        app: store
    spec:
      affinity:
        nodeAffinity:
        podAntiAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
          - labelSelector:
              matchExpressions:
              - key: app
                operator: In
                values:
                - store
            topologyKey: "kubernetes.io/hostname"
      containers:
      - name: redis-server
        image: redis:3.2-alpine
---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: mysql
spec:
  template:
    spec:
      affinity:
        podAffinity:
        nodeAffinity:
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
        image: docker.io/bitnami/mysql
`,
			output: `apiVersion: v1
kind: Pod
metadata:
  name: nginx
spec:
  containers:
  - name: nginx
    image: nginx:1.14.2
    ports:
    - containerPort: 80
---
apiVersion: batch/v1
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
kind: Deployment
metadata:
  name: redis-cache
spec:
  selector:
    matchLabels:
      app: store
  replicas: 3
  template:
    metadata:
      labels:
        app: store
    spec:
      affinity:
        podAntiAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
          - labelSelector:
              matchExpressions:
              - key: app
                operator: In
                values:
                - store
            topologyKey: "kubernetes.io/hostname"
      containers:
      - name: redis-server
        image: redis:3.2-alpine
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
        image: docker.io/bitnami/mysql
`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var fn RemoveBlankAffinitiesFunction

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
