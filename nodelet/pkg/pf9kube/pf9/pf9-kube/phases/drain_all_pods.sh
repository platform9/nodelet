#!/bin/bash

# This is a special task that only implements stop function. Whenever the
# pf9-kube service is to be stopped this task will be executed first since it
# has the highest order. This task drains the node before stop function of
# other tasks is invoked.
# This task runs only master and worker nodes.

set -e

cd /opt/pf9/pf9-kube/
source utils.sh
if [ "$ROLE" == "master" ]; then
    source master_utils.sh
elif [ "$ROLE" == "worker" ]; then
    source worker_utils.sh
fi

[ "$DEBUG" == "true" ] && set -x

function start() {

    # Remove file marker for kube stack booting up
    rm $KUBE_STACK_START_FILE_MARKER

    exit 0
}

function stop() {
    ensure_http_proxy_configured
    if kubernetes_api_available; then
        # may be permanently leaving active cluster, perform best-effort node drain and delete
        node_identifier=$NODE_NAME
        if [ $CLOUD_PROVIDER_TYPE == "local" ] && [ "$USE_HOSTNAME" == "true" ]; then
            node_identifier=$HOSTNAME
        fi
        drain_node_from_apiserver $node_identifier || true
    fi
}

function status() {
    exit 0
}

operation=$1

case $operation in
    "status")
        status
        ;;
    "start")
        start
        ;;
    "stop")
        stop
        ;;
    "name")
        echo "Drain all pods (stop only operation)"
        ;;
    "can_run_status")
        echo "no"
        exit 1
        ;;
esac
