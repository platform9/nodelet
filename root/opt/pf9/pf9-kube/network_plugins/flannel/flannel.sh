
#### Required interface function definitions
source defaults.env
source runtime.sh

function network_running()
{
    container_running $socket flannel
}

function ensure_network_running()
{
    # Bridge for containers is created by CNI. So if docker has created a
    # bridge, in the past, delete it
    # See https://platform9.atlassian.net/browse/IAAS-7740 for more information
    if [ "$PF9_MANAGED_DOCKER" != "false" ]; then
        delete_docker0_bridge_if_present
    fi

    if [ "$ROLE" == "master" ]; then
        # When running as a master, ensure flannel talks to etcd on localhost
        ensure_flannel_running_and_runtime_started_with_correct_bridge_subnet "127.0.0.1"
    else
        # FIXME (IAAS-6867): On a worker node in a public cloud cluster, flannel
        # connects to etcd via MASTER_IP / fqdn which is public and load-balanced.
        # Consider using a local etcd 'proxy' instead.
        ensure_flannel_running_and_runtime_started_with_correct_bridge_subnet "$MASTER_IP"
    fi
}

function ensure_network_config_up_to_date()
{
    ensure_containers_CIDR_up_to_date
}

function write_cni_config_file()
{
    write_flannel_cni_config_file
}

function ensure_network_controller_destroyed()
{
    ensure_container_destroyed $socket flannel
    clean_cni_bridge
    remove_cni_config_file
}

#### Plugin specific methods

function ensure_flannel_running_and_runtime_started_with_correct_bridge_subnet()
{
    local etcd_host=$1

    local iface_option=""
    if [ -n "$FLANNEL_IFACE_LABEL" ]; then
        iface_option="--iface=${FLANNEL_IFACE_LABEL}"
    fi

    local public_ip_option=""
    if [ -n "$FLANNEL_PUBLIC_IFACE_LABEL" ]; then
        public_ip_option="--public-ip=$(ipv4_address_of_interface_label $FLANNEL_PUBLIC_IFACE_LABEL)"
    fi

    mkdir -p /run/flannel

    local run_opts="--detach=true \
        --net=host \
        --privileged \
        --volume /dev/net:/dev/net \
        --volume /run/flannel:/run/flannel \
        --volume /etc/pf9/kube.d/certs/flannel/etcd:/etcd-auth"

    local container_name="flannel"
    local quay_registry="${QUAY_PRIVATE_REGISTRY:-quay.io}"
    local container_img="${quay_registry}/coreos/flannel:${FLANNEL_VERSION}"
    # Container image defines the /opt/bin/flanneld entrypoint in 0.10.0
    # Hence setting container_cmd to ""
    local container_cmd=""
    local container_cmd_args="--etcd-endpoints=https://${etcd_host}:4001 \
        --etcd-certfile=/etcd-auth/request.crt \
        --etcd-keyfile=/etcd-auth/request.key \
        --etcd-cafile=/etcd-auth/ca.crt \
        ${iface_option} \
        ${public_ip_option} \
        -v=1"
    ensure_fresh_container_running $socket "${run_opts}" "${container_name}" "${container_img}" "${container_cmd}" "${container_cmd_args}"
}

function ensure_containers_CIDR_up_to_date()
{
    # FIXME: Flannel does not support etcd v3. Hence we need to use etcdctl in v2 mode
    # to create keys for flannel. In future, when etcd v3 support is added to flannel,
    # make sure to:
    # a) remove ETCD_ENABLE_V2 env variable from etcd container
    # b) use etcdctl_tls_flags here instead of etcdctlv2_tls_flags
    # c) set is replace with put in etcd v3 CLI calls.
    update_cidr=true
    local gcr_registry="${GCR_PRIVATE_REGISTRY:-gcr.io}"
    ETCD_CONTAINER_IMG=`echo "${ETCD_CONTAINER_IMG}" | sed "s|gcr.io|${gcr_registry}|g"`

    if json=`pf9ctr_run \
        run ${etcdctl_volume_flags} \
            -e ETCDCTL_API=2 \
            --rm --net=host ${ETCD_CONTAINER_IMG} \
            etcdctl --endpoints 'https://localhost:4001' ${etcdctlv2_tls_flags} get /coreos.com/network/config`; then
                cidr=`echo $json | /opt/pf9/pf9-kube/bin/jq -r '.Network'`
        if [ "$cidr" == "$CONTAINERS_CIDR" ]; then
            echo CIDR is up to date
            update_cidr=false
        fi
    fi
    if [ "$update_cidr" == "true" ]; then
        echo Updating CIDR from $cidr to $CONTAINERS_CIDR
        json="{ \"Network\": \"$CONTAINERS_CIDR\" }"
        pf9ctr_run \
            run ${etcdctl_volume_flags} \
                -e ETCDCTL_API=2 \
                --rm --net=host ${ETCD_CONTAINER_IMG} \
                etcdctl --endpoints 'https://localhost:4001' ${etcdctlv2_tls_flags} set /coreos.com/network/config "$json"
    fi
}

function write_flannel_cni_config_file()
{

# Setting hairpin mode for bridge drive to true
# See https://github.com/kubernetes-incubator/kubespray/pull/1992
# for more information

# Flannel 0.10.0 onwards, flannel takes in conflist (with array of plugins)

cat <<EOF > $CNI_CONFIG_DIR/10-flannel.conflist
{
    "name": "containernet",
    "cniVersion": "0.4.0",
    "plugins": [
      {
        "type": "flannel",
        "delegate": {
          "bridge": "${CNI_BRIDGE}",
          "isDefaultGateway": true,
          "hairpinMode": true
        }
      },
      {
        "type": "portmap",
        "capabilities": {
          "portMappings": true
        }
      }
    ]
}
EOF

}

function clean_cni_bridge()
{
    # Delete if present, else ignore
    echo "Deleting ${CNI_BRIDGE}"
    ip link set dev "${CNI_BRIDGE}" down || echo "Failed to bring down cni bridge. Continuing"
    ip link delete "${CNI_BRIDGE}" type bridge || echo "Failed to delete cni bridge. Continuing"
    rm -rf /var/lib/cni || echo "Failed to delete flannel state files. Continuing"
}

function remove_cni_config_file()
{
    # FIXME Remove 10-flannel.conf rm -f command as this will
    # not be present post 3.6 onwards
    rm -f ${CNI_CONFIG_DIR}/10-flannel.conf || echo "Either file not present or unable to delete. Continuing"
    rm -f ${CNI_CONFIG_DIR}/10-flannel.conflist || echo "Either file not present or unable to delete. Continuing"
    rm -rf /var/run/flannel || echo "Either folder not present or unable to delete. Continuing"
}

function delete_docker0_bridge_if_present()
{
   # Opportunistically delete docker0 bridge
   ip link set dev docker0 down || true
   ip link del docker0 || true
}
