// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package testbeta

// MinimalBuildahBuildRunWithEnvVars defines a simple
// BuildRun with a referenced Build and env vars
const MinimalBuildahBuildRunWithEnvVars = `
apiVersion: shipwright.io/v1beta1
kind: BuildRun
metadata:
  name: buildah-run
spec:
  build:
    name: buildah
  env:
    - name: MY_VAR_2
      value: "my-var-2-buildrun-value"
    - name: MY_VAR_3
      valueFrom:
        fieldRef:
          fieldPath: "my-fieldpath"
`

// BuildahBuildRunWithOutputImageLabelsAndAnnotations defines a BuildRun
// with a output image labels and annotation
const BuildahBuildRunWithOutputImageLabelsAndAnnotations = `
apiVersion: shipwright.io/v1beta1
kind: BuildRun
metadata:
  name: buildah-run
  namespace: build-test
spec:
  build:
    name: buildah
  serviceAccount: buildpacks-v3-serviceaccount
  output:
    image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app-v2
	labels:
	  "maintainer": "new-team@my-company.com"
	  "foo": "bar"
	annotations:
	  "org.opencontainers.owner": "my-company"
`

// MinimalBuildahBuildRun defines a simple
// BuildRun with a referenced Build
const MinimalBuildahBuildRun = `
apiVersion: shipwright.io/v1beta1
kind: BuildRun
metadata:
  name: buildah-run
spec:
  build:
    name: buildah
`

// MinimalBuildahBuildRunWithNodeSelector defines a simple
// BuildRun with a referenced Build and nodeSelector
const MinimalBuildahBuildRunWithNodeSelector = `
apiVersion: shipwright.io/v1beta1
kind: BuildRun
metadata:
  name: buildah-run
spec:
  build:
    name: buildah
  nodeSelector:
    kubernetes.io/arch: amd64
`

// BuildahBuildRunWithSA defines a BuildRun
// with a service-account
const BuildahBuildRunWithSA = `
apiVersion: shipwright.io/v1beta1
kind: BuildRun
metadata:
  name: buildah-run
  namespace: build-test
spec:
  build:
    name: buildah
  serviceAccount: buildpacks-v3-serviceaccount
`

// BuildahBuildRunWithSAAndOutput defines a BuildRun
// with a service-account and output overrides
const BuildahBuildRunWithSAAndOutput = `
apiVersion: shipwright.io/v1beta1
kind: BuildRun
metadata:
  name: buildah-run
  namespace: build-test
spec:
  build:
    name: buildah
  serviceAccount: buildpacks-v3-serviceaccount
  output:
    image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app-v2
`

// BuildpacksBuildRunWithSA defines a BuildRun
// with a service-account
const BuildpacksBuildRunWithSA = `
apiVersion: shipwright.io/v1beta1
kind: BuildRun
metadata:
  name: buildpacks-v3-run
  namespace: build-test
spec:
  build:
    name: buildpacks-v3
  serviceAccount: buildpacks-v3-serviceaccount
`

// BuildahBuildRunWithTimeOutAndSA defines a BuildRun
// with a service-account and timeout
const BuildahBuildRunWithTimeOutAndSA = `
apiVersion: shipwright.io/v1beta1
kind: BuildRun
metadata:
  name: buildah-run
  namespace: build-test
spec:
  build:
    name: buildah
  serviceAccount: buildpacks-v3-serviceaccount
  timeout: 1m
`

// MinimalBuildRun defines a minimal BuildRun
// with a reference to a not existing Build
const MinimalBuildRun = `
apiVersion: shipwright.io/v1beta1
kind: BuildRun
spec:
  build:
    name: foobar
`

const MinimalOneOffBuildRun = `
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
          url: "https://github.com/shipwright-io/sample-go"
        contextDir: docker-build
      strategy:
        kind: ClusterBuildStrategy
        name: buildah
      output:
        image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app
`

// MinimalBuildRunWithParams defines a param override
const MinimalBuildRunWithParams = `
apiVersion: shipwright.io/v1beta1
kind: BuildRun
spec:
  paramValues:
  - name: sleep-time
    value: "15"
  build:
    name: foobar
`

const MinimalBuildRunWithReservedParams = `
apiVersion: shipwright.io/v1beta1
kind: BuildRun
spec:
  paramValues:
  - name: shp-sleep-time
    value: "15"
  build:
    name: foobar
`

// MinimalBuildRunWithSpecifiedServiceAccount defines a minimal BuildRun
// with a reference to a not existing serviceAccount
const MinimalBuildRunWithSpecifiedServiceAccount = `
apiVersion: shipwright.io/v1beta1
kind: BuildRun
spec:
  build:
    name: buildah
  serviceAccount: foobar
`

// MinimalBuildRunWithSAGeneration defines a minimal BuildRun
// with a reference to a not existing Build
const MinimalBuildRunWithSAGeneration = `
apiVersion: shipwright.io/v1beta1
kind: BuildRun
spec:
  serviceAccount: ".generate"
  build:
    name: foobar
`

// MinimalBuildRunWithTimeOut defines a BuildRun with
// an override for the Build Timeout
const MinimalBuildRunWithTimeOut = `
apiVersion: shipwright.io/v1beta1
kind: BuildRun
spec:
  timeout: 1s
  build:
    name: foobar
`

// MinimalBuildRunWithOutput defines a BuildRun with
// an override for the Build Output
const MinimalBuildRunWithOutput = `
apiVersion: shipwright.io/v1beta1
kind: BuildRun
spec:
  output:
    image: foobar.registry.com
  build:
    name: foobar
`

// MinimalBuildRunWithNodeSelector defines a minimal BuildRun
// with a reference to a not existing Build,
// and a nodeSelector
const MinimalBuildRunWithNodeSelector = `
apiVersion: shipwright.io/v1beta1
kind: BuildRun
spec:
  build:
    name: foobar
  nodeSelector:
    kubernetes.io/arch: amd64
`

// MinimalBuildRunWithToleration defines a minimal BuildRun
// with a reference to a not existing Build,
// and a Toleration specified
const MinimalBuildRunWithToleration = `
apiVersion: shipwright.io/v1beta1
kind: BuildRun
spec:
  build:
    name: foobar
  tolerations:
    - key: "buildrun-test-key"
      operator: "Equal"
      value: "buildrun-test-value"
`

// MinimalBuildRunWithVulnerabilityScan defines a BuildRun with
// an override for the Build Output
const MinimalBuildRunWithVulnerabilityScan = `
apiVersion: shipwright.io/v1beta1
kind: BuildRun
spec:
  output:
    image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app
    vulnerabilityScan:
      enabled: true
      failOnFinding: true
  build:
    name: foobar
`

// MinimalBuildRunWithVulnerabilityScan defines a BuildRun with
// an override for the Build Output
const OneOffBuildRunWithVulnerabilityScan = `
apiVersion: shipwright.io/v1beta1
kind: BuildRun
spec:
  build:
    spec:
      strategy:
        kind: ClusterBuildStrategy
        name: buildah
      output:
        image: image-registry.openshift-image-registry.svc:5000/example/buildpacks-app
        vulnerabilityScan:
          enabled: true
          failOnFinding: true
`

// MinimalBuildRunRetention defines a minimal BuildRun
// with a reference used to test retention fields
const MinimalBuildRunRetention = `
apiVersion: shipwright.io/v1beta1
kind: BuildRun
metadata:
  name: buidrun-retention-ttl
spec:
  build:
    name: build-retention-ttl
`

// MinimalBuildRunRetention defines a minimal BuildRun
// with a reference used to test retention fields
const MinimalBuildRunRetentionTTLFive = `
apiVersion: shipwright.io/v1beta1
kind: BuildRun
metadata:
  name: buidrun-retention-ttl
spec:
  build:
    name: build-retention-ttl
  retention:
    ttlAfterFailed: 5s
    ttlAfterSucceeded: 5s
`

const MinimalBuildahBuildRunWithExitCode = `
apiVersion: shipwright.io/v1beta1
kind: BuildRun
metadata:
  name: buildah-run
spec:
  paramValues:
  - name: exit-command
    value: "true"
  build:
    name: buildah
`
