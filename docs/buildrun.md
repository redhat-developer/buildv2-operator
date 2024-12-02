<!--
Copyright The Shipwright Contributors

SPDX-License-Identifier: Apache-2.0
-->

# BuildRun

- [BuildRun](#buildrun)
  - [Overview](#overview)
  - [BuildRun Controller](#buildrun-controller)
  - [Configuring a BuildRun](#configuring-a-buildrun)
    - [Defining the Build Reference](#defining-the-build-reference)
    - [Defining the Build Specification](#defining-the-build-specification)
    - [Defining the Build Source](#defining-the-build-source)
    - [Defining ParamValues](#defining-paramvalues)
    - [Defining the ServiceAccount](#defining-the-serviceaccount)
    - [Defining Retention Parameters](#defining-retention-parameters)
    - [Defining Volumes](#defining-volumes)
  - [Canceling a `BuildRun`](#canceling-a-buildrun)
  - [Automatic `BuildRun` deletion](#automatic-buildrun-deletion)
  - [Specifying Environment Variables](#specifying-environment-variables)
  - [BuildRun Status](#buildrun-status)
    - [Understanding the state of a BuildRun](#understanding-the-state-of-a-buildrun)
    - [Understanding failed BuildRuns](#understanding-failed-buildruns)
    - [Understanding failed BuildRuns due to VulnerabilitiesFound](#understanding-failed-buildruns-due-to-vulnerabilitiesfound)
      - [Understanding failed git-source step](#understanding-failed-git-source-step)
    - [Step Results in BuildRun Status](#step-results-in-buildrun-status)
    - [Build Snapshot](#build-snapshot)
  - [Relationship with Tekton Tasks](#relationship-with-tekton-tasks)

## Overview

The resource `BuildRun` (`buildruns.shipwright.io/v1beta1`) is the build process of a `Build` resource definition executed in Kubernetes.

A `BuildRun` resource allows the user to define:

- The `BuildRun` name, through which the user can monitor the status of the image construction.
- A referenced `Build` instance to use during the build construction.
- A service account for hosting all related secrets to build the image.

A `BuildRun` is available within a namespace.

## BuildRun Controller

The controller watches for:

- Updates on a `Build` resource (_CRD instance_)
- Updates on a `TaskRun` resource (_CRD instance_)

When the controller reconciles it:

- Looks for any existing owned `TaskRuns` and updates its parent `BuildRun` status.
- Retrieves the specified `SA` and sets this with the specify output secret on the `Build` resource.
- If one does not exist, it generates a new tekton `TaskRun` and sets a reference to this resource(_as a child of the controller_).
- On any subsequent updates on the `TaskRun`, the controller will update the parent `BuildRun` resource instance.

## Configuring a BuildRun

The `BuildRun` definition supports the following fields:

- Required:
  - [`apiVersion`](https://kubernetes.io/docs/concepts/overview/working-with-objects/kubernetes-objects/#required-fields) - Specifies the API version, for example `shipwright.io/v1beta1`.
  - [`kind`](https://kubernetes.io/docs/concepts/overview/working-with-objects/kubernetes-objects/#required-fields) - Specifies the Kind type, for example `BuildRun`.
  - [`metadata`](https://kubernetes.io/docs/concepts/overview/working-with-objects/kubernetes-objects/#required-fields) - Metadata that identify the CRD instance, for example the name of the `BuildRun`.

- Optional:
  - `spec.build.name` - Specifies an existing `Build` resource instance to use.
  - `spec.build.spec` - Specifies an embedded (transient) Build resource to use.
  - `spec.serviceAccount` - Refers to the SA to use when building the image. (_defaults to the `default` SA_)
  - `spec.timeout` - Defines a custom timeout. The value needs to be parsable by [ParseDuration](https://golang.org/pkg/time/#ParseDuration), for example, `5m`. The value overwrites the value that is defined in the `Build`.
  - `spec.paramValues` - Refers to a name-value(s) list to specify values for `parameters` defined in the `BuildStrategy`. This value overwrites values defined with the same name in the Build.
  - `spec.output.image` - Refers to a custom location where the generated image would be pushed. The value will overwrite the `output.image` value defined in `Build`. (**Note**: other properties of the output, for example, the credentials, cannot be specified in the buildRun spec. )
  - `spec.output.pushSecret` - Reference an existing secret to get access to the container registry. This secret will be added to the service account along with the ones requested by the `Build`.
  - `spec.output.timestamp` - Overrides the output timestamp configuration of the referenced build to instruct the build to change the output image creation timestamp to the specified value. When omitted, the respective build strategy tool defines the output image timestamp.
  - `spec.output.vulnerabilityScan` - Overrides the output vulnerabilityScan configuration of the referenced build to run the vulnerability scan for the generated image.
  - `spec.env` - Specifies additional environment variables that should be passed to the build container. Overrides any environment variables that are specified in the `Build` resource. The available variables depend on the tool used by the chosen build strategy.
  - `spec.nodeSelector` - Specifies a selector which must match a node's labels for the build pod to be scheduled on that node. If nodeSelectors are specified in both a `Build` and `BuildRun`, `BuildRun` values take precedence.
  - `spec.tolerations` - Specifies the tolerations for the build pod. Only `key`, `value`, and `operator` are supported. Only `NoSchedule` taint `effect` is supported. If tolerations are specified in both a `Build` and `BuildRun`, `BuildRun` values take precedence.

**Note**: The `spec.build.name` and `spec.build.spec` are mutually exclusive. Furthermore, the overrides for `timeout`, `paramValues`, `output`, and `env` can only be combined with `spec.build.name`, but **not** with `spec.build.spec`.

### Defining the Build Reference

A `BuildRun` resource can reference a `Build` resource, that indicates what image to build. For example:

```yaml
apiVersion: shipwright.io/v1beta1
kind: BuildRun
metadata:
  name: buildpack-nodejs-buildrun-namespaced
spec:
  build:
    name: buildpack-nodejs-build-namespaced
```

### Defining the Build Specification

A complete `BuildSpec` can be embedded into the `BuildRun` for the build.

```yaml
apiVersion: shipwright.io/v1beta1
kind: BuildRun
metadata:
  name: standalone-buildrun
spec:
  build:
    spec:
      source:
        type: Git
        git:
          url: https://github.com/shipwright-io/sample-go.git
        contextDir: source-build
      strategy:
        kind: ClusterBuildStrategy
        name: buildpacks-v3
      output:
        image: foo/bar:latest
```

### Defining the Build Source

BuildRun's support the specification of a Local type source. This is useful for working on development mode, without forcing a user to commit/push changes to their related version control system. For more information please refer to [SHIP 0016 - enabling local source code](https://github.com/shipwright-io/community/blob/main/ships/0016-enable-local-source-code-support.md).

```yaml
apiVersion: shipwright.io/v1beta1
kind: BuildRun
metadata:
  name: local-buildrun
spec:
  build:
    name: a-build
  source:
    type: Local
    local:
      name: local-source
      timeout: 3m
```

### Defining ParamValues

A `BuildRun` resource can define _paramValues_ for parameters specified in the build strategy. If a value has been provided for a parameter with the same name in the `Build` already, then the value from the `BuildRun` will have precedence.

For example, the following `BuildRun` overrides the value for _sleep-time_ param, which is defined in the _a-build_ `Build` resource.

```yaml
---
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

---
apiVersion: shipwright.io/v1beta1
kind: BuildRun
metadata:
  name: a-buildrun
  namespace: a-namespace
spec:
  build:
    name: a-build
  paramValues:
  - name: cache
    value: registry
```

See more about _paramValues_ usage in the related [Build](./build.md#defining-paramvalues) resource docs.

### Defining the ServiceAccount

A `BuildRun` resource can define a serviceaccount to use. Usually this SA will host all related secrets referenced on the `Build` resource, for example:

```yaml
apiVersion: shipwright.io/v1beta1
kind: BuildRun
metadata:
  name: buildpack-nodejs-buildrun-namespaced
spec:
  build:
    name: buildpack-nodejs-build-namespaced
  serviceAccount: pipeline
```

You can also set the value of `spec.serviceAccount` to `".generate"`. This will generate the service account during runtime for you. The name of the generated service account is the same as that of the BuildRun.

**Note**: When the service account is not defined, the `BuildRun` uses the `pipeline` service account if it exists in the namespace, and falls back to the `default` service account.

### Defining Retention Parameters

A `Buildrun` resource can specify how long a completed BuildRun can exist. Instead of manually cleaning up old BuildRuns, retention parameters provide an alternate method for cleaning up BuildRuns automatically.

As part of the buildrun retention parameters, we have the following fields:

- `retention.ttlAfterFailed` - Specifies the duration for which a failed buildrun can exist.
- `retention.ttlAfterSucceeded` - Specifies the duration for which a successful buildrun can exist.

An example of a user using buildrun TTL parameters.

```yaml
apiVersion: shipwright.io/v1beta1
kind: BuildRun
metadata:
  name: buidrun-retention-ttl
spec:
  build:
    name: build-retention-ttl
  retention:
    ttlAfterFailed: 10m
    ttlAfterSucceeded: 10m
```

**Note**: In case TTL values are defined in buildrun specifications as well as build specifications, priority will be given to the values defined in the buildrun specifications.

### Defining Volumes

`BuildRuns` can declare `volumes`. They must override `volumes` defined by the according `BuildStrategy`. If a `volume` is not `overridable` then the `BuildRun` will eventually fail.

In case `Build` and `BuildRun` that refers to this `Build` override the same `volume`, one that is defined in the `BuildRun` is the one used eventually.

`Volumes` follow the declaration of [Pod Volumes](https://kubernetes.io/docs/concepts/storage/volumes/), so all the usual `volumeSource` types are supported.

Here is an example of `BuildRun` object that overrides `volumes`:

```yaml
apiVersion: shipwright.io/v1beta1
kind: BuildRun
metadata:
  name: buildrun-name
spec:
  build:
    name: build-name
  volumes:
    - name: volume-name
      configMap:
        name: test-config
```

## Canceling a `BuildRun`

To cancel a `BuildRun` that's currently executing, update its status to mark it as canceled.

When you cancel a `BuildRun`, the underlying `TaskRun` is marked as canceled per the [Tekton cancel `TaskRun` feature](https://github.com/tektoncd/pipeline/blob/main/docs/taskruns.md).

Example of canceling a `BuildRun`:

```yaml
apiVersion: shipwright.io/v1beta1
kind: BuildRun
metadata:
  name: buildpack-nodejs-buildrun-namespaced
spec:
  # [...]
  state: "BuildRunCanceled"
```

## Automatic `BuildRun` deletion

We have two controllers that ensure that buildruns can be deleted automatically if required. This is ensured by adding `retention` parameters in either the build specifications or the buildrun specifications.

- Buildrun TTL parameters: These are used to make sure that buildruns exist for a fixed duration of time after completiion.
  - `buildrun.spec.retention.ttlAfterFailed`: The buildrun is deleted if the mentioned duration of time has passed and the buildrun has failed.
  - `buildrun.spec.retention.ttlAfterSucceeded`: The buildrun is deleted if the mentioned duration of time has passed and the buildrun has succeeded.
- Build TTL parameters: These are used to make sure that related buildruns exist for a fixed duration of time after completion.
  - `build.spec.retention.ttlAfterFailed`: The buildrun is deleted if the mentioned duration of time has passed and the buildrun has failed.
  - `build.spec.retention.ttlAfterSucceeded`: The buildrun is deleted if the mentioned duration of time has passed and the buildrun has succeeded.
- Build Limit parameters: These are used to make sure that related buildruns exist for a fixed duration of time after completiion.
  - `build.spec.retention.succeededLimit` - Defines number of succeeded BuildRuns for a Build that can exist.
  - `build.spec.retention.failedLimit` - Defines number of failed BuildRuns for a Build that can exist.

## Specifying Environment Variables

An example of a `BuildRun` that specifies environment variables:

```yaml
apiVersion: shipwright.io/v1beta1
kind: BuildRun
metadata:
  name: buildpack-nodejs-buildrun-namespaced
spec:
  build:
    name: buildpack-nodejs-build-namespaced
  env:
    - name: EXAMPLE_VAR_1
      value: "example-value-1"
    - name: EXAMPLE_VAR_2
      value: "example-value-2"
```

Example of a `BuildRun` that uses the Kubernetes Downward API to expose a `Pod` field as an environment variable:

```yaml
apiVersion: shipwright.io/v1beta1
kind: BuildRun
metadata:
  name: buildpack-nodejs-buildrun-namespaced
spec:
  build:
    name: buildpack-nodejs-build-namespaced
  env:
    - name: POD_NAME
      valueFrom:
        fieldRef:
          fieldPath: metadata.name
```

Example of a `BuildRun` that uses the Kubernetes Downward API to expose a `Container` field as an environment variable:

```yaml
apiVersion: shipwright.io/v1beta1
kind: BuildRun
metadata:
  name: buildpack-nodejs-buildrun-namespaced
spec:
  build:
    name: buildpack-nodejs-build-namespaced
  env:
    - name: MEMORY_LIMIT
      valueFrom:
        resourceFieldRef:
          containerName: my-container
          resource: limits.memory
```

## BuildRun Status

The `BuildRun` resource is updated as soon as the current image building status changes:

```sh
$ kubectl get buildrun buildpacks-v3-buildrun
NAME                    SUCCEEDED   REASON    MESSAGE   STARTTIME   COMPLETIONTIME
buildpacks-v3-buildrun  Unknown     Pending   Pending   1s
```

And finally:

```sh
$ kubectl get buildrun buildpacks-v3-buildrun
NAME                    SUCCEEDED   REASON      MESSAGE                              STARTTIME   COMPLETIONTIME
buildpacks-v3-buildrun  True        Succeeded   All Steps have completed executing   4m28s       16s
```

The above allows users to get an overview of the building mechanism state.

### Understanding the state of a BuildRun

A `BuildRun` resource stores the relevant information regarding the object's state under `status.conditions`.

[Conditions](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties) allow users to quickly understand the resource state without needing to understand resource-specific details.

For the `BuildRun`, we use a Condition of the type `Succeeded`, which is a well-known type for resources that run to completion.

The `status.conditions` hosts different fields, like `status`, `reason` and `message`. Users can expect these fields to be populated with relevant information.

The following table illustrates the different states a BuildRun can have under its `status.conditions`:

| Status  | Reason                                  | CompletionTime is set | Description                                                                                                                                                                                                                                                                                           |
|---------|-----------------------------------------|-----------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| Unknown | Pending                                 | No                    | The BuildRun is waiting on a Pod in status Pending.                                                                                                                                                                                                                                                   |
| Unknown | Running                                 | No                    | The BuildRun has been validated and started to perform its work.                                                                                                                                                                                                                                      |
| Unknown | Running                                 | No                    | The BuildRun has been validated and started to perform its work.                                                                                                                                                                                                                                      |
| Unknown | BuildRunCanceled                        | No                    | The user requested the BuildRun to be canceled. This results in the BuildRun controller requesting the TaskRun be canceled. Cancellation has not been done yet.                                                                                                                                       |
| True    | Succeeded                               | Yes                   | The BuildRun Pod is done.                                                                                                                                                                                                                                                                             |
| False   | Failed                                  | Yes                   | The BuildRun failed in one of the steps.                                                                                                                                                                                                                                                              |
| False   | BuildRunTimeout                         | Yes                   | The BuildRun timed out.                                                                                                                                                                                                                                                                               |
| False   | UnknownStrategyKind                     | Yes                   | The Build specified strategy Kind is unknown. (_options: ClusterBuildStrategy or BuildStrategy_)                                                                                                                                                                                                      |
| False   | ClusterBuildStrategyNotFound            | Yes                   | The referenced cluster strategy was not found in the cluster.                                                                                                                                                                                                                                         |
| False   | BuildStrategyNotFound                   | Yes                   | The referenced namespaced strategy was not found in the cluster.                                                                                                                                                                                                                                      |
| False   | SetOwnerReferenceFailed                 | Yes                   | Setting ownerreferences from the BuildRun to the related TaskRun failed.                                                                                                                                                                                                                              |
| False   | TaskRunIsMissing                        | Yes                   | The BuildRun related TaskRun was not found.                                                                                                                                                                                                                                                           |
| False   | TaskRunGenerationFailed                 | Yes                   | The generation of a TaskRun spec failed.                                                                                                                                                                                                                                                              |
| False   | MissingParameterValues                  | Yes                   | No value has been provided for some parameters that are defined in the build strategy without any default. Values for those parameters must be provided through the Build or the BuildRun.                                                                                                            |
| False   | RestrictedParametersInUse               | Yes                   | A value for a system parameter was provided. This is not allowed.                                                                                                                                                                                                                                     |
| False   | UndefinedParameter                      | Yes                   | A value for a parameter was provided that is not defined in the build strategy.                                                                                                                                                                                                                       |
| False   | WrongParameterValueType                 | Yes                   | A value was provided for a build strategy parameter using the wrong type. The parameter is defined as `array` or `string` in the build strategy. Depending on that, you must provide `values` or a direct value.                                                                                      |
| False   | InconsistentParameterValues             | Yes                   | A value for a parameter contained more than one of `value`, `configMapValue`, and `secretValue`. Any values including array items must only provide one of them.                                                                                                                                      |
| False   | EmptyArrayItemParameterValues           | Yes                   | An item inside the `values` of an array parameter contained none of `value`, `configMapValue`, and `secretValue`. Exactly one of them must be provided. Null array items are not allowed.                                                                                                             |
| False   | IncompleteConfigMapValueParameterValues | Yes                   | A value for a parameter contained a `configMapValue` where the `name` or the `value` were empty. You must specify them to point to an existing ConfigMap key in your namespace.                                                                                                                       |
| False   | IncompleteSecretValueParameterValues    | Yes                   | A value for a parameter contained a `secretValue` where the `name` or the `value` were empty. You must specify them to point to an existing Secret key in your namespace.                                                                                                                             |
| False   | ServiceAccountNotFound                  | Yes                   | The referenced service account was not found in the cluster.                                                                                                                                                                                                                                          |
| False   | BuildRegistrationFailed                 | Yes                   | The related Build in the BuildRun is in a Failed state.                                                                                                                                                                                                                                               |
| False   | BuildNotFound                           | Yes                   | The related Build in the BuildRun was not found.                                                                                                                                                                                                                                                      |
| False   | BuildRunCanceled                        | Yes                   | The BuildRun and underlying TaskRun were canceled successfully.                                                                                                                                                                                                                                       |
| False   | BuildRunNameInvalid                     | Yes                   | The defined `BuildRun` name (`metadata.name`) is invalid. The `BuildRun` name should be a [valid label value](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#syntax-and-character-set).                                                                                    |
| False   | BuildRunNoRefOrSpec                     | Yes                   | BuildRun does not have either `spec.build.name` or `spec.build.spec` defined. There is no connection to a Build specification.                                                                                                                                                                        |
| False   | BuildRunAmbiguousBuild                  | Yes                   | The defined `BuildRun` uses both `spec.build.name` and `spec.build.spec`. Only one of them is allowed at the same time.                                                                                                                                                                               |
| False   | BuildRunBuildFieldOverrideForbidden     | Yes                   | The defined `BuildRun` uses an override (e.g. `timeout`, `paramValues`, `output`, or `env`) in combination with `spec.build.spec`, which is not allowed. Use the `spec.build.spec` to directly specify the respective value.                                                                          |
| False   | PodEvicted                              | Yes                   | The BuildRun Pod was evicted from the node it was running on. See [API-initiated Eviction](https://kubernetes.io/docs/concepts/scheduling-eviction/api-eviction/) and [Node-pressure Eviction](https://kubernetes.io/docs/concepts/scheduling-eviction/node-pressure-eviction/) for more information. |
| False   | StepOutOfMemory                         | Yes                   | The BuildRun Pod failed because a step went out of memory.                                                                                                                                                                                                                                            |

**Note**: We heavily rely on the Tekton TaskRun [Conditions](https://github.com/tektoncd/pipeline/blob/main/docs/taskruns.md#monitoring-execution-status) for populating the BuildRun ones, with some exceptions.

### Understanding failed BuildRuns

To make it easier for users to understand why did a BuildRun failed, users can infer the pod and container where the failure took place from the `status.failureDetails` field.

In addition, the `status.conditions` hosts a compacted message under the `message` field that contains the `kubectl` command to trigger and retrieve the logs.

The `status.failureDetails` field also includes a detailed failure reason and message, if the build strategy provides them.

Example of failed BuildRun:

```yaml
# [...]
status:
  # [...]
  failureDetails:
    location:
      container: step-source-default
      pod: baran-build-buildrun-gzmv5-b7wbf-pod-bbpqr
    message: The source repository does not exist, or you have insufficient permission
      to access it.
    reason: GitRemotePrivate
```

### Understanding failed BuildRuns due to VulnerabilitiesFound

A buildrun can be failed, if the vulnerability scan finds vulnerabilities in the generated image and `failOnFinding` is set to true in the `vulnerabilityScan`. For setting `vulnerabilityScan`, see [here](build.md#defining-the-vulnerabilityscan).

Example of failed BuildRun due to vulnerabilities present in the image:

```yaml
# [...]
status:
  # [...]
  conditions:
  - type: Succeeded
    lastTransitionTime: "2024-03-12T20:00:38Z"
    status: "False"
    reason: VulnerabilitiesFound
    message: "Vulnerabilities have been found in the output image. For detailed information, check buildrun status or see kubectl --namespace default logs vuln-s6skc-v7wd2-pod --container step-image-processing"
```

#### Understanding failed git-source step

All git-related operations support error reporting via `status.failureDetails`. The following table explains the possible
error reasons:

| Reason                        | Description                                                                                                                                                        |
|-------------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `GitAuthInvalidUserOrPass`    | Basic authentication has failed. Check your username or password. **Note**: GitHub requires a personal access token instead of your regular password.              |
| `GitAuthInvalidKey`           | The key is invalid for the specified target. Please make sure that the Git repository exists, you have sufficient permissions, and the key is in the right format. |
| `GitRevisionNotFound`         | The remote revision does not exist. Check the revision specified in your Build.                                                                                    |
| `GitRemoteRepositoryNotFound` | The source repository does not exist, or you have insufficient permissions to access it.                                                                           |
| `GitRemoteRepositoryPrivate`  | You are trying to access a non-existing or private repository without having sufficient permissions to access it via HTTPS.                                        |
| `GitBasicAuthIncomplete`      | Basic Auth incomplete: Both username and password must be configured.                                                                                              |
| `GitSSHAuthUnexpected`        | Credential/URL inconsistency: SSH credentials were provided, but the URL is not an SSH Git URL.                                                                    |
| `GitSSHAuthExpected`          | Credential/URL inconsistency: No SSH credentials provided, but the URL is an SSH Git URL.                                                                          |
| `GitError`                    | The specific error reason is unknown. Check the error message for more information.                                                                                |

### Step Results in BuildRun Status

After completing a `BuildRun`, the `.status` field contains the results (`.status.taskResults`) emitted from the `TaskRun` steps generated by the `BuildRun` controller as part of processing the `BuildRun`. These results contain valuable metadata for users, like the _image digest_ or the _commit sha_ of the source code used for building.
The results from the source step will be surfaced to the `.status.sources`, and the results from
the [output step](buildstrategies.md#system-results) will be surfaced to the `.status.output` field of a `BuildRun`.

Example of a `BuildRun` with surfaced results for `git` source (note that the `branchName` is only included if the Build does not specify any `revision`):

```yaml
# [...]
status:
  buildSpec:
    # [...]
  output:
    digest: sha256:07626e3c7fdd28d5328a8d6df8d29cd3da760c7f5e2070b534f9b880ed093a53
    size: 1989004
  sources:
  - name: default
    git:
      commitAuthor: xxx xxxxxx
      commitSha: f25822b85021d02059c9ac8a211ef3804ea8fdde
      branchName: main
```

Another example of a `BuildRun` with surfaced results for local source code(`ociArtifact`) source:

```yaml
# [...]
status:
  buildSpec:
    # [...]
  output:
    digest: sha256:07626e3c7fdd28d5328a8d6df8d29cd3da760c7f5e2070b534f9b880ed093a53
    size: 1989004
  sources:
  - name: default
    ociArtifact:
      digest: sha256:0f5e2070b534f9b880ed093a537626e3c7fdd28d5328a8d6df8d29cd3da760c7
```

**Note**: The digest and size of the output image are only included if the build strategy provides them. See [System results](buildstrategies.md#system-results).

Another example of a `BuildRun` with surfaced results for vulnerability scanning.

```yaml
# [...]
status:
  buildSpec:
    # [...]
  status:
  output:
    digest: sha256:1023103
    size: 12310380
    vulnerabilities:
    - id: CVE-2022-12345
      severity: high
    - id: CVE-2021-54321
      severity: medium
```

**Note**: The vulnerability scan will only run if it is specified in the build or buildrun spec. See [Defining the `vulnerabilityScan`](build.md#defining-the-vulnerabilityscan).

### Build Snapshot

For every BuildRun controller reconciliation, the `buildSpec` in the status of the `BuildRun` is updated if an existing owned `TaskRun` is present. During this update, a `Build` resource snapshot is generated and embedded into the `status.buildSpec` path of the `BuildRun`. A `buildSpec` is just a copy of the original `Build` spec, from where the `BuildRun` executed a particular image build. The snapshot approach allows developers to see the original `Build` configuration.

## Relationship with Tekton Tasks

The `BuildRun` resource abstracts the image construction by delegating this work to the Tekton Pipeline [TaskRun](https://github.com/tektoncd/pipeline/blob/main/docs/taskruns.md). Compared to a Tekton Pipeline [Task](https://github.com/tektoncd/pipeline/blob/main/docs/tasks.md), a `TaskRun` runs all `steps` until completion of the `Task` or until a failure occurs in the `Task`.

During the Reconcile, the `BuildRun` controller will generate a new `TaskRun`. The controller will embed in the `TaskRun` `Task` definition the required `steps` to execute during the execution. These `steps` are defined in the strategy defined in the `Build` resource, either a `ClusterBuildStrategy` or a `BuildStrategy`.
