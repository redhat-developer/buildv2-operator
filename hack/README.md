<!--
Copyright The Shipwright Contributors

SPDX-License-Identifier: Apache-2.0
-->
# Assorted scripts for development

This directory contains several scripts useful in the development process of Shipwright Build.

- `build-logs.sh` Collect the log of specific BuildRun Pod in the cluster.
- `generate-copyright.sh` Generate the Shipwright Copyright header for all required files.
- `generate-fakes.sh` Use Go counterfeiter to generate fakes.
- `install-counterfeiter.sh` Install the Go counterfeiter.
- `install-kind.sh` Install the latest verified Kubernetes cluster by KinD.
- `install-kubectl.sh` Install the kubectl command line.
- `install-registry.sh` Install the local container registry in the KinD cluster.
- `install-tekton.sh` Install the latest verified Tekton Pipeline release.
- `install-gatekeeper.sh` Install Gatekeeper release.
- `release.sh` Creates a new release of Shipwright Build.
- `update-codegen.sh` Updates auto-generated client libraries.
- `verify-codegen.sh` Verifies that auto-generated client libraries are up-to-date.
