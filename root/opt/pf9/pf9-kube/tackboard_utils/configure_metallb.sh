#!/usr/bin/env bash
CONTENT_TYPE="content-type: application/json"
TACKBOARD_URL=http://localhost:9080/tackboard/

cd `dirname $0`/..
source /etc/pf9/kube.env
source ./defaults.env
source ./master_utils.sh

function main()
{
    [[ $# -eq 2 ]] || usage
    local uuid="$2"
    export METALLB_CIDR=`echo $1| base64 --decode`
    local output=`configure_metallb`
    echo "$output" \
        | NO_PROXY=localhost curl -s \
                -X POST \
                -H "${CONTENT_TYPE}" \
                -H "uuid: ${uuid}" \
                ${TACKBOARD_URL} \
                -d@-
}

function usage()
{
    echo "Usage: $0 <metallb_cidr_base64> <uuid>"
    exit 1
}

main "$@"