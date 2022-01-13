#!/usr/bin/env bash
set -eu

cd /opt/pf9/pf9-kube/
source defaults.env
source runtime.sh

CONTENT_TYPE="content-type: application/json"
TACKBOARD_URL=http://localhost:9080/tackboard/
DEBUG=${MIGRATE_DEBUG:-""}

function main()
{

    local uuid=""

    if [ -n "$DEBUG" ]; then
        [[ $# -eq 1 ]] || usage
    else
        [[ $# -eq 2 ]] || usage
        uuid=$2
    fi
    local container=$1

    local completed=false
    if stop_container $container; then
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

function stop_container() {
    local container=$1
    pf9ctr_run stop $container 2> /dev/null && rcStop=$? || rcStop=$?
    pf9ctr_run wait $container 2> /dev/null && rcWait=$? || rcWait=$?

    if [[ $rcStop -eq 0 ]] && [[ $rcWait -eq 0 ]]; then
        return 0
    else
        return 1
    fi
}


function usage()
{
    if [ -n "$DEBUG" ]; then
        echo "Usage: $0 <container_name> <uuid>"
    else
        echo "Usage: $0 <container_name>"
    fi
    exit 1
}

main "$@"