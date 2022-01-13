#!/usr/bin/env bash
set -eu

CONTENT_TYPE="content-type: application/json"
TACKBOARD_URL=http://localhost:9080/tackboard/
DEBUG=${DEBUG:-""}

PF9_KUBE=/etc/init.d/pf9-kube

function main()
{

    local uuid=""

    if [ -n "$DEBUG" ]; then
        [[ $# -eq 0 ]] || usage
    else
        [[ $# -eq 1 ]] || usage
        uuid=$1
    fi

    local output="{ \"running\": false }"

    if ${PF9_KUBE} status; then
        output="{ \"running\": true }"
    fi

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