<!--
Copyright The Shipwright Contributors

SPDX-License-Identifier: Apache-2.0
-->

# BuildStrategies

- [Overview](#overview)
- [Available ClusterBuildStrategies](#available-clusterbuildstrategies)
- [Available BuildStrategies](#available-buildstrategies)
- [Buildah](#buildah)
  - [Installing Buildah Strategy](#installing-buildah-strategy)
- [Buildpacks v3](#buildpacks-v3)
  - [Installing Buildpacks v3 Strategy](#installing-buildpacks-v3-strategy)
- [Kaniko](#kaniko)
  - [Installing Kaniko Strategy](#installing-kaniko-strategy)
  - [Scanning with Trivy](#scanning-with-trivy)
- [BuildKit](#buildkit)
  - [Cache Exporters](#cache-exporters)
  - [Known Limitations](#known-limitations)
  - [Usage in Clusters with Pod Security Standards](#usage-in-clusters-with-pod-security-standards)
  - [Installing BuildKit Strategy](#installing-buildkit-strategy)
- [ko](#ko)
  - [Installing ko Strategy](#installing-ko-strategy)
  - [Parameters](#parameters)
- [Source to Image](#source-to-image)
  - [Installing Source to Image Strategy](#installing-source-to-image-strategy)
  - [Build Steps](#build-steps)
- [Strategy parameters](#strategy-parameters)
- [System parameters](#system-parameters)
- [System parameters vs Strategy Parameters Comparison](#system-parameters-vs-strategy-parameters-comparison)
- [System results](#system-results)
- [Steps Resource Definition](#steps-resource-definition)
  - [Strategies with different resources](#strategies-with-different-resources)
  - [How does Tekton Pipelines handle resources](#how-does-tekton-pipelines-handle-resources)
  - [Examples of Tekton resources management](#examples-of-tekton-resources-management)
- [Annotations](#annotations)

## Overview

There are two types of strategies, the `ClusterBuildStrategy` (`clusterbuildstrategies.shipwright.io/v1alpha1`) and the `BuildStrategy` (`buildstrategies.shipwright.io/v1alpha1`). Both strategies define a shared group of steps, needed to fullfil the application build.

A `ClusterBuildStrategy` is available cluster-wide, while a `BuildStrategy` is available within a namespace.

## Available ClusterBuildStrategies

Well-known strategies can be bootstrapped from [here](../samples/buildstrategy). The currently supported Cluster BuildStrategy are:

| Name | Supported platforms |
| ---- | ------------------- |
| [buildah](../samples/buildstrategy/buildah/buildstrategy_buildah_cr.yaml) | linux/amd64 only |
| [BuildKit](../samples/buildstrategy/buildkit/buildstrategy_buildkit_cr.yaml) | all |
| [buildpacks-v3-heroku](../samples/buildstrategy/buildpacks-v3/buildstrategy_buildpacks-v3-heroku_cr.yaml) | linux/amd64 only |
| [buildpacks-v3](../samples/buildstrategy/buildpacks-v3/buildstrategy_buildpacks-v3_cr.yaml) | linux/amd64 only |
| [kaniko](../samples/buildstrategy/kaniko/buildstrategy_kaniko_cr.yaml) | all |
| [ko](../samples/buildstrategy/ko/buildstrategy_ko_cr.yaml) | all |
| [source-to-image](../samples/buildstrategy/source-to-image/buildstrategy_source-to-image_cr.yaml) | linux/amd64 only |

## Available BuildStrategies

The current supported namespaces BuildStrategy are:

| Name | Supported platforms |
| ---- | ------------------- |
| [buildpacks-v3-heroku](../samples/buildstrategy/buildpacks-v3/buildstrategy_buildpacks-v3-heroku_namespaced_cr.yaml) | linux/amd64 only |
| [buildpacks-v3](../samples/buildstrategy/buildpacks-v3/buildstrategy_buildpacks-v3_namespaced_cr.yaml) | linux/amd64 only |

---

## Buildah

The `buildah` ClusterBuildStrategy consists of using [`buildah`](https://github.com/containers/buildah) to build and push a container image, out of a `Dockerfile`. The `Dockerfile` should be specified on the `Build` resource.

### Installing Buildah Strategy

To install use:

```sh
kubectl apply -f samples/buildstrategy/buildah/buildstrategy_buildah_cr.yaml
```

---

## Buildpacks v3

The [buildpacks-v3][buildpacks] BuildStrategy/ClusterBuildStrategy uses a Cloud Native Builder ([CNB][cnb]) container image, and is able to implement [lifecycle commands][lifecycle]. The following CNB images are the most common options:

- [`heroku/buildpacks:18`][hubheroku]
- [`cloudfoundry/cnb:bionic`][hubcloudfoundry]
- [`docker.io/paketobuildpacks/builder:full`](https://hub.docker.com/r/paketobuildpacks/builder/tags)

### Installing Buildpacks v3 Strategy

You can install the `BuildStrategy` in your namespace or install the `ClusterBuildStrategy` at cluster scope so that it can be shared across namespaces.

To install the cluster scope strategy, use (below is a heroku example, you can also use paketo sample):

```sh
kubectl apply -f samples/buildstrategy/buildpacks-v3/buildstrategy_buildpacks-v3-heroku_cr.yaml
```

To install the namespaced scope strategy, use:

```sh
kubectl apply -f samples/buildstrategy/buildpacks-v3/buildstrategy_buildpacks-v3-heroku_namespaced_cr.yaml
```

---

## Kaniko

The `kaniko` ClusterBuildStrategy is composed by Kaniko's `executor` [kaniko], with the objective of building a container-image, out of a `Dockerfile` and context directory. The `kaniko-trivy` ClusterBuildStrategy adds [trivy](https://github.com/aquasecurity/trivy) scanning and refuses to push images with critical vulnerabilities.

### Installing Kaniko Strategy

To install the cluster scope strategy, use:

```sh
kubectl apply -f samples/buildstrategy/kaniko/buildstrategy_kaniko_cr.yaml
```

#### Scanning with Trivy

You can also incorporate scanning into the ClusterBuildStrategy. The `kaniko-trivy` ClusterBuildStrategy builds the image with `kaniko`, then scans with [trivy](https://github.com/aquasecurity/trivy). The BuildRun will then exit with an error if there is a critical vulnerability, instead of pushing the vulnerable image into the container registry.

To install the cluster scope strategy, use:

```sh
kubectl apply -f samples/buildstrategy/kaniko/buildstrategy_kaniko-trivy_cr.yaml
```

*Note: doing image scanning is not a substitute for trusting the Dockerfile you are building. The build process itself is also susceptible if the Dockerfile has a vulnerability. Frameworks/strategies such as build-packs or source-to-image (which avoid directly building a Dockerfile) should be considered if you need guardrails around the code you want to build.* 

---

## BuildKit

[BuildKit](https://github.com/moby/buildkit) is composed of the `buildctl` client and the `buildkitd` daemon. For the `buildkit` ClusterBuildStrategy, it runs on a [daemonless](https://github.com/moby/buildkit#daemonless) mode, where both client and ephemeral daemon run in a single container. In addition, it runs without privileges (_[rootless](https://github.com/moby/buildkit/blob/master/docs/rootless.md)_).

### Cache Exporters

By default, the `buildkit` ClusterBuildStrategy will use caching to optimize the build times. When pushing an image to a registry, it will use the `inline` export cache, which pushes the image and cache together. Please refer to [export-cache docs](https://github.com/moby/buildkit#export-cache) for more information.

### Known Limitations

The `buildkit` ClusterBuildStrategy currently locks the following parameters:

- A `Dockerfile` name needs to be `Dockerfile`, this is currently not configurable.
- Exporter caches are enabled by default, this is currently not configurable.
- To allow running rootless, it requires both [AppArmor](https://kubernetes.io/docs/tutorials/clusters/apparmor/) as well as [SecComp](https://kubernetes.io/docs/tutorials/clusters/seccomp/) to be disabled using the `unconfined` profile.

### Usage in Clusters with Pod Security Standards

The BuildKit strategy contains fields with regards to security settings. It therefore depends on the respective cluster setup and administrative configuration. These settings are:

- Defining the `unconfined` profile for both AppArmor and seccomp as required by the underlying `rootlesskit`.
- The `allowPrivilegeEscalation` settings is set to `true` to be able to use binaries that have the `setuid` bit set in order to run with "root" level privileges. In case of BuildKit, this is required by `rootlesskit` in order to set the user namespace mapping file `/proc/<pid>/uid_map`.
- Use of non-root user with UID 1000/GID 1000 as the `runAsUser`.

These settings have no effect in case Pod Security Standards are not used.

_Please note:_ At this point in time, there is no way to run `rootlesskit` to start the BuildKit daemon without the `allowPrivilegeEscalation` flag set to `true`. Clusters with the `Restricted` security standard in place will not be able to use this build strategy.

### Installing BuildKit Strategy

To install the cluster scope strategy, use:

```sh
kubectl apply -f samples/buildstrategy/buildkit/buildstrategy_buildkit_cr.yaml
```

---

## ko

The `ko` ClusterBuilderStrategy is using [ko](https://github.com/google/ko)'s publish command to build an image from a Golang main package.

### Installing ko Strategy

To install the cluster scope strategy, use:

```sh
kubectl apply -f samples/buildstrategy/ko/buildstrategy_ko_cr.yaml
```

### Parameters

The build strategy provides the following parameters that you can set in a Build or BuildRun to control its behavior:

| Parameter | Description | Default |
| -- | -- | -- |
| `go-flags` | Value for the GOFLAGS environment variable. | Empty |
| `go-version` | Version of Go, must match a tag from [the golang image](https://hub.docker.com/_/golang?tab=tags) | `1.16` |
| `ko-version` | Version of ko, must be either `latest` for the newest release, or a [ko release name](https://github.com/google/ko/releases) | `latest` |
| `package-directory` | The directory inside the context directory containing the main package. | `.` |
| `target-platform` | Target platform to be built. For example: `linux/arm64`. Multiple platforms can be provided separated by comma, for example: `linux/arm64,linux/amd64`. The value `all` will build all platforms supported by the base image. The value `current` will build the platform on which the build runs. | `current` |

## Source to Image

This BuildStrategy is composed by [`source-to-image`][s2i] and [`kaniko`][kaniko] in order to generate a `Dockerfile` and prepare the application to be built later on with a builder.

`s2i` requires a specially crafted image, which can be informed as `builderImage` parameter on the `Build` resource.

### Installing Source to Image Strategy

To install the cluster scope strategy use:

```sh
kubectl apply -f samples/buildstrategy/source-to-image/buildstrategy_source-to-image_cr.yaml
```

### Build Steps

1. `s2i` in order to generate a `Dockerfile` and prepare source-code for image build;
2. `kaniko` to create and push the container image to what is defined as `output.image`;

[buildpacks]: https://buildpacks.io/
[cnb]: https://buildpacks.io/docs/concepts/components/builder/
[lifecycle]: https://buildpacks.io/docs/concepts/components/lifecycle/
[hubheroku]: https://hub.docker.com/r/heroku/buildpacks/
[hubcloudfoundry]: https://hub.docker.com/r/cloudfoundry/cnb
[kaniko]: https://github.com/GoogleContainerTools/kaniko
[s2i]: https://github.com/openshift/source-to-image
[buildah]: https://github.com/containers/buildah

## Strategy parameters

Strategy parameters allow users to parameterize their strategy definition, by allowing users to control the _parameters_ values via the `Build` or `BuildRun` resources.

Users defining _parameters_ under their strategies require to understand the following:

- **Definition**: A list of parameters should be defined under `spec.parameters`. Each list item should consist of a _name_, a _description_ and a reasonable _default_ value (_type string_). Note that a default value is not mandatory.
- **Usage**: In order to use a parameter in the strategy steps, users should follow the following syntax: `$(params.your-parameter-name)`
- **Parameterize**: Any `Build` or `BuildRun` referencing your strategy, can set a value for _your-parameter-name_ parameter if needed.

The following is an example of a strategy that defines and uses the `sleep-time` parameter:

```yaml
---
apiVersion: shipwright.io/v1alpha1
kind: BuildStrategy
metadata:
  name: sleepy-strategy
spec:
  parameters:
  - name: sleep-time
    description: "time in seconds for sleeping"
    default: "1"
  buildSteps:
  - name: a-strategy-step
    image: alpine:latest
    command:
    - sleep
    args:
    - $(params.sleep-time)
```

See more information on how to use this parameter in a `Build` or `BuildRun` in the related [docs](./build.md#defining-params).

## System parameters

Contrary to the strategy `spec.parameters`, you can use system parameters and their values defined at runtime when defining the steps of a build strategy to access system information as well as information provided by the user in their Build or BuildRun. The following parameters are available:

| Parameter                      | Description |
| ------------------------------ | ----------- |
| `$(params.shp-source-root)`    | The absolute path to the directory that contains the user's sources. |
| `$(params.shp-source-context)` | The absolute path to the context directory of the user's sources. If the user specified no value for `spec.source.contextDir` in their `Build`, then this value will equal the value for `$(params.shp-source-root)`. Note that this directory is not guaranteed to exist at the time the container for your step is started, you can therefore not use this parameter as a step's working directory. |
| `$(params.shp-output-image)`      | The URL of the image that the user wants to push as specified in the Build's `spec.output.image`, or the override from the BuildRun's `spec.output.image`. |

## System parameters vs Strategy Parameters Comparison

| Parameter Type     | User Configurable | Definition    |
| ------------------ | ------------ | ------------- |
| System Parameter   |    No        |  At run-time, by the `BuildRun` controller.  |
| Strategy Parameter |    Yes       |  At build-time, during the `BuildStrategy` creation. |

## System results

You can optionally store the size and digest of the image your build strategy created to a set of files.

| Result file                        | Description                                     |
| ---------------------------------- | ----------------------------------------------- |
| `$(results.shp-image-digest.path)` | File to store the digest of the image.          |
| `$(results.shp-image-size.path)`   | File to store the compressed size of the image. |

You can look at sample build strategies, such as [Kaniko](../samples/buildstrategy/kaniko/buildstrategy_kaniko_cr.yaml), or [Buildpacks](../samples/buildstrategy/buildpacks-v3/buildstrategy_buildpacks-v3_cr.yaml), to see how they fill some or all of the results files.

This information will be available in the `.status.output` field of the BuildRun.

```yaml
apiVersion: shipwright.io/v1alpha1
kind: BuildRun
# [...]
status:
 # [...]
  output:
    digest: sha256:07626e3c7fdd28d5328a8d6df8d29cd3da760c7f5e2070b534f9b880ed093a53
    size: "1989004"
  # [...]
```

## Steps Resource Definition

All strategies steps can include a definition of resources(_limits and requests_) for CPU, memory and disk. For strategies with more than one step, each step(_container_) could require more resources than others. Strategy admins are free to define the values that they consider the best fit for each step. Also, identical strategies with the same steps that are only different in their name and step resources can be installed on the cluster to allow users to create a build with smaller and larger resource requirements.

### Strategies with different resources

If the strategy admins would require to have multiple flavours of the same strategy, where one strategy has more resources that the other. Then, multiple strategies for the same type should be defined on the cluster. In the following example, we use Kaniko as the type:

```yaml
---
apiVersion: shipwright.io/v1alpha1
kind: ClusterBuildStrategy
metadata:
  name: kaniko-small
spec:
  buildSteps:
    - name: build-and-push
      image: gcr.io/kaniko-project/executor:v1.6.0
      workingDir: $(params.shp-source-root)
      securityContext:
        runAsUser: 0
        capabilities:
          add:
            - CHOWN
            - DAC_OVERRIDE
            - FOWNER
            - SETGID
            - SETUID
            - SETFCAP
            - KILL
      env:
        - name: DOCKER_CONFIG
          value: /tekton/home/.docker
        - name: AWS_ACCESS_KEY_ID
          value: NOT_SET
        - name: AWS_SECRET_KEY
          value: NOT_SET
      command:
        - /kaniko/executor
      args:
        - --skip-tls-verify=true
        - --dockerfile=$(build.dockerfile)
        - --context=$(params.shp-source-context)
        - --destination=$(params.shp-output-image)
        - --snapshotMode=redo
        - --push-retry=3
      resources:
        limits:
          cpu: 250m
          memory: 65Mi
        requests:
          cpu: 250m
          memory: 65Mi
---
apiVersion: shipwright.io/v1alpha1
kind: ClusterBuildStrategy
metadata:
  name: kaniko-medium
spec:
  buildSteps:
    - name: build-and-push
      image: gcr.io/kaniko-project/executor:v1.6.0
      workingDir: $(params.shp-source-root)
      securityContext:
        runAsUser: 0
        capabilities:
          add:
            - CHOWN
            - DAC_OVERRIDE
            - FOWNER
            - SETGID
            - SETUID
            - SETFCAP
            - KILL
      env:
        - name: DOCKER_CONFIG
          value: /tekton/home/.docker
        - name: AWS_ACCESS_KEY_ID
          value: NOT_SET
        - name: AWS_SECRET_KEY
          value: NOT_SET
      command:
        - /kaniko/executor
      args:
        - --skip-tls-verify=true
        - --dockerfile=$(build.dockerfile)
        - --context=$(params.shp-source-context)
        - --destination=$(params.shp-output-image)
        - --snapshotMode=redo
        - --push-retry=3
      resources:
        limits:
          cpu: 500m
          memory: 1Gi
        requests:
          cpu: 500m
          memory: 1Gi
```

The above provides more control and flexibility for the strategy admins. For `end-users`, all they need to do, is to reference the proper strategy. For example:

```yaml
---
apiVersion: shipwright.io/v1alpha1
kind: Build
metadata:
  name: kaniko-medium
spec:
  source:
    url: https://github.com/shipwright-io/sample-go
    contextDir: docker-build
  strategy:
    name: kaniko
    kind: ClusterBuildStrategy
  dockerfile: Dockerfile
```

### How does Tekton Pipelines handle resources

The **Build** controller relies on the Tekton [pipeline controller](https://github.com/tektoncd/pipeline) to schedule the `pods` that execute the above strategy steps. In a nutshell, the **Build** controller creates on run-time a Tekton **TaskRun**, and the **TaskRun** generates a new pod in the particular namespace. In order to build an image, the pod executes all the strategy steps one-by-one.

Tekton manage each step resources **request** in a very particular way, see the [docs](https://github.com/tektoncd/pipeline/blob/main/docs/tasks.md#defining-steps). From this document, it mentions the following:

> The CPU, memory, and ephemeral storage resource requests will be set to zero, or, if specified, the minimums set through LimitRanges in that Namespace, if the container image does not have the largest resource request out of all container images in the Task. This ensures that the Pod that executes the Task only requests enough resources to run a single container image in the Task rather than hoard resources for all container images in the Task at once.

### Examples of Tekton resources management

For a more concrete example, let´s take a look on the following scenarios:

---

**Scenario 1.**  Namespace without `LimitRange`, both steps with the same resource values.

If we will apply the following resources:

- [buildahBuild](../samples/build/build_buildah_cr.yaml)
- [buildahBuildRun](../samples/buildrun/buildrun_buildah_cr.yaml)
- [buildahClusterBuildStrategy](../samples/buildstrategy/buildah/buildstrategy_buildah_cr.yaml)

We will see some differences between the `TaskRun` definition and the `pod` definition.

For the `TaskRun`, as expected we can see the resources on each `step`, as we previously define on our [strategy](../samples/buildstrategy/buildah/buildstrategy_buildah_cr.yaml).

```sh
$ kubectl -n test-build get tr buildah-golang-buildrun-9gmcx-pod-lhzbc -o json | jq '.spec.taskSpec.steps[] | select(.name == "step-buildah-bud" ) | .resources'
{
  "limits": {
    "cpu": "500m",
    "memory": "1Gi"
  },
  "requests": {
    "cpu": "250m",
    "memory": "65Mi"
  }
}

$ kubectl -n test-build get tr buildah-golang-buildrun-9gmcx-pod-lhzbc -o json | jq '.spec.taskSpec.steps[] | select(.name == "step-buildah-push" ) | .resources'
{
  "limits": {
    "cpu": "500m",
    "memory": "1Gi"
  },
  "requests": {
    "cpu": "250m",
    "memory": "65Mi"
  }
}
```

The pod definition is different, while Tekton will only use the **highest** values of one container, and set the rest(lowest) to zero:

```sh
$ kubectl -n test-build get pods buildah-golang-buildrun-9gmcx-pod-lhzbc -o json | jq '.spec.containers[] | select(.name == "step-step-buildah-bud" ) | .resources'
{
  "limits": {
    "cpu": "500m",
    "memory": "1Gi"
  },
  "requests": {
    "cpu": "250m",
    "ephemeral-storage": "0",
    "memory": "65Mi"
  }
}

$ kubectl -n test-build get pods buildah-golang-buildrun-9gmcx-pod-lhzbc -o json | jq '.spec.containers[] | select(.name == "step-step-buildah-push" ) | .resources'
{
  "limits": {
    "cpu": "500m",
    "memory": "1Gi"
  },
  "requests": {
    "cpu": "0",               <------------------- See how the request is set to ZERO.
    "ephemeral-storage": "0", <------------------- See how the request is set to ZERO.
    "memory": "0"             <------------------- See how the request is set to ZERO.
  }
}
```

In this scenario, only one container can have the `spec.resources.requests` definition. Even when both steps have the same values, only one container will get them, the others will be set to zero.

---

**Scenario 2.**  Namespace without `LimitRange`, steps with different resources:

If we will apply the following resources:

- [buildahBuild](../samples/build/buildah/build_buildah_cr.yaml)
- [buildahBuildRun](../samples/buildrun/buildah/buildrun_buildah_cr.yaml)
- We will use a modified buildah strategy, with the following steps resources:

  ```yaml
    - name: buildah-bud
      image: quay.io/containers/buildah:v1.20.1
      workingDir: $(params.shp-source-root)
      securityContext:
        privileged: true
      command:
        - /usr/bin/buildah
      args:
        - bud
        - --tag=$(params.shp-output-image)
        - --file=$(build.dockerfile)
        - $(build.source.contextDir)
      resources:
        limits:
          cpu: 500m
          memory: 1Gi
        requests:
          cpu: 250m
          memory: 65Mi
      volumeMounts:
        - name: buildah-images
          mountPath: /var/lib/containers/storage
    - name: buildah-push
      image: quay.io/containers/buildah:v1.20.1
      securityContext:
        privileged: true
      command:
        - /usr/bin/buildah
      args:
        - push
        - --tls-verify=false
        - docker://$(params.shp-output-image)
      resources:
        limits:
          cpu: 500m
          memory: 1Gi
        requests:
          cpu: 250m
          memory: 100Mi  <------ See how we provide more memory to step-buildah-push, compared to the 65Mi of the other step
  ```

For the `TaskRun`, as expected we can see the resources on each `step`.

```sh
$ kubectl -n test-build get tr buildah-golang-buildrun-skgrp -o json | jq '.spec.taskSpec.steps[] | select(.name == "step-buildah-bud" ) | .resources'
{
  "limits": {
    "cpu": "500m",
    "memory": "1Gi"
  },
  "requests": {
    "cpu": "250m",
    "memory": "65Mi"
  }
}

$ kubectl -n test-build get tr buildah-golang-buildrun-skgrp -o json | jq '.spec.taskSpec.steps[] | select(.name == "step-buildah-push" ) | .resources'
{
  "limits": {
    "cpu": "500m",
    "memory": "1Gi"
  },
  "requests": {
    "cpu": "250m",
    "memory": "100Mi"
  }
}
```

The pod definition is different, while Tekton will only use the **highest** values of one container, and set the rest(lowest) to zero:

```sh
$ kubectl -n test-build get pods buildah-golang-buildrun-95xq8-pod-mww8d -o json | jq '.spec.containers[] | select(.name == "step-step-buildah-bud" ) | .resources'
{
  "limits": {
    "cpu": "500m",
    "memory": "1Gi"
  },
  "requests": {
    "cpu": "250m",                <------------------- See how the CPU is preserved
    "ephemeral-storage": "0",
    "memory": "0"                 <------------------- See how the memory is set to ZERO
  }
}
$ kubectl -n test-build get pods buildah-golang-buildrun-95xq8-pod-mww8d -o json | jq '.spec.containers[] | select(.name == "step-step-buildah-push" ) | .resources'
{
  "limits": {
    "cpu": "500m",
    "memory": "1Gi"
  },
  "requests": {
    "cpu": "0",                     <------------------- See how the CPU is set to zero.
    "ephemeral-storage": "0",
    "memory": "100Mi"               <------------------- See how the memory is preserved on this container
  }
}
```

In the above scenario, we can see how the maximum numbers for resource requests are distributed between containers. The container `step-buildah-push` gets the `100mi` for the memory requests, while it was the one defining the highest number. At the same time, the container `step-buildah-bud` is assigned a `0` for its memory request.

---

**Scenario 3.**  Namespace **with** a `LimitRange`.

When a `LimitRange` exists on the namespace, `Tekton Pipeline` controller will do the same approach as stated in the above two scenarios. The difference is that for the containers that have lower values, instead of zero, they will get the `minimum values of the LimitRange`.

## Annotations

Annotations can be defined for a BuildStrategy/ClusterBuildStrategy as for any other Kubernetes object. Annotations are propagated to the TaskRun and from there, Tekton propagates them to the Pod. Use cases for this are for example:

- The Kubernetes [Network Traffic Shaping](https://kubernetes.io/docs/concepts/extend-kubernetes/compute-storage-net/network-plugins/#support-traffic-shaping) feature looks for the `kubernetes.io/ingress-bandwidth` and `kubernetes.io/egress-bandwidth` annotations to limit the network bandwidth the `Pod` is allowed to use.
- The [AppArmor profile of a container](https://kubernetes.io/docs/tutorials/clusters/apparmor/) is defined using the `container.apparmor.security.beta.kubernetes.io/<container_name>` annotation.

The following annotations are not propagated:

- `kubectl.kubernetes.io/last-applied-configuration`
- `clusterbuildstrategy.shipwright.io/*`
- `buildstrategy.shipwright.io/*`
- `build.shipwright.io/*`
- `buildrun.shipwright.io/*`

A Kubernetes administrator can further restrict the usage of annotations by using policy engines like [Open Policy Agent](https://www.openpolicyagent.org/).
