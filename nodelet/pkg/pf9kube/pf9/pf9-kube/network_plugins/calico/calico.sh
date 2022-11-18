#### Required interface function definitions

function network_running()
{
    # TODO: Check status of the local pod/app
    # See https://platform9.atlassian.net/browse/PMK-871
    # Work-around: always return desired state until we have a better algorithm.
    # When ROLE==none, report non-running status to make status_none.sh happy.
    if [ "$ROLE" == "none" ]; then
        return 1
    fi
    return 0
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
        deploy_calico_daemonset
    fi
}

function write_cni_config_file()
{
    return 0
}

function ensure_network_config_up_to_date()
{
    return 0
}


function ensure_network_controller_destroyed()
{
    remove_cni_config_file
    remove_ipip_tunnel_iface
}

#### Plugin specific methods
function deploy_calico_daemonset()
{
    local calico_app="${CONF_SRC_DIR}/networkapps/calico-${KUBERNETES_VERSION}.yaml"
    local calico_app_configured="${CONF_SRC_DIR}/networkapps/calico-${KUBERNETES_VERSION}-configured.yaml"
    local mtu_size=$((${MTU_SIZE}))

    local CALICO_IPV4POOL_CIDR=${CONTAINERS_CIDR}

    # Replace configuration values in calico spec with user input
    sed -e "s|__CALICO_IPV4POOL_CIDR__|${CALICO_IPV4POOL_CIDR}|g" \
        -e "s|__PF9_ETCD_ENDPOINTS__|https://${MASTER_IP}:4001|g" \
        -e "s|__MTU_SIZE__|${mtu_size}|g" \
        -e "s|__CALICO_IPV4_BLOCK_SIZE__|${CALICO_IPV4_BLOCK_SIZE}|g" \
        -e "s|__CALICO_IPIP_MODE__|${CALICO_IPIP_MODE}|g" \
        -e "s|__CALICO_NAT_OUTGOING__|${CALICO_NAT_OUTGOING}|g" \
        -e "s|__CALICO_IPV4__|${CALICO_IPV4}|g" \
        -e "s|__CALICO_IPV6__|${CALICO_IPV6}|g" \
        -e "s|__CALICO_IPV4_DETECTION_METHOD__|${CALICO_IPV4_DETECTION_METHOD}|g" \
        -e "s|__CALICO_IPV6_DETECTION_METHOD__|${CALICO_IPV6_DETECTION_METHOD}|g" \
        -e "s|__CALICO_ROUTER_ID__|${CALICO_ROUTER_ID}|g" \
        -e "s|__CALICO_IPV6POOL_CIDR__|${CALICO_IPV6POOL_CIDR}|g" \
        -e "s|__CALICO_IPV6POOL_BLOCK_SIZE__|${CALICO_IPV6POOL_BLOCK_SIZE}|g" \
        -e "s|__CALICO_IPV6POOL_NAT_OUTGOING__|${CALICO_IPV6POOL_NAT_OUTGOING}|g" \
        -e "s|__FELIX_IPV6SUPPORT__|${FELIX_IPV6SUPPORT}|g" \
        -e "s|__IPV6_ENABLED__|${IPV6_ENABLED}|g" \
        -e "s|__IPV4_ENABLED__|${IPV4_ENABLED}|g" \
        < ${calico_app} > ${calico_app_configured}
    # Apply daemon set yaml
    ${KUBECTL_SYSTEM} apply -f ${calico_app_configured}
}

function remove_cni_config_file()
{
    rm -f ${CNI_CONFIG_DIR}/10-calico* || echo "Either file not present or unable to delete. Continuing"
}

function delete_docker0_bridge_if_present()
{
   # Opportunistically delete docker0 bridge
   ip link set dev docker0 down || true
   ip link del docker0 || true
}

function remove_ipip_tunnel_iface()
{
   ip link set dev tunl0 down || true
   ip link del tunl0 || true
}
