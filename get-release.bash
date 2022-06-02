#!/bin/bash
if [[ -z "${BUILD_NUMBER}" ]]; then
    RELEASE=$(git rev-parse --short HEAD)
else
    RELEASE=${BUILD_NUMBER}
fi
echo $RELEASE