#!/bin/bash

# This task installs and, configures docker and containerd
# This task runs only master and worker nodes.

set -e

cd /opt/pf9/pf9-kube/
source utils.sh
source defaults.env
source runtime.sh

if [ "$ROLE" == "master" ]; then
    source master_utils.sh
elif [ "$ROLE" == "worker" ]; then
    source worker_utils.sh
fi

[ "$DEBUG" == "true" ] && set -x

function start() {
    if [ "$PF9_MANAGED_DOCKER" != "false" ]; then
        ensure_runtime_installed_and_stopped
        configure_runtime
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
        echo "${DOCKER_CFG}"
        ;;
    "can_run_status")
        echo "no"
        exit 1
        ;;
esac
