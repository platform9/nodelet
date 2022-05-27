#!/bin/bash
# Processes Tag name in order of priority:
# 1. TAG_NAME env
# 2. Latest git tag using git describe --tags HEAD, if available
# 3. Using branch name
if [[ -n "${TAG_NAME}" ]]; then
  TAG=$TAG_NAME
elif [[ $(git describe --tags HEAD > /dev/null 2>&1; echo $?) -eq 0 ]]; then
  TAG=$(git describe --tags HEAD)
  IFS='-'
  read -a strarr <<< "$TAG"
  TAG=${strarr[0]}
else
  TAG=$(git rev-parse --abbrev-ref HEAD | sed 's/[^a-zA-Z0-9]/-/g')
fi
echo $TAG