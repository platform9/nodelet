#!/usr/bin/env bash
set -u

CONTENT_TYPE="content-type: application/json"
TACKBOARD_URL=http://localhost:9080/tackboard/
DEBUG=${DEBUG:-""}
KUBE_OVERRIDE=/etc/pf9/kube_override.env
#TEST=${TEST:-""}
#PASS="true"

function main()
{
#    if [ -n "$TEST" ]; then
#        tests::run_all
#        exit 0
#    fi
    if [ -f ${KUBE_OVERRIDE} ]; then
        source ${KUBE_OVERRIDE}
    fi

    [[ $# -eq 1 ]] || usage
    local uuid="$1"

    local swapStat="{ \"enabled\": \"false\" }"
    if [  $(swapon --show | wc -l) -gt 0 ]; then
        # When $ALLOW_SWAP is enabled we want to keep the swapStat set to
        # "false", such that the node is not rejected by qbert (which rejects
        # an attachment if it is set to "true".
        if [ "x${ALLOW_SWAP:-}" != "xtrue" ]; then
            swapStat="{ \"enabled\": \"true\" }"
        fi
    fi

    local selnxStat="{ \"enabled\": \"false\" }"
    local selnxBin=$(command -v "selinuxenabled")
    if [ ! -z $selnxBin  ] && [ -x $selnxBin  ]; then
        selinuxenabled
        if [ $? -eq 0 ]; then
            selnxStat="{ \"enabled\": \"true\" }"
        fi
    fi

    local firewallStat="{ \"enabled\": \"false\" }"
    if [ $(ps --no-headers -o comm -C firewalld | wc -l) -gt 0 ]; then
      firewallStat="{ \"enabled\": \"true\" }"
    fi

    local output="{ \"swap\": ${swapStat}, \"selinux\": ${selnxStat}, \"firewall\": ${firewallStat} }"
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
    echo "Usage: $0 <uuid>"
    exit 1
}

main "$@"
