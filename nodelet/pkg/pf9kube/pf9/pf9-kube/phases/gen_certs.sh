#!/bin/bash

# This task performs some of the pre-requisites checks before generating certificates.
# These certificates are used during etcd and k8s cluster creation.
# This task runs only master and worker nodes.

set -e
[ "$DEBUG" == "true" ] && set -x

cd /opt/pf9/pf9-kube/
source utils.sh
if [ "$ROLE" == "master" ]; then
    source master_utils.sh
elif [ "$ROLE" == "worker" ]; then
    source worker_utils.sh
fi

function start() {
    # Pre-req check part
    check_swap_disabled
    check_required_params
    os_specific_config
    set_sysctl_params

    # write node is bootstrapping to a file
    touch $KUBE_STACK_START_FILE_MARKER
    
    # Actual generate certs
    echo node IP is $NODE_IP
    echo node name is "$NODE_NAME"
    echo node endpoint type is "$NODE_NAME_TYPE"
    if [[ "x$NODE_NAME" == "x"  ||  "x$NODE_IP" == "x"  ||  "x$NODE_NAME_TYPE" == "x" || "$NODE_NAME" == "127.0.0.1" ]]; then
        echo "Required node IP or endpoint info not found"
        exit 2
    fi
    if ! [ -e "$CERTS_DIR/.done" ]; then
        prepare_certs $NODE_NAME $NODE_NAME_TYPE $NODE_IP
        # store some information so that we can cross check later
        # TODO: figure out better ways of baking this info in the cert itself.
        echo "CLUSTER_ROLE ${ROLE}" > "$CERTS_DIR/.done"
        echo "CLUSTER_ID ${CLUSTER_ID}" >> "$CERTS_DIR/.done"
    fi
}

function stop() {
    cleanup_dynamic_kubelet_dir="false"
    if ! [ -f $CERTS_DIR/.done ]; then
        cleanup_dynamic_kubelet_dir="true"
    fi
    exit_code=$(grep -q "CLUSTER_ROLE ${ROLE}" $CERTS_DIR/.done; echo $?)
    if ! [ $exit_code -eq 0 ]; then
        cleanup_dynamic_kubelet_dir="true"
    fi
    exit_code=$(grep -q "CLUSTER_ID ${CLUSTER_ID}" $CERTS_DIR/.done; echo $?)
    if ! [ $exit_code -eq 0 ]; then
        cleanup_dynamic_kubelet_dir="true"
    fi
    exit_code=$(grep -q "custom_dynamic_kubeconfig_used" ${KUBELET_DYNAMIC_CONFIG_DIR}/.dynamic_config; echo $?)
    if [ $exit_code -eq 0 ]; then
        cleanup_dynamic_kubelet_dir="false"
    fi
    if [ "$cleanup_dynamic_kubelet_dir" == "true" ]; then
        echo "Noticed a cluster or cluster role change. Cleaning kubelet dynamic config."
        sudo /bin/chown -R pf9:pf9group ${KUBELET_DYNAMIC_CONFIG_DIR}
        rm -rf ${KUBELET_DYNAMIC_CONFIG_DIR}
    else
        echo "No change in cluster or cluster role. Kubelet dynamic config will not be cleaned up."
    fi
    # Tear down for generate cert script
    teardown_certs
    # No tear down for pre req check
}

function status() {

    if ! [ -f $CERTS_DIR/.done ]; then
        exit 1
    fi
    exit_code=$(grep -q "CLUSTER_ROLE ${ROLE}" $CERTS_DIR/.done; echo $?)
    if ! [ $exit_code -eq 0 ]; then
        exit 1
    fi
    exit_code=$(grep -q "CLUSTER_ID ${CLUSTER_ID}" $CERTS_DIR/.done; echo $?)
    if ! [ $exit_code -eq 0 ]; then
        exit 1
    fi
    # kubelet certs get created on both worker and master
    kubelet_cert=$CERTS_DIR/kubelet/server/ca.crt
    if ! [ -f $kubelet_cert ]; then
        echo "$CERTS_DIR/.done file present but kubelet ca cert file missing"
        exit 1
    fi
    # check if cert end date is at least greater than now
    cert_date=`openssl x509 -noout -enddate -in $kubelet_cert | awk -F '=' '{print $2}'`
    epoch_cert_date=`date -d "$cert_date" +%s`
    epoch_curr_date=`date +%s`
    if [ $epoch_cert_date -lt $epoch_curr_date ]; then
        echo "Cert end date is less than current date"
        exit 1
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
        echo "${GEN_CERTS}"
        ;;
    "can_run_status")
        echo "yes"
        exit 0
        ;;
esac
