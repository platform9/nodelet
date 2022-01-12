#!/usr/bin/env bash

# test.sh - CI script for running all tests for nodelet.
#
# Parameters:
# - GO_VERSION      Version of Go to use for testing. (default: 1.17.1)

set -o nounset
set -o errexit
set -o pipefail

project_root=$(realpath "$(dirname $0)/..")
build_dir=${project_root}/.ci/build
GO_VERSION=${GO_VERSION:-1.17.1}
export DEBUG=true

set -x

pushd "${project_root}"
trap popd EXIT

# Install gimme
mkdir -p "${build_dir}/bin"
export PATH="${build_dir}/bin:${PATH}"
curl -sL -o "${build_dir}/bin/gimme" https://raw.githubusercontent.com/travis-ci/gimme/master/gimme
chmod +x "${build_dir}/bin/gimme"
gimme --version

# Install go
eval "$(GIMME_GO_VERSION=${GO_VERSION} gimme)"
mkdir -p ${build_dir}/gopath
export GOPATH="${build_dir}/gopath"
export PATH="${GOPATH}/bin:${PATH}"
go version

# Install required commands
GO111MODULE=off go get golang.org/x/tools/cmd/goimports && which goimports

# Run static analysis checks
make nodelet-verify

# Run test suite
make nodelet-test

# Build nodelet and just verify that it is versioned as expected.
make nodelet NODELET_VERSION=v42
build/nodelet/bin/nodeletd version
build/nodelet/bin/nodeletd version | grep 'version: v42' >/dev/null || (echo "error: nodelet did not have expected embedded version\!" && exit 1)

# Run e2e test suite
make nodelet-test-e2e
