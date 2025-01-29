// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package testbeta

// MinimalBuildahBuildWithEnvVars defines a simple
// Build with a source, strategy, and env vars
const MinimalBuildahBuildWithEnvVars = `
apiVersion: shipwright.io/v1beta1
kind: Build
metadata:
  name: buildah
spec:
  source:
    type: Git
    git:
      url: "https://github.com/shipwright-io/sample-go"
  strategy:
    name: buildah
    kind: ClusterBuildStrategy
  paramValues:
  - name: dockerfile
    value: Dockerfile
  env:
    - name: MY_VAR_1
      value: "my-var-1-build-value"
    - name: MY_VAR_2
      valueFrom:
        fieldRef:
          fieldPath: "my-fieldpath"
`

// MinimalBuildahBuild defines a simple
// Build with a source and a strategy
const MinimalBuildahBuild = `
apiVersion: shipwright.io/v1beta1
kind: Build
metadata:
  name: buildah
spec:
  source:
    type: Git
    git:
      url: "https://github.com/shipwright-io/sample-go"
  strategy:
    name: buildah
    kind: ClusterBuildStrategy
  paramValues:
  - name: dockerfile
    value: Dockerfile
`

// MinimalBuildahBuildWithNodeSelector defines a simple
// Build with a source, strategy, and nodeSelector
const MinimalBuildahBuildWithNodeSelector = `
apiVersion: shipwright.io/v1beta1
kind: Build
metadata:
  name: buildah
spec:
  source:
    type: Git
    git:
      url: "https://github.com/shipwright-io/sample-go"
  strategy:
    name: buildah
    kind: ClusterBuildStrategy
  nodeSelector:
    kubernetes.io/arch: amd64
`

// BuildahBuildWithOutput defines a simple
// Build with a source, strategy and output
const BuildahBuildWithOutput = `
apiVersion: shipwright.io/v1beta1
kind: Build
metadata:
  name: buildah
  namespace: build-test
spec:
  source:
    type: Git
    git:
      url: "https://github.com/shipwright-io/sample-go"
  strategy:
    name: buildah
  output:
    image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app
`

// BuildahBuildWithAnnotationAndLabel defines a simple
// Build with a source, strategy, output,
// annotations and labels
const BuildahBuildWithAnnotationAndLabel = `
apiVersion: shipwright.io/v1beta1
kind: Build
metadata:
  name: buildah
spec:
  source:
    type: Git
    git:
      url: "https://github.com/shipwright-io/sample-go"
  strategy:
    name: buildah
    kind: ClusterBuildStrategy
  paramValues:
  - name: dockerfile
    value: Dockerfile
  output:
    image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app
    labels:
      "maintainer": "team@my-company.com"
    annotations:
      "org.opencontainers.image.url": https://my-company.com/images
`

// BuildahBuildWithMultipleAnnotationAndLabel defines a
// Build with a source, strategy, output,
// multiple annotations and labels
const BuildahBuildWithMultipleAnnotationAndLabel = `
apiVersion: shipwright.io/v1beta1
kind: Build
metadata:
  name: buildah
spec:
  source:
    type: Git
    git:
      url: "https://github.com/shipwright-io/sample-go"
  strategy:
    name: buildah
    kind: ClusterBuildStrategy
  paramValues:
  - name: dockerfile
    value: Dockerfile
  output:
    image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app
    labels:
      "maintainer": "team@my-company.com"
      "description": "This is my cool image"
    annotations:
      "org.opencontainers.image.url": https://my-company.com/images
      "org.opencontainers.image.source": "https://github.com/org/repo"
`

// BuildpacksBuildWithBuilderAndTimeOut defines a Build with
// source, strategy, builder, output and
// timeout
const BuildpacksBuildWithBuilderAndTimeOut = `
apiVersion: shipwright.io/v1beta1
kind: Build
metadata:
  name: buildpacks-v3
  namespace: build-test
spec:
  source:
    type: Git
    git:
      url: "https://github.com/shipwright-io/sample-go"
    contextDir: docker-build
  strategy:
    name: buildpacks-v3
    kind: ClusterBuildStrategy
  output:
    image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app
  timeout: 30s
`

// BuildahBuildWithTimeOut defines a Build for
// Buildah with source, strategy, output and
// timeout
const BuildahBuildWithTimeOut = `
apiVersion: shipwright.io/v1beta1
kind: Build
metadata:
  name: buildah
  namespace: build-test
spec:
  source:
    type: Git
    git:
      url: "https://github.com/shipwright-io/sample-go"
  strategy:
    name: buildah
  output:
    image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app
  timeout: 30s
`

// BuildBSMinimal defines a Build with a BuildStrategy
const BuildBSMinimal = `
apiVersion: shipwright.io/v1beta1
kind: Build
spec:
  source:
    type: Git
    git:
      url: "https://github.com/shipwright-io/sample-go"
  strategy:
    kind: BuildStrategy
  output:
    image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app
`

// BuildBSMinimalNoSource defines a Build with a BuildStrategy without source
const BuildBSMinimalNoSource = `
apiVersion: shipwright.io/v1beta1
kind: Build
spec:
  strategy:
    kind: BuildStrategy
  output:
    image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app
`

// BuildCBSMinimal defines a Build with a
// ClusterBuildStrategy
const BuildCBSMinimal = `
apiVersion: shipwright.io/v1beta1
kind: Build
spec:
  source:
    type: Git
    git:
      url: "https://github.com/shipwright-io/sample-go"
    contextDir: docker-build
  strategy:
    kind: ClusterBuildStrategy
  output:
    image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app
`

// BuildCBSMinimalWithFakeSecret defines a Build with a
// ClusterBuildStrategy and an not existing secret
const BuildCBSMinimalWithFakeSecret = `
apiVersion: shipwright.io/v1beta1
kind: Build
spec:
  source:
    type: Git
    git:
      url: "https://github.com/shipwright-io/sample-go"
  strategy:
    kind: ClusterBuildStrategy
  output:
    image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app
    pushSecret: fake-secret
`

// BuildWithOutputRefSecret defines a Build with a
// referenced secret under spec.output
const BuildWithOutputRefSecret = `
apiVersion: shipwright.io/v1beta1
kind: Build
spec:
  source:
    type: Git
    git:
      url: "https://github.com/shipwright-io/sample-go"
  strategy:
    kind: ClusterBuildStrategy
  output:
    image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app
    pushSecret: output-secret
  timeout: 5s
`

// BuildWithSourceRefSecret defines a Build with a
// referenced secret under spec.source
const BuildWithSourceRefSecret = `
apiVersion: shipwright.io/v1beta1
kind: Build
spec:
  source:
    type: Git
    git:
      url: "https://github.com/shipwright-io/sample-go"
      cloneSecret: source-secret
  strategy:
    kind: ClusterBuildStrategy
  output:
    image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app
  timeout: 5s
`

// BuildWithBuilderRefSecret defines a Build with a
// referenced secret under spec.builder
const BuildWithBuilderRefSecret = `
apiVersion: shipwright.io/v1beta1
kind: Build
spec:
  source:
    type: Git
    git:
      url: "https://github.com/shipwright-io/sample-go"
  paramValues:
  - name: builder-image
    value: heroku/builder:22
  strategy:
    kind: ClusterBuildStrategy
  output:
    image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app
  timeout: 5s
`

// BuildWithMultipleRefSecrets defines a Build with
// multiple referenced secrets under spec
const BuildWithMultipleRefSecrets = `
apiVersion: shipwright.io/v1beta1
kind: Build
spec:
  source:
    type: Git
    git:
      url: "https://github.com/shipwright-io/sample-go"
      cloneSecret: source-secret
  paramValues:
  - name: builder-image
    value: heroku/builder:22
  strategy:
    kind: ClusterBuildStrategy
  output:
    image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app
  timeout: 5s
`

// BuildCBSWithShortTimeOut defines a Build with a
// ClusterBuildStrategy and a short timeout
const BuildCBSWithShortTimeOut = `
apiVersion: shipwright.io/v1beta1
kind: Build
spec:
  source:
    type: Git
    git:
      url: "https://github.com/shipwright-io/sample-go"
  strategy:
    kind: ClusterBuildStrategy
  output:
    image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app
  timeout: 5s
`

// BuildCBSWithShortTimeOutAndRefOutputSecret defines a Build with a
// ClusterBuildStrategy, a short timeout and an output secret
const BuildCBSWithShortTimeOutAndRefOutputSecret = `
apiVersion: shipwright.io/v1beta1
kind: Build
spec:
  source:
    type: Git
    git:
      url: "https://github.com/shipwright-io/sample-go"
  strategy:
    kind: ClusterBuildStrategy
  output:
    image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app
    pushSecret: foobarsecret
  timeout: 5s
`

// BuildCBSWithWrongURL defines a Build with a
// ClusterBuildStrategy and a non-existing url
const BuildCBSWithWrongURL = `
apiVersion: shipwright.io/v1beta1
kind: Build
metadata:
  annotations:
    build.shipwright.io/verify.repository: "true"
spec:
  source:
    type: Git
    git:
      url: "https://github.foobar.com/sbose78/taxi"
  strategy:
    kind: ClusterBuildStrategy
  output:
    image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app
`

// BuildCBSWithVerifyRepositoryAnnotation defines a Build
// with the verify repository annotation key
const BuildCBSWithVerifyRepositoryAnnotation = `
apiVersion: shipwright.io/v1beta1
kind: Build
metadata:
  annotations:
    build.shipwright.io/verify.repository: ""
spec:
  strategy:
    kind: ClusterBuildStrategy
  output:
    image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app
`

// BuildCBSWithoutVerifyRepositoryAnnotation defines a minimal
// Build without source url and annotation
const BuildCBSWithoutVerifyRepositoryAnnotation = `
apiVersion: shipwright.io/v1beta1
kind: Build
spec:
  strategy:
    kind: ClusterBuildStrategy
  output:
    image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app
`

// BuildCBSWithBuildRunDeletion defines a Build with a
// ClusterBuildStrategy and the annotation for automatic BuildRun
// deletion
const BuildCBSWithBuildRunDeletion = `
apiVersion: shipwright.io/v1beta1
kind: Build
spec:
  source:
    type: Git
    git:
      url: "https://github.com/shipwright-io/sample-go"
  strategy:
    kind: ClusterBuildStrategy
  retention:
    atBuildDeletion: true
  output:
    image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app
`

// BuildWithSleepTimeParam defines a Build with a parameter
const BuildWithSleepTimeParam = `
apiVersion: shipwright.io/v1beta1
kind: Build
spec:
  source:
    type: Git
    git:
      url: "https://github.com/shipwright-io/sample-go"
  paramValues:
  - name: sleep-time
    value: "30"
  strategy:
    kind: BuildStrategy
  output:
    image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app
`

// BuildWithArrayParam defines a Build with an array parameter
const BuildWithArrayParam = `
apiVersion: shipwright.io/v1beta1
kind: Build
spec:
  source:
    type: Git
    git:
      url: "https://github.com/shipwright-io/sample-go"
  paramValues:
  - name: array-param
    values:
    - value: "3"
    - value: "-1"
  strategy:
    kind: BuildStrategy
  output:
    image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app
`

// BuildWithConfigMapSecretParams defines a Build with parameter values referencing a ConfigMap and Secret
const BuildWithConfigMapSecretParams = `
apiVersion: shipwright.io/v1beta1
kind: Build
spec:
  source:
    type: Git
    git:
      url: "https://github.com/shipwright-io/sample-go"
  paramValues:
  - name: array-param
    values:
    - value: "3"
    - configMapValue:
        name: a-configmap
        key: a-cm-key
    - value: "-1"
  - name: sleep-time
    secretValue:
      name: a-secret
      key: a-secret-key
  strategy:
    kind: BuildStrategy
  output:
    image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app
`

// BuildWithRestrictedParam defines a Build using params that are reserved only
// for shipwright
const BuildWithRestrictedParam = `
apiVersion: shipwright.io/v1beta1
kind: Build
spec:
  source:
    type: Git
    git:
      url: "https://github.com/shipwright-io/sample-go"
  paramValues:
  - name: shp-something
    value: "30"
  - name: DOCKERFILE
    value: "30"
  strategy:
    kind: BuildStrategy
  output:
    image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app
`

// BuildWithVulnerabilityScanOptions defines a Build with vulnerability scan options
const BuildWithVulnerabilityScanOptions = `
apiVersion: shipwright.io/v1beta1
kind: Build
spec:
  strategy:
    kind: ClusterBuildStrategy
  output:
    vulnerabilityScan:
      enabled: true
      failOnFinding: true
    image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app
`

// BuildWithVulnerabilityScanOptions defines a Build with vulnerability scan options
const BuildWithDisableVulnerabilityScan = `
apiVersion: shipwright.io/v1beta1
kind: Build
spec:
  strategy:
    kind: ClusterBuildStrategy
  output:
    vulnerabilityScan:
      enabled: false
    image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app
`

// MinimalBuild defines a simple
// Build with a strategy output
const MinimalBuild = `
apiVersion: shipwright.io/v1beta1
kind: Build
spec:
  strategy:
    kind: ClusterBuildStrategy
  output:
    image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app
`

// MinimalBuildWithNodeSelector defines a simple
// Build with a strategy, output, and NodeSelector
const MinimalBuildWithNodeSelector = `
apiVersion: shipwright.io/v1beta1
kind: Build
spec:
  strategy:
    kind: ClusterBuildStrategy
  output:
    image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app
  nodeSelector:
    kubernetes.io/arch: amd64
`

// MinimalBuildWithToleration defines a simple
// Build with a strategy, output, and a Toleration specified
const MinimalBuildWithToleration = `
apiVersion: shipwright.io/v1beta1
kind: Build
spec:
  strategy:
    kind: ClusterBuildStrategy
  output:
    image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app
  tolerations:
    - key: "build-test-key"
      operator: "Equal"
      value: "build-test-value"
`

// BuildWithUndefinedParameter defines a param that was not declared under the
// strategy parameters
const BuildWithUndefinedParam = `
apiVersion: shipwright.io/v1beta1
kind: Build
spec:
  source:
    type: Git
    git:
      url: "https://github.com/shipwright-io/sample-go"
  paramValues:
  - name: sleep-not
    value: "30"
  strategy:
    kind: BuildStrategy
  output:
    image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app
`

// BuildWithEmptyStringParam defines a param that with an empty string value
const BuildWithEmptyStringParam = `
apiVersion: shipwright.io/v1beta1
kind: Build
spec:
  source:
    type: Git
    git:
      url: "https://github.com/shipwright-io/sample-go"
  paramValues:
  - name: sleep-time
    value: ""
  strategy:
    kind: BuildStrategy
  output:
    image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app
`

// BuildWithUndefinedParamAndCBS defines a param that was not declared under the
// strategy parameters of a ClusterBuildStrategy
const BuildWithUndefinedParamAndCBS = `
apiVersion: shipwright.io/v1beta1
kind: Build
spec:
  source:
    type: Git
    git:
      url: "https://github.com/shipwright-io/sample-go"
  paramValues:
  - name: sleep-not
    value: "30"
  strategy:
    kind: ClusterBuildStrategy
  output:
    image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app
`

// BuildWithSleepTimeParamAndCBS defines a Build with a parameter
const BuildWithSleepTimeParamAndCBS = `
apiVersion: shipwright.io/v1beta1
kind: Build
spec:
  source:
    type: Git
    git:
      url: "https://github.com/shipwright-io/sample-go"
  paramValues:
  - name: sleep-time
    value: "30"
  strategy:
    kind: ClusterBuildStrategy
  output:
    image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app
`

// MinimalBuildWithRetentionTTLFive defines a simple
// Build with a source, a strategy and ttl
const MinimalBuildWithRetentionTTLFive = `
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
    image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app
  retention:
    ttlAfterFailed: 5s
    ttlAfterSucceeded: 5s
`

// MinimalBuildWithRetentionLimitOne defines a simple
// Build with a source, a strategy and limits set as 1
const MinimalBuildWithRetentionLimitOne = `
apiVersion: shipwright.io/v1beta1
kind: Build
metadata:
  name: build-retention-limit
spec:
  source:
    type: Git
    git:
      url: "https://github.com/shipwright-io/sample-go"
    contextDir: docker-build
  strategy:
    kind: ClusterBuildStrategy
  output:
    image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app
  retention:
    failedLimit: 1
    succeededLimit: 1
`

// MinimalBuildWithRetentionLimitDiff defines a simple Build with a source,
// a strategy and different failed and succeeded limits
const MinimalBuildWithRetentionLimitDiff = `
apiVersion: shipwright.io/v1beta1
kind: Build
metadata:
  name: build-retention-limit
spec:
  source:
    type: Git
    git:
      url: "https://github.com/shipwright-io/sample-go"
    contextDir: docker-build
  strategy:
    kind: ClusterBuildStrategy
  output:
    image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app
  retention:
    failedLimit: 1
    succeededLimit: 2
`

// MinimalBuildWithRetentionTTL defines a simple
// Build with a source, a strategy ttl
const MinimalBuildWithRetentionTTLOneMin = `
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
    image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app
  retention:
    ttlAfterFailed: 1m
    ttlAfterSucceeded: 1m
`
