#!/bin/bash

# This task is responsible to configuring and deploying 2 services - monocular and tiller.
# This task runs only master nodes only.

set -e

cd /opt/pf9/pf9-kube/
source utils.sh
source master_utils.sh

[ "$DEBUG" == "true" ] && set -x

function start() {
    if [ "${APP_CATALOG_ENABLED}" == "true" ]; then
        ensure_appcatalog
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
        echo "Deploy app catalog"
        ;;
    "can_run_status")
        echo "no"
        exit 1
        ;;
esac
