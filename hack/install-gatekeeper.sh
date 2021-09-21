#!/bin/bash

# Copyright The Shipwright Contributors
# 
# SPDX-License-Identifier: Apache-2.0

#
# Installs Gatekeeper.
# (Used for example gatekeeper policies)
#
# At the time of writing, installing 3.4 instead of 3.5
# because of a regression in the latest release
# https://github.com/open-policy-agent/gatekeeper/issues/1468

set -eu

GATEKEEPER_VERSION="${GATEKEEPER_VERSION:-3.4}"
GATEKEEPER_HOST="raw.githubusercontent.com"
GATEKEEPER_HOST_PATH="open-policy-agent/gatekeeper/release-${GATEKEEPER_VERSION}/deploy"

echo "# Deploying Gatekeeper '${GATEKEEPER_VERSION}'"

kubectl apply -f "https://${GATEKEEPER_HOST}/${GATEKEEPER_HOST_PATH}/gatekeeper.yaml"
