#!/bin/bash
# Processes Tag name in order of priority:
# 1. TAG_NAME env
# 2. Latest git tag using git describe --tags HEAD, if available
# 3. Using branch name
if [[ -n "${TAG_NAME}" ]]; then
  echo "TAG_NAME env found, so using that TAG_NAME:${TAG_NAME}"
  TAG=$TAG_NAME
elif [[ $(git describe --tags HEAD | echo $?) -eq 0 ]]; then
  TAG=$(git describe --tags HEAD)
  IFS='-'
  echo "A latest tag found in git:${TAG}"
  read -a strarr <<< "$TAG"
  TAG=${strarr[0]}
  echo "After replacing TAG:${TAG}"
else
  TAG=$(git rev-parse --abbrev-ref HEAD | sed 's/[^a-zA-Z0-9]/-/g')
  echo "Using branch name for tagging:${TAG}"
fi
echo $TAG