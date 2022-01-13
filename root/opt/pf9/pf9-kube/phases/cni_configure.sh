#!/bin/bash

# This task is responsible for configuring CNI
# This task runs on master and worker nodes.

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
    write_cni_config_file
    set_iptable_forward_policy_allow
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
        echo "${CNI_CFG}"
        ;;
    "can_run_status")
        echo "no"
        exit 1
        ;;
esac
