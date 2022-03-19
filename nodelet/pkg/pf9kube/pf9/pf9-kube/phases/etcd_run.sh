#!/bin/bash

# This task starts and ensures that etcd container is running. This task only runs for master nodes.

set -e

cd /opt/pf9/pf9-kube/
source utils.sh
source master_utils.sh

[ "$DEBUG" == "true" ] && set -x

function start() {
    # check if etcd backup and raft index check is required
    # Performed once during
    # 1. new cluster
    # 2. cluster upgrade
    ETCD_UPGRADE="$(! is_eligible_for_etcd_backup || echo "true")"
    echo "Node endpoint is $NODE_NAME"
    if [ "${NODE_NAME}" == "127.0.0.1" ]; then
        echo "Fetched node endpoint as 127.0.0.1. Node interface might have lost IP address. Failing."
        exit 1
    fi
    ensure_etcd_running $NODE_NAME

    if [ "${ETCD_UPGRADE}" == "true" ]; then
        echo "etcd upgrade done. performing etcd raft index check"
        # wait for 10 sec 18 times. 180 sec = 3 min
        wait_until ensure_etcd_cluster_status 10 18
    fi
}

function stop() {
    ensure_etcd_destroyed
}

function status() {
    etcd_running
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
        echo "${ETCD_START}"
        ;;
    "can_run_status")
        echo "yes"
        exit 0
        ;;
esac
