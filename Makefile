SHELL := /bin/bash

# output directory, where all artifacts will be created and managed
OUTPUT_DIR ?= build/_output
# relative path to controller binary
CONTROLLER = $(OUTPUT_DIR)/bin/shipwright-build-controller

# golang cache directory path
GOCACHE ?= $(shell echo ${PWD})/$(OUTPUT_DIR)/gocache

# golang target architecture
# Check if GO_OS is defined, if not set it to `linux` which is the only supported OS
# this provides flexibility to users to set GO_OS in the future if require
ifeq ($(origin GO_OS), undefined)
GO_OS = "linux"
endif
GO_ARCH ?= $(shell uname -m | sed -e 's/x86_64/amd64/' -e 's/aarch64/arm64/')

# golang global flags
GO_FLAGS ?= -v -mod=vendor -ldflags=-w

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# configure zap based logr
ZAP_FLAGS ?= --zap-log-level=debug --zap-encoder=console

# test namespace name
TEST_NAMESPACE ?= default

# CI: tekton pipelines controller version
TEKTON_VERSION ?= v0.25.0

# E2E test flags
TEST_E2E_FLAGS ?= -failFast -p -randomizeAllSpecs -slowSpecThreshold=300 -timeout=30m -progress -stream -trace -v

# E2E test service account name to be used for the build runs, can be set to generated to use the generated service account feature
TEST_E2E_SERVICEACCOUNT_NAME ?= pipeline

# E2E test verify Tekton objects
TEST_E2E_VERIFY_TEKTONOBJECTS ?= true

# E2E test timeout multiplier
TEST_E2E_TIMEOUT_MULTIPLIER ?= 1

# test repository to store images build during end-to-end tests
TEST_IMAGE_REPO ?= quay.io/shipwright-io/build-e2e
# test container registyr secret name
TEST_IMAGE_REPO_SECRET ?=
# test container registry secret, must be defined during runtime
TEST_IMAGE_REPO_DOCKERCONFIGJSON ?=

# enable private git repository tests
TEST_PRIVATE_REPO ?= false
# github private repository url
TEST_PRIVATE_GITHUB ?=
# gitlab private repository url
TEST_PRIVATE_GITLAB ?=
# private repository authentication secret
TEST_SOURCE_SECRET ?=

# Image settings for building and pushing images
IMAGE_HOST ?= quay.io
IMAGE_NAMESPACE ?= shipwright
TAG ?= latest

# options for generating crds with controller-gen
CONTROLLER_GEN="${GOBIN}/controller-gen"
CRD_OPTIONS ?= "crd:trivialVersions=true,preserveUnknownFields=false"

# The CONTAINER_ENGINE variable is used for specifying the container engine. By default 'docker' is used
# # but this can be overridden when calling make, e.g.
# # CONTAINER_ENGINE=podman make container-image
CONTAINER_ENGINE ?= docker
IMAGE_REGISTRY ?= quay.io/shipwright-io
SHIPWRIGHT_VERSION = 0.6.0
IMAGE_VERSION=$(shell scripts/hash-files.sh Dockerfile Makefile | cut -c 1-12)
CONTAINER_IMAGE   = $(IMAGE_REGISTRY)/shipwright-builder-image:v$(SHIPWRIGHT_VERSION)-$(IMAGE_VERSION)
CONTAINER_RUN     = $(CONTAINER_ENGINE) run --rm --interactive --tty --volume $(CURDIR):/src

.EXPORT_ALL_VARIABLES:

default: build

container-image: ## Build a container image for the preview of the website
	$(CONTAINER_ENGINE) build . --tag $(CONTAINER_IMAGE)

container-make: container-image ## Run make using the development container. A target can be passed using the TARGET=my-target argument. Example: `make container-make TARGET=ginkgo` Run `make container-image` before this.
	$(CONTAINER_RUN) --cap-drop=ALL --cap-add=AUDIT_WRITE --mount type=tmpfs,destination=/tmp,tmpfs-mode=01777 $(CONTAINER_IMAGE) make $(TARGET)


.PHONY: vendor
vendor: go.mod go.sum
	go mod vendor

.PHONY: build
build: $(CONTROLLER)

$(CONTROLLER): vendor
	go build -trimpath $(GO_FLAGS) -o $(CONTROLLER) cmd/shipwright-build-controller/main.go

.PHONY: build-plain
build-plain: 
	go build -trimpath $(GO_FLAGS) -o $(CONTROLLER) cmd/shipwright-build-controller/main.go

.PHONY: build-image
build-image:
	KO_DOCKER_REPO="$(IMAGE_HOST)/$(IMAGE_NAMESPACE)" GOFLAGS="$(GO_FLAGS)" ko publish --base-import-paths ./cmd/shipwright-build-controller

.PHONY: build-image-with-pprof
build-image-with-pprof:
	KO_DOCKER_REPO="$(IMAGE_HOST)/$(IMAGE_NAMESPACE)" GOFLAGS="$(GO_FLAGS) -tags=pprof_enabled" ko publish --base-import-paths --tags=pprof ./cmd/shipwright-build-controller

.PHONY: release
release:
	hack/release.sh

.PHONY: generate
generate:
	hack/update-codegen.sh
	hack/generate-fakes.sh
	hack/generate-copyright.sh

.PHONY: verify-codegen
verify-codegen: generate
	# TODO: Verify vendor tree is accurate
	git diff --quiet -- ':(exclude)go.mod' ':(exclude)go.sum' ':(exclude)vendor/*'

ginkgo:
ifeq (, $(shell which ginkgo))
	@{ \
	set -e ;\
	GINKGO_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$GINKGO_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get -u github.com/onsi/ginkgo/ginkgo ;\
	go get -u github.com/onsi/gomega/... ;\
	rm -rf $$GINKGO_GEN_TMP_DIR ;\
	}
GINKGO=$(GOBIN)/ginkgo
else
GINKGO=$(shell which ginkgo)
endif

gocov:
ifeq (, $(shell which gocov))
	@{ \
	set -e ;\
	GOCOV_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$GOCOV_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get github.com/axw/gocov/gocov@v1.0.0 ;\
	rm -rf $$GOCOV_GEN_TMP_DIR ;\
	}
GOCOV=$(GOBIN)/gocov
else
GOCOV=$(shell which gocov)
endif

install-counterfeiter:
	hack/install-counterfeiter.sh

.PHONY: govet
govet:
	@echo "Checking go vet"
	@go vet ./...

# Install it via: go get -u github.com/gordonklaus/ineffassign
.PHONY: ineffassign
ineffassign:
	@echo "Checking ineffassign"
	@ineffassign ./...

# Install it via: go get -u golang.org/x/lint/golint
# See https://github.com/golang/lint/issues/320 for details regarding the grep
.PHONY: golint
golint:
	@echo "Checking golint"
	@go list ./... | grep -v -e /vendor -e /test | xargs -L1 golint -set_exit_status

# Install it via: go get -u github.com/client9/misspell/cmd/misspell
.PHONY: misspell
misspell:
	@echo "Checking misspell"
	@find . -type f -not -path './vendor/*' -not -path './.git/*' -not -path './build/*' -print0 | xargs -0 misspell -source=text -error

# Install it via: go get -u honnef.co/go/tools/cmd/staticcheck
.PHONY: staticcheck
staticcheck:
	@echo "Checking staticcheck"
	@go list ./... | grep -v /test | xargs staticcheck

.PHONY: sanity-check
sanity-check: ineffassign golint govet misspell staticcheck

# https://github.com/shipwright-io/build/issues/123
test: test-unit

.PHONY: test-unit
test-unit:
	rm -rf build/coverage
	mkdir build/coverage
	go test \
		./cmd/... \
		./pkg/... \
		-coverprofile=unit.coverprofile \
		-outputdir=build/coverage \
		-race \
		-v

test-unit-coverage: test-unit gocov
	echo "Combining coverage profiles"
	cat build/coverage/*.coverprofile | sed -E 's/([0-9])github.com/\1\ngithub.com/g' | sed -E 's/([0-9])mode: atomic/\1/g' > build/coverage/coverprofile
	$(GOCOV) convert build/coverage/coverprofile > build/coverage/coverprofile.json
	$(GOCOV) report build/coverage/coverprofile.json

.PHONY: test-unit-ginkgo
test-unit-ginkgo: ginkgo
	GO111MODULE=on $(GINKGO) \
		-randomizeAllSpecs \
		-randomizeSuites \
		-failOnPending \
		-p \
		-compilers=2 \
		-slowSpecThreshold=240 \
		-race \
		-trace \
		cmd/... \
		pkg/...

# Based on https://github.com/kubernetes/community/blob/master/contributors/devel/sig-testing/integration-tests.md
.PHONY: test-integration
test-integration: install-apis ginkgo
	GO111MODULE=on $(GINKGO) \
		-randomizeAllSpecs \
		-randomizeSuites \
		-failOnPending \
		-slowSpecThreshold=240 \
		-trace \
		test/integration/...


.PHONY: test-e2e
test-e2e: install-strategies test-e2e-plain

.PHONY: test-e2e-plain
test-e2e-plain: ginkgo
	GO111MODULE=on \
	TEST_CONTROLLER_NAMESPACE=${TEST_NAMESPACE} \
	TEST_WATCH_NAMESPACE=${TEST_NAMESPACE} \
	TEST_E2E_SERVICEACCOUNT_NAME=${TEST_E2E_SERVICEACCOUNT_NAME} \
	TEST_E2E_TIMEOUT_MULTIPLIER=${TEST_E2E_TIMEOUT_MULTIPLIER} \
	TEST_E2E_VERIFY_TEKTONOBJECTS=${TEST_E2E_VERIFY_TEKTONOBJECTS} \
	$(GINKGO) ${TEST_E2E_FLAGS} test/e2e

.PHONY: test-e2e-kind-with-prereq-install
test-e2e-kind-with-prereq-install: ginkgo install-controller-kind install-strategies test-e2e-plain

.PHONY: install install-apis install-controller install-strategies

install:
	@echo "Building Shipwright Build controller for platform ${GO_OS}/${GO_ARCH}"
	GOOS=$(GO_OS) GOARCH=$(GO_ARCH) KO_DOCKER_REPO="$(IMAGE_HOST)/$(IMAGE_NAMESPACE)" GOFLAGS="$(GO_FLAGS)" ko apply --base-import-paths -R -f deploy/

install-with-pprof:
	GOOS=$(GO_OS) GOARCH=$(GO_ARCH) GOFLAGS="$(GO_FLAGS) -tags=pprof_enabled" ko apply -R -f deploy/

install-apis:
	kubectl apply -f deploy/crds/
	# Wait for the CRD type to be established; this can take a second or two.
	kubectl wait --timeout=10s --for condition=established crd/clusterbuildstrategies.shipwright.io

install-controller: install-apis
	@echo "Building Shipwright Build controller for platform ${GO_OS}/${GO_ARCH}"
	GOOS=$(GO_OS) GOARCH=$(GO_ARCH) KO_DOCKER_REPO="$(IMAGE_HOST)/$(IMAGE_NAMESPACE)" GOFLAGS="$(GO_FLAGS)" ko apply --base-import-paths -f deploy/

install-controller-kind: install-apis
	KO_DOCKER_REPO=kind.local GOFLAGS="$(GO_FLAGS)" ko apply -f deploy/

install-strategies: install-apis
	kubectl apply -R -f samples/buildstrategy/

local: vendor install-strategies
	CONTROLLER_NAME=shipwright-build-controller \
	go run cmd/shipwright-build-controller/main.go $(ZAP_FLAGS)

local-plain: vendor
	CONTROLLER_NAME=shipwright-build-controller \
	go run cmd/shipwright-build-controller/main.go $(ZAP_FLAGS)

clean:
	rm -rf $(OUTPUT_DIR)

gen-fakes:
	./hack/generate-fakes.sh

kubectl:
	./hack/install-kubectl.sh

kind-registry:
	./hack/install-registry.sh

kind-tekton:
	./hack/install-tekton.sh

kind:
	./hack/install-kind.sh
	./hack/install-registry.sh

generate-crds:
	./hack/install-controller-gen.sh
	"$(CONTROLLER_GEN)" "$(CRD_OPTIONS)" rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=deploy/crds
