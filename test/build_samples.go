// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package test

// MinimalBuildahBuildWithEnvVars defines a simple
// Build with a source, strategy, and env vars
const MinimalBuildahBuildWithEnvVars = `
apiVersion: shipwright.io/v1alpha1
kind: Build
metadata:
  name: buildah
spec:
  source:
    url: "https://github.com/shipwright-io/sample-go"
  strategy:
    name: buildah
    kind: ClusterBuildStrategy
  dockerfile: Dockerfile
  env:
    - name: MY_VAR_1
      value: "my-var-1-build-value"
    - name: MY_VAR_2
      value: "my-var-2-build-value"
`

// MinimalBuildahBuild defines a simple
// Build with a source and a strategy
const MinimalBuildahBuild = `
apiVersion: shipwright.io/v1alpha1
kind: Build
metadata:
  name: buildah
spec:
  source:
    url: "https://github.com/shipwright-io/sample-go"
  strategy:
    name: buildah
    kind: ClusterBuildStrategy
  dockerfile: Dockerfile
`

// BuildahBuildWithOutput defines a simple
// Build with a source, strategy and output
const BuildahBuildWithOutput = `
apiVersion: shipwright.io/v1alpha1
kind: Build
metadata:
  name: buildah
  namespace: build-test
spec:
  source:
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
apiVersion: shipwright.io/v1alpha1
kind: Build
metadata:
  name: buildah
spec:
  source:
    url: "https://github.com/shipwright-io/sample-go"
  strategy:
    name: buildah
    kind: ClusterBuildStrategy
  dockerfile: Dockerfile
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
apiVersion: shipwright.io/v1alpha1
kind: Build
metadata:
  name: buildah
spec:
  source:
    url: "https://github.com/shipwright-io/sample-go"
  strategy:
    name: buildah
    kind: ClusterBuildStrategy
  dockerfile: Dockerfile
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
apiVersion: shipwright.io/v1alpha1
kind: Build
metadata:
  name: buildpacks-v3
  namespace: build-test
spec:
  source:
    url: "https://github.com/shipwright-io/sample-go"
    contextDir: docker-build
  strategy:
    name: buildpacks-v3
    kind: ClusterBuildStrategy
  dockerfile: Dockerfile
  builder:
    image: heroku/buildpacks:18
  output:
    image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app
  timeout: 30s
`

// BuildahBuildWithTimeOut defines a Build for
// Buildah with source, strategy, output and
// timeout
const BuildahBuildWithTimeOut = `
apiVersion: shipwright.io/v1alpha1
kind: Build
metadata:
  name: buildah
  namespace: build-test
spec:
  source:
    url: "https://github.com/shipwright-io/sample-go"
  strategy:
    name: buildah
  output:
    image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app
  timeout: 30s
`

// BuildBSMinimal defines a Build with a BuildStrategy
const BuildBSMinimal = `
apiVersion: shipwright.io/v1alpha1
kind: Build
spec:
  source:
    url: "https://github.com/shipwright-io/sample-go"
  strategy:
    kind: BuildStrategy
  output:
    image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app
`

// BuildCBSMinimal defines a Build with a
// ClusterBuildStrategy
const BuildCBSMinimal = `
apiVersion: shipwright.io/v1alpha1
kind: Build
spec:
  source:
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
apiVersion: shipwright.io/v1alpha1
kind: Build
spec:
  source:
    url: "https://github.com/shipwright-io/sample-go"
  strategy:
    kind: ClusterBuildStrategy
  output:
    image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app
    credentials:
      name: fake-secret
`

// BuildWithOutputRefSecret defines a Build with a
// referenced secret under spec.output
const BuildWithOutputRefSecret = `
apiVersion: shipwright.io/v1alpha1
kind: Build
spec:
  source:
    url: "https://github.com/shipwright-io/sample-go"
  strategy:
    kind: ClusterBuildStrategy
  output:
    image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app
    credentials:
      name: output-secret
  timeout: 5s
`

// BuildWithSourceRefSecret defines a Build with a
// referenced secret under spec.source
const BuildWithSourceRefSecret = `
apiVersion: shipwright.io/v1alpha1
kind: Build
spec:
  source:
    url: "https://github.com/shipwright-io/sample-go"
    credentials:
      name: source-secret
  strategy:
    kind: ClusterBuildStrategy
  output:
    image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app
  timeout: 5s
`

// BuildWithBuilderRefSecret defines a Build with a
// referenced secret under spec.builder
const BuildWithBuilderRefSecret = `
apiVersion: shipwright.io/v1alpha1
kind: Build
spec:
  source:
    url: "https://github.com/shipwright-io/sample-go"
  builder:
    image: heroku/buildpacks:18
    credentials:
      name: builder-secret
  strategy:
    kind: ClusterBuildStrategy
  output:
    image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app
  timeout: 5s
`

// BuildWithMultipleRefSecrets defines a Build with
// multiple referenced secrets under spec
const BuildWithMultipleRefSecrets = `
apiVersion: shipwright.io/v1alpha1
kind: Build
spec:
  source:
    url: "https://github.com/shipwright-io/sample-go"
    credentials:
      name: source-secret
  builder:
    image: heroku/buildpacks:18
    credentials:
      name: builder-secret
  strategy:
    kind: ClusterBuildStrategy
  output:
    image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app
  timeout: 5s
`

// BuildCBSWithShortTimeOut defines a Build with a
// ClusterBuildStrategy and a short timeout
const BuildCBSWithShortTimeOut = `
apiVersion: shipwright.io/v1alpha1
kind: Build
spec:
  source:
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
apiVersion: shipwright.io/v1alpha1
kind: Build
spec:
  source:
    url: "https://github.com/shipwright-io/sample-go"
  strategy:
    kind: ClusterBuildStrategy
  output:
    image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app
    credentials:
      name: foobarsecret
  timeout: 5s
`

// BuildCBSWithWrongURL defines a Build with a
// ClusterBuildStrategy and a non-existing url
const BuildCBSWithWrongURL = `
apiVersion: shipwright.io/v1alpha1
kind: Build
metadata:
  annotations:
    build.shipwright.io/verify.repository: "true"
spec:
  source:
    url: "https://github.foobar.com/sbose78/taxi"
  strategy:
    kind: ClusterBuildStrategy
  output:
    image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app
`

// BuildCBSWithVerifyRepositoryAnnotation defines a Build
// with the verify repository annotation key
const BuildCBSWithVerifyRepositoryAnnotation = `
apiVersion: shipwright.io/v1alpha1
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
apiVersion: shipwright.io/v1alpha1
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
apiVersion: shipwright.io/v1alpha1
kind: Build
metadata:
  annotations:
    build.shipwright.io/build-run-deletion: "true"
spec:
  source:
    url: "https://github.com/shipwright-io/sample-go"
  strategy:
    kind: ClusterBuildStrategy
  output:
    image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app
`

// BuildWithSleepTimeParam defines a Build with a parameter
const BuildWithSleepTimeParam = `
apiVersion: shipwright.io/v1alpha1
kind: Build
spec:
  source:
    url: "https://github.com/shipwright-io/sample-go"
  paramValues:
  - name: sleep-time
    value: 30
  strategy:
    kind: BuildStrategy
  output:
    image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app
`

// BuildWithRestrictedParam defines a Build using params that are reserved only
// for shipwright
const BuildWithRestrictedParam = `
apiVersion: shipwright.io/v1alpha1
kind: Build
spec:
  source:
    url: "https://github.com/shipwright-io/sample-go"
  paramValues:
  - name: shp-something
    value: 30
  - name: DOCKERFILE
    value: 30
  strategy:
    kind: BuildStrategy
  output:
    image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app
`

// BuildWithUndefinedParameter defines a param that was not declared under the
// strategy parameters
const BuildWithUndefinedParam = `
apiVersion: shipwright.io/v1alpha1
kind: Build
spec:
  source:
    url: "https://github.com/shipwright-io/sample-go"
  paramValues:
  - name: sleep-not
    value: 30
  strategy:
    kind: BuildStrategy
  output:
    image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app
`

// BuildWithEmptyStringParam defines a param that with an empty string value
const BuildWithEmptyStringParam = `
apiVersion: shipwright.io/v1alpha1
kind: Build
spec:
  source:
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
apiVersion: shipwright.io/v1alpha1
kind: Build
spec:
  source:
    url: "https://github.com/shipwright-io/sample-go"
  paramValues:
  - name: sleep-not
    value: 30
  strategy:
    kind: ClusterBuildStrategy
  output:
    image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app
`

// BuildWithSleepTimeParamAndCBS defines a Build with a parameter
const BuildWithSleepTimeParamAndCBS = `
apiVersion: shipwright.io/v1alpha1
kind: Build
spec:
  source:
    url: "https://github.com/shipwright-io/sample-go"
  paramValues:
  - name: sleep-time
    value: 30
  strategy:
    kind: ClusterBuildStrategy
  output:
    image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app
`
