#!/bin/bash

# This task is responsible for customizing the kubeconfigs required for starting the k8s clusters.
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
    local config_dir="/etc/pf9/kube.d/kubeconfigs"
    if [ "$ROLE" == "master" ]; then
        kustomize_config
        prepare_conf_files
        prepare_kubeconfigs_master_only
    fi
    prepare_kubeconfigs
    if [ "$ROLE" == "master" ]; then
        prepare_rolebindings
    fi

    # Make the config directory read write executable
    chmod -R 0600 $config_dir
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
        echo "${PREP_CFG}"
        ;;
    "can_run_status")
        echo "no"
        exit 1
        ;;
esac
