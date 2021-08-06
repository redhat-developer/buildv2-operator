#!/bin/bash

# Copyright The Shipwright Contributors
# 
# SPDX-License-Identifier: Apache-2.0

#
# Installs a private git repo into the cluster (for testing private repo builds)
#

set -eu

DOCKER_PRIVATE_REPO_IMAGE=private-git-repo-test
KIND_CLUSTER_NAME="${KIND_CLUSTER_NAME:-kind}"

tmp_dir=$(mktemp -d -t ssh-XXXXXXXXXX)
trap "rm -rf $tmp_dir" EXIT

ssh-keygen -b 2048 -t rsa -f $tmp_dir/sshkey -q -N ""

echo "# Building a private repo docker image..."

# The Dockerhub repo
# https://hub.docker.com/r/jkarlos/git-server-docker/
cat <<EOF | docker build -t $DOCKER_PRIVATE_REPO_IMAGE -
FROM docker.io/jkarlos/git-server-docker

RUN echo "$(cat ${tmp_dir}/sshkey.pub)" > /git-server/keys/sshkey.pub \
    && chmod 600 /git-server/keys/sshkey.pub

RUN git clone https://github.com/shipwright-io/sample-nodejs \
      /git-server/repos/sample-nodejs.git

WORKDIR /git-server/
EOF

echo "# Loading into kind..."
kind load docker-image $DOCKER_PRIVATE_REPO_IMAGE --name $KIND_CLUSTER_NAME

echo "# Deploying Git Server..."
cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: gitserver
  labels:
    app: gitserver
spec:
  selector:
    matchLabels:
      app: gitserver
  replicas: 1
  template:
    metadata:
      labels:
        app: gitserver
    spec:
      containers:
      - name: gitserver
        image: $DOCKER_PRIVATE_REPO_IMAGE
        imagePullPolicy: IfNotPresent
        ports:
        - name: ssh
          containerPort: 22
---
apiVersion: v1
kind: Service
metadata:
  name: gitserver
spec:
  ports:
  - name: ssh
    port: 22
    targetPort: 22
  selector:
    app: gitserver
---
apiVersion: v1
kind: Secret
metadata:
  name: ${DOCKER_PRIVATE_REPO_IMAGE}-secret
type: kubernetes.io/ssh-auth
data:
  ssh-privatekey: "$(cat ${tmp_dir}/sshkey | base64 | sed 's/$/\\n/' | tr -d '\n')"
EOF

kubectl rollout restart deployment gitserver

# The GIT_SSH_COMMAND magic
# https://stackoverflow.com/a/29754018

cat <<EOF
# To clone from this repo, run
#
#   > kubectl get secrets private-git-repo-test-secret -o json | jq -r '.data[]' | base64 -d > sshkey
#   > chmod 600 sshkey
#   > kubectl port-forward svc/gitserver 2222:22
#   > GIT_SSH_COMMAND='ssh -i sshkey -o IdentitiesOnly=yes -o StrictHostKeyChecking=no' \
#         git clone ssh://git@localhost:2222/git-server/repos/sample-private-repo.git
#
EOF
