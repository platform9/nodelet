#!/bin/bash

# This task handles starting kubelet with appropriate config.
# This task runs only master and worker nodes

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
    echo "Node endpoint is $NODE_NAME"
    if [ "${NODE_NAME}" == "127.0.0.1" ]; then
        echo "Fetched node endpoint as 127.0.0.1. Node interface might have lost IP address. Failing."
        exit 1
    fi
    ensure_kubelet_running $NODE_NAME
}

function stop() {
    ensure_kubelet_stopped
}

function status() {
    if kubelet_running; then
        exit 0
    fi
    exit 1
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
        echo "${KUBELET_CFG}"
        ;;
    "can_run_status")
        echo "yes"
        exit 0
        ;;
esac
