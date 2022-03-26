#!/bin/bash

# Default configuration
source defaults.env

# This task uncordons the node. Task should run towards the end of service start.
# On master nodes this task also removes old metrics server and heapster (post_upgrade_cleanup).
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
    # remove KubeStackShutDown annotation (if present) as this is kube stack startup
    remove_annotation_from_node ${node_identifier} KubeStackShutDown

    # check if node cordoned (By User)
    user_node_cordon=$(${KUBECTL} get node/${node_identifier} -o jsonpath='{.metadata.annotations.UserNodeCordon}')

    # If cordoned by user DO NOT uncordon, exit
    if [ $user_node_cordon == "true" ]; then
        exit 0
    fi
    uncordon_node $node_identifier
    prevent_auto_reattach
    if [ "$ROLE" == "worker" ]; then
        exit 0
    fi
    post_upgrade_cleanup
}

function stop() {
    exit 0
}

function status() {
    node_identifier=$NODE_NAME

    if [ $CLOUD_PROVIDER_TYPE == "local" ] && [ "$USE_HOSTNAME" == "true" ]; then
        node_identifier=$HOSTNAME
    fi

    # Check file to see if the node is bootstrapping
    if [ -f $KUBE_STACK_START_FILE_MARKER ]; then
        echo "Kube stack is still booting up, nodes not ready yet"
        exit 0
    fi

    # if KubeStackShutDown is present then node was cordoned by PF9
    kube_stack_shut_down=$(${KUBECTL} get node/${node_identifier} -o jsonpath='{.metadata.annotations.KubeStackShutDown}')

    if [ ! -z $kube_stack_shut_down ] && [ $kube_stack_shut_down == "true" ]; then
        exit 0
    fi

    # if KubeStackShutDown is not present then node was cordoned by the User
    node_cordoned=$(${KUBECTL} describe node ${node_identifier} | grep 'Unschedulable:' | awk '{print $2}')
    if [ "$node_cordoned" == "true" ]; then
            add_annotation_to_node ${node_identifier} UserNodeCordon

    elif [ "$node_cordoned" == "false" ]; then
            remove_annotation_from_node  ${node_identifier} UserNodeCordon
    fi
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
        echo "${UNCRDN_NODE}"
        ;;
    "can_run_status")
        echo "yes"
        exit 0
        ;;
esac
