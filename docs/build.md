<!--
Copyright The Shipwright Contributors

SPDX-License-Identifier: Apache-2.0
-->

# Build

- [Build](#build)
  - [Overview](#overview)
  - [Build Controller](#build-controller)
  - [Build Validations](#build-validations)
  - [Configuring a Build](#configuring-a-build)
    - [Defining the Source](#defining-the-source)
    - [Defining the Strategy](#defining-the-strategy)
    - [Defining ParamValues](#defining-paramvalues)
      - [Example](#example)
    - [Defining the Builder or Dockerfile](#defining-the-builder-or-dockerfile)
    - [Defining the Output](#defining-the-output)
    - [Defining the vulnerabilityScan](#defining-the-vulnerabilityscan)
    - [Defining Retention Parameters](#defining-retention-parameters)
    - [Defining Volumes](#defining-volumes)
    - [Defining Triggers](#defining-triggers)
      - [GitHub](#github)
      - [Image](#image)
      - [Tekton Pipeline](#tekton-pipeline)
  - [BuildRun Deletion](#buildrun-deletion)

## Overview

A `Build` resource allows the user to define:

- source
- trigger
- strategy
- paramValues
- output
- timeout
- env
- retention
- volumes
- nodeSelector
- tolerations

A `Build` is available within a namespace.

## Build Controller

The controller watches for:

- Updates on the `Build` resource (_CRD instance_)

When the controller reconciles it:

- Validates if the referenced `Strategy` exists.
- Validates if the specified `paramValues` exist on the referenced strategy parameters. It also validates if the `paramValues` names collide with the Shipwright reserved names.
- Validates if the container `registry` output secret exists.
- Validates if the referenced `spec.source.git.url` endpoint exists.

## Build Validations

**Note**: reported validations in build status are deprecated, and will be removed in a future release.

To prevent users from triggering `BuildRun`s (_execution of a Build_) that will eventually fail because of wrong or missing dependencies or configuration settings, the Build controller will validate them in advance. If all validations are successful, users can expect a `Succeeded` `status.reason`. However, if any validations fail, users can rely on the `status.reason` and `status.message` fields to understand the root cause.

| Status.Reason                                   | Description                                                                                                                                                                                                  |
|-------------------------------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| BuildStrategyNotFound                           | The referenced namespace-scope strategy doesn't exist.                                                                                                                                                       |
| ClusterBuildStrategyNotFound                    | The referenced cluster-scope strategy doesn't exist.                                                                                                                                                         |
| SetOwnerReferenceFailed                         | Setting ownerreferences between a Build and a BuildRun failed. This status is triggered when you set the `spec.retention.atBuildDeletion` to true in a Build.                                                |
| SpecSourceSecretRefNotFound                     | The secret used to authenticate to git doesn't exist.                                                                                                                                                        |
| SpecOutputSecretRefNotFound                     | The secret used to authenticate to the container registry doesn't exist.                                                                                                                                     |
| SpecBuilderSecretRefNotFound                    | The secret used to authenticate the container registry doesn't exist.                                                                                                                                        |
| MultipleSecretRefNotFound                       | More than one secret is missing. At the moment, only three paths on a Build can specify a secret.                                                                                                            |
| RestrictedParametersInUse                       | One or many defined `paramValues` are colliding with Shipwright reserved parameters. See [Defining Params](#defining-paramvalues) for more information.                                                      |
| UndefinedParameter                              | One or many defined `paramValues` are not defined in the referenced strategy. Please ensure that the strategy defines them under its `spec.parameters` list.                                                 |
| RemoteRepositoryUnreachable                     | The defined `spec.source.git.url` was not found. This validation only takes place for HTTP/HTTPS protocols.                                                                                                  |
| BuildNameInvalid                                | The defined `Build` name (`metadata.name`) is invalid. The `Build` name should be a [valid label value](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#syntax-and-character-set). |
| SpecEnvNameCanNotBeBlank                        | The name for a user-provided environment variable is blank.                                                                                                                                                  |
| SpecEnvValueCanNotBeBlank                       | The value for a user-provided environment variable is blank.                                                                                                                                                 |
| SpecEnvOnlyOneOfValueOrValueFromMustBeSpecified | Both value and valueFrom were specified, which are mutually exclusive.                                                                                                                                       |
| RuntimePathsCanNotBeEmpty                       | The `spec.runtime` feature is used but the paths were not specified.                                                                                                                                         |
| WrongParameterValueType                         | A single value was provided for an array parameter, or vice-versa.                                                                                                                                           |
| InconsistentParameterValues                     | Parameter values have more than one of _configMapValue_, _secretValue_, or _value_ set.                                                                                                                      |
| EmptyArrayItemParameterValues                   | Array parameters contain an item where none of _configMapValue_, _secretValue_, or _value_ is set.                                                                                                           |
| IncompleteConfigMapValueParameterValues         | A _configMapValue_ is specified where the name or the key is empty.                                                                                                                                          |
| IncompleteSecretValueParameterValues            | A _secretValue_ is specified where the name or the key is empty.                                                                                                                                             |
| VolumeDoesNotExist                              | Volume referenced by the Build does not exist, therefore Build cannot be run.                                                                                                                                |
| VolumeNotOverridable                            | Volume defined by build is not set as overridable in the strategy.                                                                                                                                           |
| UndefinedVolume                                 | Volume defined by build is not found in the strategy.                                                                                                                                                        |
| TriggerNameCanNotBeBlank                        | Trigger condition does not have a name.                                                                                                                                                                      |
| TriggerInvalidType                              | Trigger type is invalid.                                                                                                                                                                                     |
| TriggerInvalidGitHubWebHook                     | Trigger type GitHub is invalid.                                                                                                                                                                              |
| TriggerInvalidImage                             | Trigger type Image is invalid.                                                                                                                                                                               |
| TriggerInvalidPipeline                          | Trigger type Pipeline is invalid.                                                                                                                                                                            |
| OutputTimestampNotSupported                     | An unsupported output timestamp setting was used.                                                                                                                                                            |
| OutputTimestampNotValid                         | The output timestamp value is not valid.                                                                                                                                                                     |
| NodeSelectorNotValid                            | The specified nodeSelector is not valid. |
| TolerationNotValid                              | The specified tolerations are not valid. |

## Configuring a Build

The `Build` definition supports the following fields:

- Required:
  - [`apiVersion`](https://kubernetes.io/docs/concepts/overview/working-with-objects/kubernetes-objects/#required-fields) - Specifies the API version, for example `shipwright.io/v1beta1`.
  - [`kind`](https://kubernetes.io/docs/concepts/overview/working-with-objects/kubernetes-objects/#required-fields) - Specifies the Kind type, for example `Build`.
  - [`metadata`](https://kubernetes.io/docs/concepts/overview/working-with-objects/kubernetes-objects/#required-fields) - Metadata that identify the custom resource instance, especially the name of the `Build`, and in which [namespace](https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/) you place it. **Note**: You should use your own namespace, and not put your builds into the shipwright-build namespace where Shipwright's system components run.
  - `spec.source` - Refers to the location of the source code, for example a Git repository or OCI artifact image.
  - `spec.strategy` - Refers to the `BuildStrategy` to be used, see the [examples](../samples/v1beta1/buildstrategy)
  - `spec.output`- Refers to the location where the generated image would be pushed.
  - `spec.output.pushSecret`- Reference an existing secret to get access to the container registry.

- Optional:
  - `spec.paramValues` - Refers to a name-value(s) list to specify values for `parameters` defined in the `BuildStrategy`.
  - `spec.timeout` - Defines a custom timeout. The value needs to be parsable by [ParseDuration](https://golang.org/pkg/time/#ParseDuration), for example, `5m`. The default is ten minutes. You can overwrite the value in the `BuildRun`.
  - `spec.output.annotations` - Refers to a list of `key/value` that could be used to [annotate](https://github.com/opencontainers/image-spec/blob/main/annotations.md) the output image.
  - `spec.output.labels` - Refers to a list of `key/value` that could be used to label the output image.
  - `spec.output.timestamp` - Instruct the build to change the output image creation timestamp to the specified value. When omitted, the respective build strategy tool defines the output image timestamp.
    - Use string `Zero` to set the image timestamp to UNIX epoch timestamp zero.
    - Use string `SourceTimestamp` to set the image timestamp to the source timestamp, i.e. the timestamp of the Git commit that was used.
    - Use string `BuildTimestamp` to set the image timestamp to the timestamp of the build run.
    - Use any valid UNIX epoch seconds number as a string to set this as the image timestamp.
  - `spec.output.vulnerabilityScan` to enable a security vulnerability scan for your generated image. Further options in vulnerability scanning are defined [here](#defining-the-vulnerabilityscan)
  - `spec.env` - Specifies additional environment variables that should be passed to the build container. The available variables depend on the tool that is being used by the chosen build strategy.
  - `spec.retention.atBuildDeletion` - Defines if all related BuildRuns needs to be deleted when deleting the Build. The default is false.
  - `spec.retention.ttlAfterFailed` - Specifies the duration for which a failed buildrun can exist.
  - `spec.retention.ttlAfterSucceeded` - Specifies the duration for which a successful buildrun can exist.
  - `spec.retention.failedLimit` - Specifies the number of failed buildrun that can exist.
  - `spec.retention.succeededLimit` - Specifies the number of successful buildrun can exist.
  - `spec.nodeSelector` - Specifies a selector which must match a node's labels for the build pod to be scheduled on that node. If nodeSelectors are specified in both a `Build` and `BuildRun`, `BuildRun` values take precedence.
  - `spec.tolerations` - Specifies the tolerations for the build pod. Only `key`, `value`, and `operator` are supported. Only `NoSchedule` taint `effect` is supported. If tolerations are specified in both a `Build` and `BuildRun`, `BuildRun` values take precedence.

### Defining the Source

A `Build` resource can specify a source type, such as a Git repository or an OCI artifact, together with other parameters like:

- `source.type` - Specify the type of the data-source. Currently, the supported types are "Git", "OCIArtifact", and "Local".
- `source.git.url` - Specify the source location using a Git repository.
- `source.git.cloneSecret` - For private repositories or registries, the name references a secret in the namespace that contains the SSH private key or Docker access credentials, respectively.
- `source.git.revision` - A specific revision to select from the source repository, this can be a commit, tag or branch name. If not defined, it will fall back to the Git repository default branch.
- `source.contextDir` - For repositories where the source code is not located at the root folder, you can specify this path here.

By default, the Build controller does not validate that the Git repository exists. If the validation is desired, users can explicitly define the `build.shipwright.io/verify.repository` annotation with `true`. For example:

Example of a `Build` with the **build.shipwright.io/verify.repository** annotation to enable the `spec.source.git.url` validation.

```yaml
apiVersion: shipwright.io/v1beta1
kind: Build
metadata:
  name: buildah-golang-build
  annotations:
    build.shipwright.io/verify.repository: "true"
spec:
  source:
    type: Git
    git:
      url: https://github.com/shipwright-io/sample-go
    contextDir: docker-build
```

**Note**: The Build controller only validates two scenarios. The first one is when the endpoint uses an `http/https` protocol. The second one is when an `ssh` protocol such as `git@` has been defined but a referenced secret, such as `source.git.cloneSecret`, has not been provided.

Example of a `Build` with a source with **credentials** defined by the user.

```yaml
apiVersion: shipwright.io/v1beta1
kind: Build
metadata:
  name: buildpack-nodejs-build
spec:
  source:
    type: Git
    git:
      url: https://github.com/sclorg/nodejs-ex
      cloneSecret: source-repository-credentials
```

Example of a `Build` with a source that specifies a specific subfolder on the repository.

```yaml
apiVersion: shipwright.io/v1beta1
kind: Build
metadata:
  name: buildah-custom-context-dockerfile
spec:
  source:
    type: Git
    git:
      url: https://github.com/SaschaSchwarze0/npm-simple
    contextDir: renamed
```

Example of a `Build` that specifies the tag `v0.1.0` for the git repository:

```yaml
apiVersion: shipwright.io/v1beta1
kind: Build
metadata:
  name: buildah-golang-build
spec:
  source:
    type: Git
    git:
      url: https://github.com/shipwright-io/sample-go
      revision: v0.1.0
    contextDir: docker-build
```

Example of a `Build` that specifies environment variables:

```yaml
apiVersion: shipwright.io/v1beta1
kind: Build
metadata:
  name: buildah-golang-build
spec:
  source:
    type: Git
    git:
      url: https://github.com/shipwright-io/sample-go
    contextDir: docker-build
  env:
    - name: EXAMPLE_VAR_1
      value: "example-value-1"
    - name: EXAMPLE_VAR_2
      value: "example-value-2"
```

Example of a `Build` that uses the Kubernetes Downward API to expose a `Pod` field as an environment variable:

```yaml
apiVersion: shipwright.io/v1beta1
kind: Build
metadata:
  name: buildah-golang-build
spec:
  source:
    type: Git
    git:
      url: https://github.com/shipwright-io/sample-go
    contextDir: docker-build
  env:
    - name: POD_NAME
      valueFrom:
        fieldRef:
          fieldPath: metadata.name
```

Example of a `Build` that uses the Kubernetes Downward API to expose a `Container` field as an environment variable:

```yaml
apiVersion: shipwright.io/v1beta1
kind: Build
metadata:
  name: buildah-golang-build
spec:
  source:
    type: Git
    git:
      url: https://github.com/shipwright-io/sample-go
    contextDir: docker-build
  env:
    - name: MEMORY_LIMIT
      valueFrom:
        resourceFieldRef:
          containerName: my-container
          resource: limits.memory
```

### Defining the Strategy

A `Build` resource can specify the `BuildStrategy` to use, these are:

- [Buildah](buildstrategies.md#buildah)
- [Buildpacks-v3](buildstrategies.md#buildpacks-v3)
- [BuildKit](buildstrategies.md#buildkit)
- [Kaniko](buildstrategies.md#kaniko)
- [ko](buildstrategies.md#ko)
- [Source-to-Image](buildstrategies.md#source-to-image)

Defining the strategy is straightforward. You define the `name` and the `kind`. For example:

```yaml
apiVersion: shipwright.io/v1beta1
kind: Build
metadata:
  name: buildpack-nodejs-build
spec:
  strategy:
    name: buildpacks-v3
    kind: ClusterBuildStrategy
```

### Defining ParamValues

A `Build` resource can specify _paramValues_ for parameters that are defined in the referenced `BuildStrategy`. You specify these parameter values to control how the steps of the build strategy behave. You can overwrite values in the `BuildRun` resource. See the related [documentation](buildrun.md#defining-paramvalues) for more information.

The build strategy author can define a parameter as either a simple string or an array. Depending on that, you must specify the value accordingly. The build strategy parameter can be specified with a default value. You must specify a value in the `Build` or `BuildRun` for parameters without a default.

You can either specify values directly or reference keys from [ConfigMaps](https://kubernetes.io/docs/concepts/configuration/configmap/) and [Secrets](https://kubernetes.io/docs/concepts/configuration/secret/). **Note**: the usage of ConfigMaps and Secrets is limited by the usage of the parameter in the build strategy steps. You can only use them if the parameter is used in the command, arguments, or environment variable values.

When using _paramValues_, users should avoid:

- Defining a `spec.paramValues` name that doesn't match one of the `spec.parameters` defined in the `BuildStrategy`.
- Defining a `spec.paramValues` name that collides with the Shipwright reserved parameters. These are _BUILDER\_IMAGE_, _DOCKERFILE_, _CONTEXT\_DIR_, and any name starting with _shp-_.

In general, _paramValues_ are tightly bound to Strategy _parameters_. Please make sure you understand the contents of your strategy of choice before defining _paramValues_ in the _Build_.

#### Example

The [BuildKit sample `BuildStrategy`](../samples/v1beta1/buildstrategy/buildkit/buildstrategy_buildkit_cr.yaml) contains various parameters. Two of them are outlined here:

```yaml
apiVersion: shipwright.io/v1beta1
kind: ClusterBuildStrategy
metadata:
  name: buildkit
  ...
spec:
  parameters:
  - name: build-args
    description: "The ARG values in the Dockerfile. Values must be in the format KEY=VALUE."
    type: array
    defaults: []
  - name: cache
    description: "Configure BuildKit's cache usage. Allowed values are 'disabled' and 'registry'. The default is 'registry'."
    type: string
    default: registry
  ...
  steps:
  ...
```

The `cache` parameter is a simple string. You can provide it like this in your Build:

```yaml
apiVersion: shipwright.io/v1beta1
kind: Build
metadata:
  name: a-build
  namespace: a-namespace
spec:
  paramValues:
  - name: cache
    value: disabled
  strategy:
    name: buildkit
    kind: ClusterBuildStrategy
  source:
  ...
  output:
  ...
```

If you have multiple Builds and want to control this parameter centrally, then you can create a ConfigMap:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: buildkit-configuration
  namespace: a-namespace
data:
  cache: disabled
```

You reference the ConfigMap as a parameter value like this:

```yaml
apiVersion: shipwright.io/v1beta1
kind: Build
metadata:
  name: a-build
  namespace: a-namespace
spec:
  paramValues:
  - name: cache
    configMapValue:
      name: buildkit-configuration
      key: cache
  strategy:
    name: buildkit
    kind: ClusterBuildStrategy
  source:
  ...
  output:
  ...
```

The `build-args` parameter is defined as an array. In the BuildKit strategy, you use `build-args` to set the [`ARG` values in the Dockerfile](https://docs.docker.com/engine/reference/builder/#arg), specified as key-value pairs separated by an equals sign, for example, `NODE_VERSION=16`. Your Build then looks like this (the value for `cache` is retained to outline how multiple _paramValue_ can be set):

```yaml
apiVersion: shipwright.io/v1beta1
kind: Build
metadata:
  name: a-build
  namespace: a-namespace
spec:
  paramValues:
  - name: cache
    configMapValue:
      name: buildkit-configuration
      key: cache
  - name: build-args
    values:
    - value: NODE_VERSION=16
  strategy:
    name: buildkit
    kind: ClusterBuildStrategy
  source:
  ...
  output:
  ...
```

Like simple values, you can also reference ConfigMaps and Secrets for every item in the array. Example:

```yaml
apiVersion: shipwright.io/v1beta1
kind: Build
metadata:
  name: a-build
  namespace: a-namespace
spec:
  paramValues:
  - name: cache
    configMapValue:
      name: buildkit-configuration
      key: cache
  - name: build-args
    values:
    - configMapValue:
        name: project-configuration
        key: node-version
        format: NODE_VERSION=${CONFIGMAP_VALUE}
    - value: DEBUG_MODE=true
    - secretValue:
        name: npm-registry-access
        key: npm-auth-token
        format: NPM_AUTH_TOKEN=${SECRET_VALUE}
  strategy:
    name: buildkit
    kind: ClusterBuildStrategy
  source:
  ...
  output:
  ...
```

Here, we pass three items in the `build-args` array:

1. The first item references a ConfigMap. Because the ConfigMap just contains the value (for example `"16"`) as the data of the `node-version` key, the `format` setting is used to prepend `NODE_VERSION=` to make it a complete key-value pair.
2. The second item is just a hard-coded value.
3. The third item references a Secret, the same as with ConfigMaps.

**Note**: The logging output of BuildKit contains expanded `ARG`s in `RUN` commands. Also, such information ends up in the final container image if you use such args in the [final stage of your Dockerfile](https://docs.docker.com/develop/develop-images/multistage-build/). An alternative approach to pass secrets is using [secret mounts](https://docs.docker.com/develop/develop-images/build_enhancements/#new-docker-build-secret-information). The BuildKit sample strategy supports them using the `secrets` parameter.

### Defining the Builder or Dockerfile

In the `Build` resource, you use the parameters (`spec.paramValues`) to specify the image that contains the tools to build the final image. For example, the following Build definition specifies a `Dockerfile` image.

```yaml
apiVersion: shipwright.io/v1beta1
kind: Build
metadata:
  name: buildah-golang-build
spec:
  source:
    type: Git
    git:
      url: https://github.com/shipwright-io/sample-go
    contextDir: docker-build
  strategy:
    name: buildah
    kind: ClusterBuildStrategy
  paramValues:
  - name: dockerfile
    value: Dockerfile
```

Another example is when the user chooses the `builder` image for a specific language as part of the `source-to-image` buildStrategy:

```yaml
apiVersion: shipwright.io/v1beta1
kind: Build
metadata:
  name: s2i-nodejs-build
spec:
  source:
    type: Git
    git:
      url: https://github.com/shipwright-io/sample-nodejs
    contextDir: source-build/
  strategy:
    name: source-to-image
    kind: ClusterBuildStrategy
  paramValues:
  - name: builder-image
    value: "docker.io/centos/nodejs-10-centos7"
```

### Defining the Output

A `Build` resource can specify the output where it should push the image. For external private registries, it is recommended to specify a secret with the related data to access it. An option is available to specify the annotation and labels for the output image. The annotations and labels mentioned here are specific to the container image and do not relate to the `Build` annotations. Analogous, the timestamp refers to the timestamp of the output image.

**Note**: When you specify annotations, labels, or timestamp, the output image **may** get pushed twice, depending on the respective strategy. For example, strategies that push the image to the registry as part of their build step will lead to an additional push of the image in case image processing like labels is configured. If you have automation based on push events in your container registry, be aware of this behavior.

For example, the user specifies a public registry:

```yaml
apiVersion: shipwright.io/v1beta1
kind: Build
metadata:
  name: s2i-nodejs-build
spec:
  source:
    type: Git
    git:
      url: https://github.com/shipwright-io/sample-nodejs
    contextDir: source-build/
  strategy:
    name: source-to-image
    kind: ClusterBuildStrategy
  paramValues:
  - name: builder-image
    value: "docker.io/centos/nodejs-10-centos7"
  output:
    image: image-registry.openshift-image-registry.svc:5000/build-examples/nodejs-ex
```

Another example is when the user specifies a private registry:

```yaml
apiVersion: shipwright.io/v1beta1
kind: Build
metadata:
  name: s2i-nodejs-build
spec:
  source:
    git:
      url: https://github.com/shipwright-io/sample-nodejs
    contextDir: source-build/
  strategy:
    name: source-to-image
    kind: ClusterBuildStrategy
  paramValues:
  - name: builder-image
    value: "docker.io/centos/nodejs-10-centos7"
  output:
    image: us.icr.io/source-to-image-build/nodejs-ex
    pushSecret: icr-knbuild
```

Example of user specifies image annotations and labels:

```yaml
apiVersion: shipwright.io/v1beta1
kind: Build
metadata:
  name: s2i-nodejs-build
spec:
  source:
    type: Git
    git:
      url: https://github.com/shipwright-io/sample-nodejs
    contextDir: source-build/
  strategy:
    name: source-to-image
    kind: ClusterBuildStrategy
  paramValues:
  - name: builder-image
    value: "docker.io/centos/nodejs-10-centos7"
  output:
    image: us.icr.io/source-to-image-build/nodejs-ex
    pushSecret: icr-knbuild
    annotations:
      "org.opencontainers.image.source": "https://github.com/org/repo"
      "org.opencontainers.image.url": "https://my-company.com/images"
    labels:
      "maintainer": "team@my-company.com"
      "description": "This is my cool image"
```

Example of user specified image timestamp set to `SourceTimestamp` to set the output timestamp to match the timestamp of the Git commit used for the build:

```yaml
apiVersion: shipwright.io/v1beta1
kind: Build
metadata:
  name: sample-go-build
spec:
  source:
    type: Git
    git:
      url: https://github.com/shipwright-io/sample-go
    contextDir: source-build
  strategy:
    name: buildkit
    kind: ClusterBuildStrategy
  output:
    image: some.registry.com/namespace/image:tag
    pushSecret: credentials
    timestamp: SourceTimestamp
```

### Defining the vulnerabilityScan

`vulnerabilityScan` provides configurations to run a scan for your generated image.

- `vulnerabilityScan.enabled` - Specify whether to run vulnerability scan for image. The supported values are true and false.
- `vulnerabilityScan.failOnFinding` - indicates whether to fail the build run if the vulnerability scan results in vulnerabilities. The supported values are true and false. This field is optional and false by default.
- `vulnerabilityScan.ignore.issues` - references the security issues to be ignored in vulnerability scan
- `vulnerabilityScan.ignore.severity` - denotes the severity levels of security issues to be ignored, valid values are:
  - `low`: it will exclude low severity vulnerabilities, displaying only medium, high and critical vulnerabilities
  - `medium`: it will exclude low and medium severity vulnerabilities, displaying only high and critical vulnerabilities
  - `high`: it will exclude low, medium and high severity vulnerabilities, displaying only the critical vulnerabilities
- `vulnerabilityScan.ignore.unfixed` - indicates to ignore vulnerabilities for which no fix exists. The supported types are true and false.

Example of user specified image vulnerability scanning options:

```yaml
apiVersion: shipwright.io/v1beta1
kind: Build
metadata:
  name: sample-go-build
spec:
  source:
    type: Git
    git:
      url: https://github.com/shipwright-io/sample-go
    contextDir: source-build
  strategy:
    name: buildkit
    kind: ClusterBuildStrategy
  output:
    image: some.registry.com/namespace/image:tag
    pushSecret: credentials
    vulnerabilityScan:
      enabled: true
      failOnFinding: true
      ignore:
        issues:
          - CVE-2022-12345
        severity: Low
        unfixed: true
```

Annotations added to the output image can be verified by running the command:

```sh
docker manifest inspect us.icr.io/source-to-image-build/nodejs-ex | jq ".annotations"
```

You can verify which labels were added to the output image that is available on the host machine by running the command:

```sh
docker inspect us.icr.io/source-to-image-build/nodejs-ex | jq ".[].Config.Labels"
```

### Defining Retention Parameters

A `Build` resource can specify how long a completed BuildRun can exist and the number of buildruns that have failed or succeeded that should exist. Instead of manually cleaning up old BuildRuns, retention parameters provide an alternate method for cleaning up BuildRuns automatically.

As part of the retention parameters, we have the following fields:

- `retention.atBuildDeletion` - Defines if all related BuildRuns needs to be deleted when deleting the Build. The default is false.
- `retention.succeededLimit` - Defines number of succeeded BuildRuns for a Build that can exist.
- `retention.failedLimit` - Defines number of failed BuildRuns for a Build that can exist.
- `retention.ttlAfterFailed` - Specifies the duration for which a failed buildrun can exist.
- `retention.ttlAfterSucceeded` - Specifies the duration for which a successful buildrun can exist.

An example of a user using both TTL and Limit retention fields. In case of such a configuration, BuildRun will get deleted once the first criteria is met.

```yaml
  apiVersion: shipwright.io/v1beta1
  kind: Build
  metadata:
    name: build-retention-ttl
  spec:
    source:
      type: Git
      git:
        url: "https://github.com/shipwright-io/sample-go"
      contextDir: docker-build
    strategy:
      kind: ClusterBuildStrategy
    output:
    ...
    retention:
      ttlAfterFailed: 30m
      ttlAfterSucceeded: 1h
      failedLimit: 10
      succeededLimit: 20
```

**Note**: When changes are made to `retention.failedLimit` and `retention.succeededLimit` values, they come into effect as soon as the build is applied, thereby enforcing the new limits. On the other hand, changing the `retention.ttlAfterFailed` and `retention.ttlAfterSucceeded` values will only affect new buildruns. Old buildruns will adhere to the old TTL retention values. In case TTL values are defined in buildrun specifications as well as build specifications, priority will be given to the values defined in the buildrun specifications.

### Defining Volumes

`Builds` can declare `volumes`. They must override `volumes` defined by the according `BuildStrategy`. If a `volume`
is not `overridable` then the `BuildRun` will eventually fail.

`Volumes` follow the declaration of [Pod Volumes](https://kubernetes.io/docs/concepts/storage/volumes/), so
all the usual `volumeSource` types are supported.

Here is an example of `Build` object that overrides `volumes`:

```yaml
apiVersion: shipwright.io/v1beta1
kind: Build
metadata:
  name: build-name
spec:
  source:
    type: Git
    git:
      url: https://github.com/example/url
  strategy:
    name: buildah
    kind: ClusterBuildStrategy
  paramValues:
  - name: dockerfile
    value: Dockerfile
  output:
    image: registry/namespace/image:latest
  volumes:
    - name: volume-name
      configMap:
        name: test-config
```

### Defining Triggers

Using the triggers, you can submit `BuildRun` instances when certain events happen. The idea is to be able to trigger Shipwright builds in an event driven fashion, for that purpose you can watch certain types of events.

**Note**: triggers rely on the [Shipwright Triggers](https://github.com/shipwright-io/triggers) project to be deployed and configured in the same Kubernetes cluster where you run Shipwright Build. If it is not set up, the triggers defined in a Build are ignored.

The types of events under watch are defined on the `.spec.trigger` attribute, please consider the following example:

```yaml
apiVersion: shipwright.io/v1beta1
kind: Build
spec:
  source:
    type: Git
    git:
      url: https://github.com/shipwright-io/sample-go
      cloneSecret: webhook-secret
    contextDir: docker-build
  trigger:
    when: []
```

Certain types of events will use attributes defined on `.spec.source` to complete the information needed in order to dispatch events.

#### GitHub

The GitHub type is meant to react upon events coming from GitHub WebHook interface, the events are compared against the existing `Build` resources, and therefore it can identify the `Build` objects based on `.spec.source.git.url` combined with the attributes on `.spec.trigger.when[].github`.

To identify a given `Build` object, the first criteria is the repository URL, and then the branch name listed on the GitHub event payload must also match. Following the criteria:

- First, the branch name is checked against the `.spec.trigger.when[].github.branches` entries
- If the `.spec.trigger.when[].github.branches` is empty, the branch name is compared against `.spec.source.git.revision`
- If `spec.source.git.revision` is empty, the default revision name is used ("main")

The following snippet shows a configuration matching `Push` and `PullRequest` events on the `main` branch, for example:

```yaml
# [...]
spec:
  source:
    git:
      url: https://github.com/shipwright-io/sample-go
  trigger:
    when:
      - name: push and pull-request on the main branch
        type: GitHub
        github:
          events:
            - Push
            - PullRequest
          branches:
            - main
```

#### Image

In order to watch over images, in combination with the [Image](https://github.com/shipwright-io/image) controller, you can trigger new builds when those container image names change.

For instance, lets imagine the image named `ghcr.io/some/base-image` is used as input for the Build process and every time it changes we would like to trigger a new build. Please consider the following snippet:

```yaml
# [...]
spec:
  trigger:
    when:
      - name: watching for the base-image changes
        type: Image
        image:
          names:
            - ghcr.io/some/base-image:latest
```

#### Tekton Pipeline

Shipwright can also be used in combination with [Tekton Pipeline](https://github.com/tektoncd/pipeline), you can configure the Build to watch for `Pipeline` resources in Kubernetes reacting when the object reaches the desired status (`.objectRef.status`), and is identified either by its name (`.objectRef.name`) or a label selector (`.objectRef.selector`). The example below uses the label selector approach:

```yaml
# [...]
spec:
  trigger:
    when:
      - name: watching over for the Tekton Pipeline
        type: Pipeline
        objectRef:
          status:
            - Succeeded
          selector:
            label: value
```

While the next snippet uses the object name for identification:

```yaml
# [...]
spec:
  trigger:
    when:
      - name: watching over for the Tekton Pipeline
        type: Pipeline
        objectRef:
          status:
            - Succeeded
          name: tekton-pipeline-name
```

## BuildRun Deletion

A `Build` can automatically delete a related `BuildRun`. To enable this feature set the `spec.retention.atBuildDeletion` to `true` in the `Build` instance. The default value is set to `false`. See an example of how to define this field:

```yaml
apiVersion: shipwright.io/v1beta1
kind: Build
metadata:
  name: kaniko-golang-build
spec:
  retention:
    atBuildDeletion: true
  # [...]
```
