Konvert ![Build](https://github.com/kumorilabs/konvert/actions/workflows/ci.yaml/badge.svg) [![Go Report Card](https://goreportcard.com/badge/github.com/kumorilabs/konvert)](https://goreportcard.com/report/github.com/kumorilabs/konvert)
=

`konvert` renders [Helm](https://helm.sh/) charts into [Kustomize](https://kustomize.io/) bases (or plain Kubernetes manifests).

`konvert` allows you to take advantage of the Helm chart ecosystem while:

* Expanding your ability to customize and configure your deployments in ways not supported by the underlying Helm chart
* Unifying your [GitOps](https://www.weave.works/technologies/gitops/) pipelines with plain Kubernetes manifests (instead of a hybrid mix of Helm charts and manifests)
* Facilitating maintainable and easily upgradable packages of deployment manifests

`konvert` is similar to the builtin Kustomize [HelmChartInflationGenerator]( https://kubectl.docs.kubernetes.io/references/kustomize/builtins/#_helmchartinflationgenerator_) generator or the [Kpt](https://kpt.dev/) [render-helm-chart function](https://catalog.kpt.dev/render-helm-chart/v0.1/) but with additional capabilities.

## Features

* Render Helm charts to [Kustomize](https://kustomize.io/) bases
* Render Helm charts to plain Kubernetes manifests
* Set a namespace for all resources
* Render a chart into sub-directories
* Configured declaratively
* Enable easy configuration changes or upgrades, especially when used in conjunction with git
* Run as a standalone binary, docker image, or as a [KRM function](https://github.com/kubernetes-sigs/kustomize/blob/master/cmd/config/docs/api-conventions/functions-spec.md) via [Kpt](https://kpt.dev/)

## Usage

We will use the following `konvert` configuration for [cert-manager](https://cert-manager.io/) in the following examples.

``` shell
mkdir cert-manager

cat << EOF > cert-manager/konvert.yaml
apiVersion: konvert.kumorilabs.io/v1alpha1
kind: Konvert
metadata:
  name: cert-manager
  annotations:
    config.kubernetes.io/local-config: "true"
spec:
  kustomize: true
  repo: https://charts.jetstack.io
  chart: cert-manager
  version: v1.6.1
  namespace: cert-manager
  path: upstream
  values:
    installCRDs: true
    global:
      leaderElection:
        namespace: cert-manager
EOF
```

All of the examples below (except the Kustomize generator plugin) will produce the same result: the Helm chart with the specified configuration rendered to your local disk. You can use git to manage these manifests and use standard Kustomize to further modify them for your environments without worrying about changing the upstream source. Upgrading the chart is as simple as changing the version in the configuration and running `konvert` again. Since the manifests exist in your source code repository, reconciling upstream changes is easily accomplished with a pure git workflow.

The Kustomize generator plugin behaves slightly differently as the chart will render when running `kustomize build` and will not persist to disk.

* [Binary](#binary)
* [Kpt Function](#kpt-function)
* [Container Image](#container-image)
* [Kustomize Generator Plugin](#kustomize-generator-plugin)

### Binary

``` shell
konvert -f cert-manager
```

### Kpt Function

Because `kpt` currently does not [allow network access](https://kpt.dev/book/04-using-functions/02-imperative-function-execution?id=privileged-execution) when executing functions declaratively, you must use `kpt fn eval` if you are rendering a chart from a remote repository.

``` shell
kpt fn eval cert-manager --image ghcr.io/kumorilabs/krm-fn-konvert:latest --network --fn-config cert-manager/konvert.yaml
```

If you don't wish to use the container image, you can also execute the `konvert` function using the binary.

``` shell
kpt fn eval cert-manager --exec "konvert fn" --fn-config cert-manager/konvert.yaml

```

### Container Image

Using docker:

``` shell
docker run --rm -it -v "$(pwd)":/src ghcr.io/kumorilabs/konvert:latest -f src/cert-manager
```

### Kustomize Generator Plugin

Using `konvert` in a Kustomize generator does require a small configuration change. We need to annotate the Konvert file to specify how to run the function. Add the following to the metadata section of the cert-manager Konvert file:

``` yaml
annotations:
  config.kubernetes.io/function: |
    container:
      network: true
      image: ghcr.io/kumorilabs/krm-fn-konvert:latest
```

You also need a kustomization.yaml that specifies the generator configuration.

``` yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
metadata:
  name: kustomization
generators:
  - konvert.yaml
```

Finally, to render the chart with `konvert`, run:

``` shell
kustomize build cert-manager
```

## Konvert schema

`konvert` uses a client-side KRM resource to configure how to find and render a Helm chart.

Example:

``` yaml
apiVersion: konvert.kumorilabs.io/v1alpha1
kind: Konvert
metadata:
  name: mysql
spec:
  repo: https://charts.bitnami.com/bitnami
  chart: mysql
  version: 8.6.2
  namespace: mysql
  kustomize: true
  values:
    auth:
      username: admin
      password: password
      rootPassword: password
```


| Field          | Description
|----------------|---------------------------------------------------------------------------------------------------------------------------------------------
| `apiVersion`   | konvert.kumorilabs.io/v1alpha1
| `kind`         | Konvert
| **`metadata`** |
| `name`         | The release name used when rendering the Helm chart.
| **`spec`**     |
| `repo`         | The URL for the Helm chart repository.
| `chart`        | The name of the chart.
| `version`      | The version of the chart.
| `namespace`    | The namespace to use when rendering the chart. When kustomize is `true`, this will also configure the Kustomize namespace transformer. 
| `path`         | The path (relative to the Konvert file) in which to render the chart.
| `kustomize`    | If `true`, `konvert` will write a kustomization.yaml for the generated chart resources. If `path` is configured, it will write a kustomization.yaml including the rendered chart subdirectory at the same level as the Konvert file.
| `values`       | The configuration values to use when rendering the chart.

## Contributing

### Build

``` shell
make build
```

### Install

``` shell
make install
```

### Run Tests

``` shell
make test
```
