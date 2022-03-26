#!/bin/bash

# This task starts and makes sure docker is running
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

# This file should have the following values:
# DOCKERHUB_ID="sample_user"
# DOCKERHUB_PASSWORD="password"
# It will be sourced here and the credentials will be used for `docker login`
DOCKERHUB_CREDS_FILE="/tmp/.dockerhub_creds"

function start() {
    if [ "$PF9_MANAGED_DOCKER" != "false" ]; then
        runtime_start
        
        if [ -f ${DOCKERHUB_CREDS_FILE} ]; then
            echo "Found a DockerHub credentials file."
            source ${DOCKERHUB_CREDS_FILE}
        fi
        
        if [ -z "$DOCKERHUB_ID" ] || [ -z "$DOCKERHUB_PASSWORD" ]; then
            echo "DockerHub credentials not available. You may hit the DockerHub image pull rate limit."
        else
            echo "Found DockerHub credentials. Logging in..."
            pf9ctr_run login -u ${DOCKERHUB_ID} -p ${DOCKERHUB_PASSWORD}
        fi
    fi
}

function stop() {
    destroy_all_k8s_containers
    if [ "$PF9_MANAGED_DOCKER" != "false" ]; then
        ensure_runtime_stopped
    fi
}

function status() {
    if [ "$PF9_MANAGED_DOCKER" != "false" ]; then
        runtime_running
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
        echo "${DOCKER_START}"
        ;;
    "can_run_status")
        echo "yes"
        exit 0
        ;;
esac
