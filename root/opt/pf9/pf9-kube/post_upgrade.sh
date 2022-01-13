#!/usr/bin/env bash
set -e

cd `dirname $0`

CONFIG_DIR=${CONFIG_DIR:-/etc/pf9}
source $CONFIG_DIR/kube.env

source master_utils.sh

[ "$DEBUG" == "true" ] && set -x

CONTENT_TYPE="content-type: application/json"
TACKBOARD_URL=http://localhost:9080/tackboard/
DEBUG=${DEBUG:-""}

function main()
{
    [[ $# -eq 1 ]] || usage
    local uuid="$1"

    local output="{ \"ok\": \"true\" }"

    ${KUBECTL} delete --ignore-not-found configmap pmk

    if [[ "$ENABLE_CAS" == "true" ]]; then
        delete_cluster_autoscaler_post_upgrade
    fi

    post_upgrade_monitoring_fix

    # Migrate master and worker kubelet configmap as the older one of k8s v1.17
    # has PodPriority in feature gate. This is no longer expected in k8s v1.18 kubelet
    # 
    # We are removing it from the configmaps so that rest of the changes made by
    # end-users are preserved.
    ${KUBECTL} get configmap master-default-kubelet-config -n kube-system -oyaml | \
        grep -v "PodPriority: true" | \
        ${KUBECTL} replace configmap master-default-kubelet-config -n kube-system -f -

    ${KUBECTL} get configmap worker-default-kubelet-config -n kube-system -oyaml | \
        grep -v "PodPriority: true" | \
        ${KUBECTL} replace configmap master-default-kubelet-config -n kube-system -f -


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
    echo "Usage: $0 <uuid>"
    exit 1
}

main "$@"
