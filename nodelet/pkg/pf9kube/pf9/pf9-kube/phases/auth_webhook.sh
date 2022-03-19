#!/bin/bash

# This task handles setting up of bouncer. Bouncer is responsible for
# handling authentication with keystone.
# This task runs on master nodes only.

set -e

cd /opt/pf9/pf9-kube/
source utils.sh
source master_utils.sh

[ "$DEBUG" == "true" ] && set -x

# NOTE (pacharya): source network_plugin.sh inside the start/stop/status functions
# some of the network config may not be available when
# the script is called with name or can_run_status
function start() {
    source network_plugin.sh
    if [ "$KEYSTONE_ENABLED" == "true" ]; then
        ensure_authn_webhook_image_available
        ensure_authn_webhook_running
        # try reaching webhook to 8 times, waiting 5 seconds in between (each try
        # itself can take up to 5 seconds, so up to 80 seconds can elapse)
        wait_until authn_webhook_listening 5 8
    fi
}

function stop() {
    source network_plugin.sh
    if [ "$KEYSTONE_ENABLED" == "true" ]; then
        ensure_authn_webhook_stopped
    fi
}

function status() {
    source network_plugin.sh
    if [ "$KEYSTONE_ENABLED" == "true" ]; then
        authn_webhook_listening
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
        echo "${AUTH_WEBHOOK}"
        ;;
    "can_run_status")
        echo "yes"
        exit 0
        ;;
esac
