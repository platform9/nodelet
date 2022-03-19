#!/bin/bash

# This task configures and starts the kube-proxy container.
# This task runs only master and worker nodes.

set -e

cd /opt/pf9/pf9-kube/
source utils.sh
source runtime.sh

if [ "$ROLE" == "master" ]; then
    source master_utils.sh
elif [ "$ROLE" == "worker" ]; then
    source worker_utils.sh
fi

[ "$DEBUG" == "true" ] && set -x

function start() {
    if [ "${NODE_NAME}" == "127.0.0.1" ]; then
        echo "Fetched node endpoint as 127.0.0.1. Node interface might have lost IP address. Failing."
        exit 1
    fi
    ensure_proxy_running $NODE_NAME
}

function stop() {
    exit 0
}

function status() {
    container_running $socket proxy
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
        echo "${KUBE_PROXY_CFG}"
        ;;
    "can_run_status")
        echo "yes"
        exit 0
        ;;
esac
