#!/bin/bash

# This task is responsible for configuring and starting keepalived service.
# This task runs on master nodes only.

set -e

cd /opt/pf9/pf9-kube/
source utils.sh
source master_utils.sh

[ "$DEBUG" == "true" ] && set -x

function start() {
    if [ "${MASTER_VIP_ENABLED}" == "true" ]; then
        ensure_keepalived_installed
        ensure_keepalived_configured
        start_keepalived
    fi
}

function stop() {
    if [ "${MASTER_VIP_ENABLED}" == "true" ]; then
        stop_and_remove_keepalived
    fi
}

function status() {
    if [ "${MASTER_VIP_ENABLED}" == "true" ]; then
        # Check if correct version of keepalived installed
        IS_KEEPALIVED_INSTALLED=-1
        check_keepalived_installed
        if [ $IS_KEEPALIVED_INSTALLED == 0 ]; then
          exit 1
        fi
        # check if keepalived is running
        keepalived_running
    fi
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
        echo "${KEEPALIVED_CFG}"
        ;;
    "can_run_status")
        echo "yes"
        exit 0
        ;;
esac
