#!/usr/bin/env bash

set -eo pipefail

PF9_KUBE_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PF9_QBERT_DIR=${PF9_KUBE_DIR}/../pf9-qbert


function main()
{
    # If there exists a qbert branch with same name as teamcity build branch
    # use that to run unit tests
    if stat "${PF9_QBERT_DIR}" ; then
      pushd "${PF9_QBERT_DIR}"
      qbertBranch=$(git rev-parse --abbrev-ref HEAD | tr -d '\n')
      if [[ "$qbertBranch" == "$TEAMCITY_BUILD_BRANCH" ]]; then
          echo "Using checked out teamcity branch ${TEAMCITY_BUILD_BRANCH} from pf9-qbert for running unit tests"
      else
          echo "Using ${AMI_BRANCH} from pf9-qbert for running unit tests"
          git checkout ${AMI_BRANCH}
      fi
      popd
    fi

    make nodelet-test
    make agent-tests

    if [[ -z $PROMOTE_BUILD_NUMBER ]]; then
        make agent-wrapper
        make upload-host-packages
    else
        echo "Not building pf9-kube as we are promoting already published pf9-kube package."
    fi
}

# Clean everything
make cache-clean

main "@"
