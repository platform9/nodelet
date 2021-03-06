#!/bin/bash

set -o nounset
set -o errexit
set -o pipefail
set -x

# This script can be used to generate mocks for different modules. The mock files are generated once and checked in.
# If new mocks are needed, please update this file to simplify developer workflow.
# NOTE: mock files need to be re-generated if new methods are added/removed/modified in the existing mocked modules.

# To install mockgen uncomment the next line
# GO111MODULE=on go get github.com/golang/mock/mockgen@v1.4.3

# Observed that mockgen commands fail sometime after upgrading to golang 1.16+. Remove vendor directory to workaround this issue.

rm -rf mocks/mock_*.go

go generate ./...
