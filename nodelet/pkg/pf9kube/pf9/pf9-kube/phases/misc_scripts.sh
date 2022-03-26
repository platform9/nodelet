#!/bin/bash

# This is miscellaneous script. Tasks from start_master, start_worker and
# stop_worker that could not be cleanly bundled with other task ended up in this task :)
# On all nodes, this task is responsible for writing the cloud provider config on the fs.
# On worker nodes, this task also initializes the node to operate in a
# master-less deployment model.
# This task runs only master and worker nodes.

set -e

cd /opt/pf9/pf9-kube/
source utils.sh
if [ "$ROLE" == "master" ]; then
    source master_utils.sh
elif [ "$ROLE" == "worker" ]; then
    source worker_utils.sh
    source masterless_utils.sh
fi

[ "$DEBUG" == "true" ] && set -x

function start() {
    if [ "$ROLE" == "worker" ]; then
        init_masterless_worker_if_necessary
    fi
    write_cloud_provider_config

}

function stop() {
    if [ "$ROLE" == "worker" ]; then
        teardown_masterless_worker_if_necessary
    fi
}

function status() {
    if [ "$ROLE" == "worker" ]; then
        node_identifier=$NODE_NAME
        if [ $CLOUD_PROVIDER_TYPE == "local" ] && [ "$USE_HOSTNAME" == "true" ]; then
            node_identifier=$HOSTNAME
        fi
        if [ "${node_identifier}" == "127.0.0.1" ]; then
            echo "Fetched node endpoint as 127.0.0.1. Node interface might have lost IP address. Skipping status check."
            exit 0
        fi
        if kubernetes_api_available; then
            if ! err=$(node_is_up $node_identifier 2>&1); then
                echo "Warning: node ${node_identifier} is not up: ${err}" >&2
                exit 1
            fi
        fi
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
        echo "Miscellaneous scripts and checks"
        ;;
    "can_run_status")
        echo "yes"
        exit 0
        ;;
esac
