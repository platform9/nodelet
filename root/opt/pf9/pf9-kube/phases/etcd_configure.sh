#!/bin/bash

# This task ensures that etcd data is present on host fs. This will be mounted in the etcd container.
# This task only runs on master nodes.

set -e

cd /opt/pf9/pf9-kube/
source utils.sh
source master_utils.sh

[ "$DEBUG" == "true" ] && set -x

function start() {
    ensure_etcd_data_stored_on_host
    # check if etcd backup and raft index check is required
    # Performed once during
    # 1. new cluster
    # 2. cluster upgrade
    ETCD_UPGRADE="$(! is_eligible_for_etcd_backup || echo "true")"

    if [ "${ETCD_UPGRADE}" == "true" ]; then
        echo "etcd to be upgraded. performing etcd data backup"
        ensure_etcd_data_backup
    fi
}

function stop() {
    ensure_etcd_destroyed
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
        echo "${ETCD_CFG}"
        ;;
    "can_run_status")
        echo "no"
        exit 1
        ;;
esac
