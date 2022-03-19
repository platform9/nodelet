#!/bin/bash

# This task labels a node as "master" or "worker". Also taints the master nodes if workloads are not allowed on masters.
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
    label_node $node_identifier $ROLE
    if [ "$ALLOW_WORKLOADS_ON_MASTER" == "false" -a "$ROLE" == "master" ]; then
        taint_node $node_identifier
    fi
}

function stop() {
    exit 0
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
        echo "${NODE_TAINT}"
        ;;
    "can_run_status")
        echo "no"
        exit 1
        ;;
esac
