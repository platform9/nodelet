#!/bin/bash

# This task configures and start pf9-sentry service inside platform9-system namespace.
# This task runs on master nodes only.

set -e

cd /opt/pf9/pf9-kube/
source utils.sh
source master_utils.sh

[ "$DEBUG" == "true" ] && set -x

function start() {
    ensure_pf9_sentry
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
        echo "${SENTRY_CFG}"
        ;;
    "can_run_status")
        echo "yes"
        exit 0
        ;;
esac
