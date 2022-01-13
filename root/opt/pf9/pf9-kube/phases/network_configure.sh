#!/bin/bash

# This task makes sure that the CIDR configuration for flannel is up-to-date.
# For other network plugins like calico, canal and weave it is a NOOP.
# This task runs on master nodes only.

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
    if [ $ROLE == "master" ]; then
        source network_plugin.sh
        ensure_network_config_up_to_date
    fi
}

function stop() {
    source network_plugin.sh
    ensure_network_controller_destroyed
}

function status() {
    source network_plugin.sh
    network_running
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
        echo "${NW_CFG}"
        ;;
    "can_run_status")
        echo "yes"
        exit 0
        ;;
esac
