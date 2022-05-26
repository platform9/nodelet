CGO_ENABLED=0
GOOS=linux
GOARCH=amd64
GOLANG_DOCKER_IMG=golang:1.16.3-alpine3.13
COMPILE_IMAGE=pf9-compile
REPO_URL=github.com
ORG=platform9
REPO=nodelet
DOCKER=docker
DOCKER_PULL_CMD=${DOCKER} pull
DOCKER_PUSH_CMD=${DOCKER} push
DOCKER_RUN_CMD=${DOCKER} run
DOCKER_BUILD_CMD=${DOCKER} build
DOCKER_RMI_CMD=${DOCKER} rmi
RM=rm
REPEAT_TESTS ?= "1"
GITBRANCH=$(git symbolic-ref --short HEAD)
GITCOMMIT=$(git rev-parse HEAD)

# Using PWD is not guaranteed to be the directory of the Makefile. Use these instead:
MAKE_PATH := $(abspath $(lastword $(MAKEFILE_LIST)))
MAKE_DIR := $(dir $(MAKE_PATH))

# To enable using this Makefile directly, set the AGENT_* variables if none are
# provided:
AGENT_SRC_DIR ?= ..
AGENT_BUILD_DIR ?= build

# Version to embed into Nodelet.
NODELET_VERSION_RELEASE ?= $(shell ../get-version.bash)-$(shell ../get-release.bash)

# Note: we need realpath because these variables are used to identify Docker
# volume paths, which require an absolute path.
NODELET_SRC_DIR := $(shell realpath $(AGENT_SRC_DIR)/nodelet)
NODELET_BUILD_DIR := $(shell realpath -m $(AGENT_BUILD_DIR)/nodelet)
NODELET_BIN_DIR := $(NODELET_BUILD_DIR)/bin
NODELET := $(NODELET_BIN_DIR)/nodeletd

# We need the PROJECT_DIR to get the .git directory which is used in the  versioning.
PROJECT_DIR := $(shell realpath $(NODELET_SRC_DIR)/../..)

# Makefile markers
GOLANG_DOCKER_MARKER := ${NODELET_BUILD_DIR}/${COMPILE_IMAGE}.marker

.PHONY: nodelet-setup nodelet nodelet-test nodelet-clean
.DEFAULT_GOAL:= nodelet

# Download/create build envs
nodelet-setup: ${NODELET_BIN_DIR} ${GOLANG_DOCKER_MARKER}
	echo "setting up env for nodeletd"
	cd ${NODELET_SRC_DIR} && go mod vendor || true

${GOLANG_DOCKER_MARKER}: ${NODELET_BIN_DIR} ${NODELET_SRC_DIR}/tooling/compile/compile.df
	${DOCKER_BUILD_CMD} --network host \
		-f ${NODELET_SRC_DIR}/tooling/compile/compile.df \
		-t ${COMPILE_IMAGE} \
		${NODELET_SRC_DIR};
	touch ${GOLANG_DOCKER_MARKER}

# Note: creating the directory structure should be done before any of the
# docker commands run, to ensure that the directories are not created by the
# Docker daemon with user `root`.
${NODELET_BIN_DIR}:
	mkdir -p ${NODELET_BIN_DIR}

# Compile and generate static binaries
nodelet: nodelet-setup
	echo "make nodelet"
	# Include .git directory to be able to retrieve git-related information for goversion.
	# The way TeamCity worker is set up, on all volumes need the objects need to relabeled (option 'Z').
	# Call to goversion explicitly to verify that is present on the path, to avoid accidentally ignoring it.
	${DOCKER_RUN_CMD} --rm \
			-e CGO_ENABLED=${CGO_ENABLED} \
			-e GO111MODULE="on" \
			-e GOOS=${GOOS} \
			-e GOARCH=${GOARCH} \
			-v ${NODELET_SRC_DIR}:/go/src/${REPO_URL}/${ORG}/${REPO}:Z \
			-v ${PROJECT_DIR}/.git:/go/src/${REPO_URL}/${ORG}/${REPO}/.git:Z \
			-v ${NODELET_BIN_DIR}:/build/bin:Z \
			${COMPILE_IMAGE} \
			sh -xc 'cd /go/src/${REPO_URL}/${ORG}/${REPO} && \
			which goversion > /dev/null && \
			go build -mod=vendor -ldflags "-extldflags \"-static\" $$(goversion ldflags --print-ldflag=false --build-by=docker-golang --version=${NODELET_VERSION_RELEASE} --git-branch=${GITBRANCH} --git-commit=${GITBRANCH})" \
			-o /build/bin/nodeletd \
			./cmd/nodeletd';

nodelet-test: nodelet-setup
	echo "make nodelet-test"
	# The way TeamCity worker is set up, on all volumes need the objects need to relabeled (option 'Z').
	${DOCKER_RUN_CMD} --rm \
		-e CGO_ENABLED=${CGO_ENABLED} \
		-e GO111MODULE="on" \
		-e GOOS=${GOOS} \
		-e GOARCH=${GOARCH} \
		-e REPEAT_TESTS=${REPEAT_TESTS} \
		-v ${NODELET_SRC_DIR}:/go/src/${REPO_URL}/${ORG}/${REPO}:Z \
		--cap-add=SYS_PTRACE \
		--name $@ \
		${COMPILE_IMAGE} \
		sh -xc 'cd /go/src/${REPO_URL}/${ORG}/${REPO} && \
			./run_unit_tests.sh'

# Note: this command expects Go and Docker to be present on the local machine
nodelet-test-e2e: nodelet
	# Build the sunpike-conductor image to use in the test.
	# TODO(erwin) allow overriding of TEST_SUNPIKE_CONDUCTOR_IMAGE to use an existing image instead rebuilding it always.
	$(MAKE) -C $(MAKE_DIR)/../../pf9-qbert/sunpike/conductor docker-build IMAGE_NAME_TAG=sunpike-conductor:e2e
	# Add the build binary to the PATH to make it available to the e2e tests.
	TEST_NODELETD_PATH="$(NODELET)" go test -v ./test

# Cleanup build artifacts
nodelet-clean: nodelet-setup
	echo "make nodelet-clean"
	- ${DOCKER_RUN_CMD} \
		-v ${NODELET_BIN_DIR}:/go/bin:Z \
		${COMPILE_IMAGE} ${RM} -f /go/* && \
		${DOCKER_RMI_CMD} ${COMPILE_IMAGE};
	rm -f **/junit.xml
	rm -f **/coverage.out

# Note: we need to namespace the commands because this makefile is included other Makefiles.
.PHONY: nodelet-generate
nodelet-generate:
	./generate_mocks.sh
	$(MAKE) nodelet-format # Ensure that the generated code is properly formatted

.PHONY: nodelet-verify
nodelet-verify: ## Run all static analysis checks.
	# Check if codebase is formatted.
	@bash -c "[ -z \"$$(goimports -l cmd pkg mocks)\" ] && echo 'OK' || (echo 'ERROR: files are not formatted:' && goimports -l cmd pkg mocks && echo -e \"\nRun 'make nodelet-format' or manually fix the formatting issues.\n\" && false)"
	# Run static checks on codebase.
	go vet ./cmd/... ./pkg/... ./mocks/...

.PHONY: nodelet-format
nodelet-format: ## Run all formatters on the codebase.
	# Format the Go codebase.
	goimports -w cmd pkg mocks

	# Format the go.mod file.
	go mod tidy
