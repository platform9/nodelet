#!/bin/bash

# This tasks starts and waits for different k8s services to be available.
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

# NOTE (pacharya): source network_plugin.sh inside the start/stop/status functions
# some of the network config may not be available when
# the script is called with name or can_run_status
function start() {
    source network_plugin.sh
    if [ "$ROLE" == "worker" ]; then
        ensure_network_running
        exit 0
    fi
    wait_until local_apiserver_running 5 48
    wait_until kubernetes_api_available 5 48
    wait_until ensure_role_binding 5 12
    ensure_network_running
}

function stop() {
    exit 0
}

function status() {
    source network_plugin.sh
    network_running
    if [ "$ROLE" == "worker" ]; then
        exit 0
    fi
    if kubernetes_api_available; then
        return 0
    else
        return 1
    fi
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
        echo "Wait for k8s services and network to be up"
        ;;
    "can_run_status")
        echo "yes"
        exit 0
        ;;
esac
