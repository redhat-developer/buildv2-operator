#!/bin/bash

# Copyright The Shipwright Contributors
#
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

if ! hash jq >/dev/null 2>&1 ; then
  echo "[ERROR] jq is not installed"
  exit 1
fi

# Delete old job for storage version migration
kubectl -n shipwright-build delete job --selector app=storage-version-migration-shipwright --wait=true

# create new job for storage version migration
cat <<EOF | kubectl create -f -
apiVersion: batch/v1
kind: Job
metadata:
  generateName: storage-version-migration-shipwright-
  labels:
    app: storage-version-migration-shipwright
    app.kubernetes.io/component: storage-version-migration-job
    app.kubernetes.io/name: shipwright-build
  namespace: shipwright-build
spec:
  backoffLimit: 10
  template:
    metadata:
      annotations:
        sidecar.istio.io/inject: "false"
      labels:
        app: storage-version-migration-shipwright
        app.kubernetes.io/component: storage-version-migration-job
        app.kubernetes.io/name: shipwright-build
    spec:
      serviceAccountName: shipwright-build-controller
      containers:
        - args:
            - buildruns.shipwright.io
            - builds.shipwright.io
            - buildstrategies.shipwright.io
            - clusterbuildstrategies.shipwright.io
          image: gcr.io/knative-releases/knative.dev/pkg/apiextensions/storageversion/cmd/migrate
          name: migrate
          resources:
            limits:
              cpu: 1000m
              memory: 1000Mi
            requests:
              cpu: 100m
              memory: 100Mi
          securityContext:
            allowPrivilegeEscalation: false
            capabilities:
              drop:
                - ALL
            readOnlyRootFilesystem: true
            runAsNonRoot: true
            seccompProfile:
              type: RuntimeDefault
      restartPolicy: OnFailure
  ttlSecondsAfterFinished: 600
EOF


NAMESPACE="shipwright-build"
JOB_NAME=$(kubectl -n "${NAMESPACE}" get job --selector app=storage-version-migration-shipwright -o jsonpath='{.items[0].metadata.name}')

while true; do
    jobStatus=$(kubectl get job "${JOB_NAME}" -n "${NAMESPACE}" -o=jsonpath='{.status.conditions[*].type}')

    if [[ "${jobStatus}" == *"Complete"* ]]; then
		echo "Job ${JOB_NAME} has completed successfully!"
		exit 0
    elif [[ "${jobStatus}" == *"Failed"* ]]; then
      	echo "[ERROR] Storage version migration failed"
		exit 1
    fi

    echo "[INFO] Storage version migration job is still running"
    sleep 10
done

echo "[DONE]"