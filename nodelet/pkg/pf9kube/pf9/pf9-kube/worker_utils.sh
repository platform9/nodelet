#!/usr/bin/env bash

source cert_utils.sh



function populate_cert_command_map_worker()
{
    cert_path_to_params_map["flannel/etcd"]="--cn=flannel --cert_type=client"
     # (not used) cert_bg kubelet    server     "${tmp_dir}/kubelet"
    cert_path_to_params_map["kubelet/apiserver"]="--cn=kubelet --cert_type=client"

    cert_path_to_params_map["kubelet/server"]="--cn=kubelet --cert_type=server --sans=${trimmed_kubelet_sans} "

    cert_path_to_params_map["kube-proxy/apiserver"]="--cn=kube-proxy --cert_type=client"

    cert_path_to_params_map["admin"]="--cn=admin --cert_type=client --org=system:masters "

    cert_path_to_params_map["calico/etcd"]="--cn=calico --cert_type=client"

}

function prepare_certs()
{
    local node_name=$1
    local node_name_type=$2
    local node_ip=$3

    local kubelet_sans="\
        IP:${node_ip}, \
        ${node_name_type}:${node_name}"

    if [ "${DUALSTACK}" == "true" ]; then
        kubelet_sans="${kubelet_sans}, IP:${NODE_IPV6}"
    fi

    local trimmed_kubelet_sans=$(trim_sans "$kubelet_sans")
    ensure_certs_dir_backedup "startup"

    init_pki
    (
        tmp_dir=$(mktemp -d --tmpdir=/tmp authbs-certs.XXXX)

        populate_cert_command_map_worker

        if run_certs_requests; then
            echo 'All Certs generated successfully'
            return;
        else
            echo "Failed to generate certificates even after ${MAX_CERTS_RETRIES} retries"
            return 1
        fi
    )
}

