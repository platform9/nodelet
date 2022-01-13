CGO_ENABLED=0
GOOS=linux
GOARCH=amd64
COMPILE_IMAGE=golang:1.13.6-alpine3.11
REPO_URL=github.com
ORG=platform9
REPO=pf9-addons
COMPONENTS= pf9-sentry
DOCKER=docker
DOCKER_PULL_CMD=${DOCKER} pull
DOCKER_PUSH_CMD=${DOCKER} push
DOCKER_RUN_CMD=${DOCKER} run --rm
DOCKER_BUILD_CMD=${DOCKER} build
DOCKER_RMI_CMD=${DOCKER} rmi
RM=rm
REGISTRY=docker.io
IMAGE_PROJECT=platform9
RELEASE=1.0.0


.PHONY: all release clean
default: compile

all: compile test build

# Compile and generate static binaries
compile:
	for component in ${COMPONENTS}; do \
		echo "Compiling and building $${component} static binary"; \
		${DOCKER_RUN_CMD} -e CGO_ENABLED=${CGO_ENABLED} \
			-e GO111MODULE="on" \
			-e GOOS=${GOOS} \
			-e GOARCH=${GOARCH} \
			-v ${PWD}:/go/src/${REPO_URL}/${ORG}/${REPO}:Z \
			-v ${PWD}/bin:/go/bin:Z \
			-w /go/src/${REPO_URL}/${ORG}/${REPO} \
			${COMPILE_IMAGE} \
			sh -c \
            "go mod download && \
            go build -i -v -a -ldflags \"-extldflags \"-static\"\" -gcflags=\"all=-N -l\" \
			-o /go/bin/$${component} \
			${REPO_URL}/${ORG}/${REPO}/cmd/$${component}"; \
	done

build: test compile
	for component in ${COMPONENTS}; do \
		echo "Building docker image for $${component}"; \
		${DOCKER_BUILD_CMD} --network host -t ${REGISTRY}/${IMAGE_PROJECT}/$${component}:${RELEASE} \
			-f ${PWD}/tooling/$${component}.df \
			${PWD}; \
	done

test:
	echo "Running unit tests";
	${DOCKER_RUN_CMD} -e CGO_ENABLED=${CGO_ENABLED} \
		-e GO111MODULE="on" \
		-e GOOS=${GOOS} \
		-e GOARCH=${GOARCH} \
		-v ${PWD}:/go/src/${REPO_URL}/${ORG}/${REPO}:Z \
		-v ${PWD}/bin:/go/bin:Z \
		${COMPILE_IMAGE} \
		sh -c 'cd /go/src/${REPO_URL}/${ORG}/${REPO} && go mod tidy  && go test -v -cover -covermode count --coverprofile=cov.out \
		${REPO_URL}/${ORG}/${REPO}/...';


release: build
	for component in ${COMPONENTS}; do \
		echo "Pushing docker image for $${component} to registy"; \
		${DOCKER_PUSH_CMD} ${REGISTRY}/${IMAGE_PROJECT}/$${component}:${RELEASE}; \
	done

# Cleanup build artifacts
clean:
	- for component in ${COMPONENTS}; do \
		  ${DOCKER_RUN_CMD} \
			-v ${PWD}/bin:/go/bin:Z \
			${COMPILE_IMAGE} ${RM} -f /go/bin/$${component} && \
			${DOCKER_RMI_CMD} ${REGISTRY}/${IMAGE_PROJECT}/$${component}:${RELEASE}; \
		done
	- rm -rf ${PWD}/bin
