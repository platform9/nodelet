#!/bin/bash

# This task configures and starts CoreDNS.
# This task runs on master nodes only.

set -e

cd /opt/pf9/pf9-kube/
source utils.sh
source master_utils.sh

[ "$DEBUG" == "true" ] && set -x

function start() {
    ensure_dns
}

function stop() {
    exit 0
}

function status() {
    #TODO: Add status check for coredns
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
        echo "${COREDNS_CFG}"
        ;;
    "can_run_status")
        echo "no"
        exit 1
        ;;
esac
