############################
# Bouncer build
BOUNCER_SRC_ROOT := $(ROOT_DIR)/bouncer
BOUNCER_SRC_FILES := $(shell find $(BOUNCER_SRC_ROOT) -name '*.go')
BOUNCER_BUILD_DIR := $(BUILD_DIR)/bouncer
BOUNCER_BIN := $(BOUNCER_BUILD_DIR)/bouncerd
BOUNCER_VERSION := 1.2.0
BOUNCER_IMAGE := platform9/bouncer:$(BOUNCER_VERSION)
BOUNCER_IMAGE_TARBALL := $(BOUNCER_BUILD_DIR)/bouncer-$(BOUNCER_VERSION).tar

mkfile_path := $(abspath $(lastword $(MAKEFILE_LIST)))
mkfile_dir := $(dir $(mkfile_path))

.PHONY: bouncer bouncer-docker-image bouncer-test bouncer-vet bouncer-clean

$(BOUNCER_BUILD_DIR):
	echo "make BOUNCER_BUILD_DIR $(BOUNCER_BUILD_DIR)"
	echo "-> BUILD_DIR $(BUILD_DIR)"
	echo "-> QBERT_SRC_DIR $(QBERT_SRC_DIR)"
	mkdir -p $@

# Notes:
# - Bouncer is built with CGO disabled, resulting in a statically-linked binary.
# There are parts of the stdlib that require CGO, but bouncer does
# not uses them. (see https://www.reddit.com/r/golang/comments/2ut9hw/real_pitfalls_of_crosscompiling_with_cgodisabled/)
# - `go build` must be used instead of `go install`, as the latter will attempt
# to overwrite any go stdlib object files compiled with CGO enabled. (see
# https://github.com/golang/go/issues/18981)
# - `go build` ignores `GOBIN` and writes the `bouncer` binary in the current
# working directory
$(BOUNCER_BIN): $(BOUNCER_SRC_FILES) | $(BOUNCER_BUILD_DIR)
	echo "make BOUNCER_BIN $(BOUNCER_BIN)"
	cd ./bouncer && docker run --rm -v "$(mkfile_dir)":/app -w /app \
		-e GOOS=linux \
	 	-e GOARCH=amd64 \
	 	-e CGO_ENABLED=0 \
 	 	golang:1.13.8 go -- build \
 	 		-v \
 	 		-ldflags="-X github.com/platform9/pf9-qbert/bouncer/pkg/api.Version=${BOUNCER_VERSION}" \
 	 		"./cmd/bouncerd"
	cd ./bouncer && mv bouncerd $(BOUNCER_BUILD_DIR)/bouncerd

$(BOUNCER_IMAGE_TARBALL): $(BOUNCER_SRC_ROOT)/Dockerfile $(BOUNCER_BIN)
	echo "make BOUNCER_IMAGE_TARBALL $(BOUNCER_IMAGE_TARBALL)"
	$(eval $@_TEMPDIR := $(shell mktemp -d))
	cd $($@_TEMPDIR) && \
	cp $(BOUNCER_SRC_ROOT)/Dockerfile . && \
	cp -r $(BOUNCER_SRC_ROOT)/config . && \
	cp $(BOUNCER_BIN) . && \
	docker build --network host -t $(BOUNCER_IMAGE) . && \
	rm -rf $($@_TEMPDIR) && \
	cd $(BOUNCER_BUILD_DIR) && \
	docker save --output $(BOUNCER_IMAGE_TARBALL) $(BOUNCER_IMAGE) && \
	docker rmi -f $(BOUNCER_IMAGE)

bouncer: $(BOUNCER_BIN)
	echo "make bouncer"

bouncer-docker-image: $(BOUNCER_IMAGE_TARBALL)
	echo "make bouncer-docker-image"

bouncer-image-push: $(BOUNCER_IMAGE_TARBALL)
	docker load --input $(BOUNCER_IMAGE_TARBALL) && \
	docker push $(BOUNCER_IMAGE) && \
	docker rmi $(BOUNCER_IMAGE)

bouncer-test: bouncer-vet $(BOUNCER_SRC_FILES)
	echo "make bouncer-test"
	cd ./bouncer && docker run --rm -v "$(mkfile_dir)":/app -w /app golang:1.13.8 go -- test ./...

bouncer-vet: $(BOUNCER_SRC_FILES)
	echo "make bouncer-vet"
	cd ./bouncer && docker run --rm -v "$(mkfile_dir)":/app -w /app golang:1.13.8 go -- vet ./...

bouncer-clean: | $(BOUNCER_BUILD_DIR)
	echo "make bouncer-clean"
	rm -rf $(BOUNCER_BUILD_DIR)
