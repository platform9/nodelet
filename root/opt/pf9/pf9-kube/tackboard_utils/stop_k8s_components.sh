#!/usr/bin/env bash
set -eu

CONTENT_TYPE="content-type: application/json"
TACKBOARD_URL=http://localhost:9080/tackboard/
DEBUG=${MIGRATE_DEBUG:-""}


MASTER_YAML=/etc/pf9/kube.d/master.yaml

function main()
{

    local uuid=""

    if [ -n "$DEBUG" ]; then
        [[ $# -eq 0 ]] || usage
    else
        [[ $# -eq 1 ]] || usage
        uuid=$1
    fi

    local completed=false
    if stop_k8s_components; then
        completed=true
    fi

    local output="{ \"completed\": $completed }"

    if [ -n "$DEBUG" ]; then
        echo "$output"
    else
        echo "$output" \
            | NO_PROXY=localhost curl -s \
                    -X POST \
                    -H "${CONTENT_TYPE}" \
                    -H "uuid: ${uuid}" \
                    ${TACKBOARD_URL} \
                    -d@-
    fi
}

function stop_k8s_components() {

    # Stop the master kubernetes components by removing the manifest from the static-pod-manifest location.
    rm ${MASTER_YAML} && rc=$? || rc=$?
    return $rc
}


function usage()
{
    if [ -n "$DEBUG" ]; then
        echo "Usage: $0 <uuid>"
    else
        echo "Usage: $0"
    fi
    exit 1
}

main "$@"