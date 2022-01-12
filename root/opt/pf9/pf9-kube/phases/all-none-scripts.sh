#!/bin/bash
set -e
[ "$DEBUG" == "true" ] && set -x

cd /opt/pf9/pf9-kube/
source utils.sh
source master_utils.sh
source masterless_utils.sh
source defaults.env
source runtime.sh

function start() {
    exit 0
}

function stop() {
    source network_plugin.sh
    ensure_http_proxy_configured

    if kubernetes_api_available; then
        # may be permanently leaving active cluster, perform best-effort node drain and delete
        if NODE_NAME=$(get_node_name); then
            drain_node_from_apiserver $NODE_NAME || true
            delete_node_from_apiserver $NODE_NAME || true
        fi
    fi

    ensure_kubelet_stopped
    teardown_masterless_worker_if_necessary
    ensure_authn_webhook_stopped
    ensure_network_controller_destroyed
    ensure_etcd_destroyed
    destroy_all_k8s_containers
    if [ "$PF9_MANAGED_DOCKER" != "false" ]; then
        ensure_runtime_stopped
    fi

    if [ "${MASTER_VIP_ENABLED}" == "true" ]; then
        stop_and_remove_keepalived
    fi

    cleanup_file_system_state
}

function status() {
    source network_plugin.sh
    ensure_http_proxy_configured
    # We compute status differently when ROLE=master|none. When ROLE=none, the
    # desired state of the pf9-kube service is 'stopped.' If any component that can
    # run on the master or worker is up, we report the service to be running,
    # triggering hostagent to stop the pf9-kube service.
    if [ "$PF9_MANAGED_DOCKER" != "false" ]; then
        kubelet_running \
        || container_running $socket proxy \
        || network_running \
        || runtime_running
    else
        kubelet_running \
        || container_running $socket proxy \
        || network_running
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
        echo "No role assigned. (Cleanup scripts only)"
        ;;
    "can_run_status")
        echo "yes"
        ;;
esac
