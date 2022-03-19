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
        deploy_canal_daemonset
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
function deploy_canal_daemonset()
{
    local canal_app="${CONF_SRC_DIR}/networkapps/canal-${KUBERNETES_VERSION}.yaml"
    local canal_app_configured="${CONF_SRC_DIR}/networkapps/canal-${KUBERNETES_VERSION}-configured.yaml"

    # Replace configuration values in canal spec with user input
    sed -e "s|__CONTAINERS_CIDR__|${CONTAINERS_CIDR}|g" \
        < ${canal_app} > ${canal_app_configured}
    # Apply daemon set yaml
    ${KUBECTL_SYSTEM} apply -f ${canal_app_configured}
}

function remove_cni_config_file()
{
    rm -f ${CNI_CONFIG_DIR}/10-canal* || echo "Either file not present or unable to delete. Continuing"
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
