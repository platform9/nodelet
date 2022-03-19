#!/bin/bash

# This task creates a ConfigMap, in kube-system namespace, which is the default
# kubelet config for node of type "master" or "worker".
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
    node_identifier=$NODE_NAME
    if [ $CLOUD_PROVIDER_TYPE == "local" ] && [ "$USE_HOSTNAME" == "true" ]; then
        node_identifier=$HOSTNAME
    fi
    echo "Node name is $node_identifier"
    if [ "${node_identifier}" == "127.0.0.1" ]; then
        echo "Fetched node endpoint as 127.0.0.1. Node interface might have lost IP address. Failing."
        exit 1
    fi
    ensure_dynamic_kubelet_default_configmap
    ensure_node_using_dynamic_configmap $node_identifier
    wait_until kubelet_running 5 5
}

function stop() {
    exit 0
}

function status() {
    node_identifier=$NODE_NAME
    if [ $CLOUD_PROVIDER_TYPE == "local" ] && [ "$USE_HOSTNAME" == "true" ]; then
        node_identifier=$HOSTNAME
    fi
    if [ "${node_identifier}" == "127.0.0.1" ]; then
        echo "Fetched node endpoint as 127.0.0.1. Node interface might have lost IP address. Skipping status check."
        exit 0
    fi
    check_node_using_custom_configmap $node_identifier
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
        echo "${DYN_KUBELET_CFG}"
        ;;
    "can_run_status")
        echo "yes"
        exit 0
        ;;
esac
