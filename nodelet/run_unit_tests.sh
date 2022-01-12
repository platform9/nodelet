#!/bin/sh

set -o nounset
set -o errexit

export UNIT_TEST=1
if [ ! -z "${REPEAT_TESTS}" ]; then
    for i in `seq 1 ${REPEAT_TESTS}`
    do
        echo "Run # ${i}"
        if ginkgo -mod=vendor -v -cover -coverprofile=coverage.out -outputdir=/go/bin \
            --randomizeSuites --randomizeAllSpecs --trace  --progress \
            ./pkg/... ./cmd/... ; then
            echo "Run ${i} completed successfully"
        else
            echo "Run ${i} failed. Stopping"
            exit 1
        fi
        sleep 5
    done
else
    ginkgo -mod=vendor -v -cover -coverprofile=coverage.out -outputdir=/go/bin \
            --randomizeSuites --randomizeAllSpecs --trace  --progress \
             ./pkg/... ./cmd/...
fi

