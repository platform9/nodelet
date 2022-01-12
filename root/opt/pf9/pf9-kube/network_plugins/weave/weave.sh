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
        deploy_weave_daemonset
    fi
}

function write_cni_config_file()
{
    write_weave_cni_config_file
}

function ensure_network_config_up_to_date()
{
    return 0
}


function ensure_network_controller_destroyed()
{
    remove_cni_config_file
}


#### Plugin specific methods
function deploy_weave_daemonset()
{
    local weave_app="${CONF_SRC_DIR}/networkapps/weave-${KUBERNETES_VERSION}.yaml"
    local weave_app_with_cidr="${CONF_SRC_DIR}/networkapps/weave-${KUBERNETES_VERSION}-configured.yaml"
    # Replace __CONTAINER_CIDR__ with $CONTAINERS_CIDR
    sed -e "s|__CONTAINERS_CIDR__|${CONTAINERS_CIDR}|g" < ${weave_app} > ${weave_app_with_cidr}
    # Apply daemon set yaml
    ${KUBECTL_SYSTEM} apply -f ${weave_app_with_cidr}
}

function write_weave_cni_config_file()
{

cat <<EOF > $CNI_CONFIG_DIR/09-weave.conflist
{
    "cniVersion": "0.3.0",
    "name": "containernet",
      "plugins": [
        {
            "name": "containernet",
            "type": "weave-net",
            "hairpinMode": true
        },
        {
            "type": "portmap",
            "capabilities": {"portMappings": true},
            "snat": true
        }
    ]
}
EOF

}

function remove_cni_config_file()
{
    # TODO: Remove 10-weave.conf post 3.6 release. From 3.6 onwards, we use conflist instead.
    # See https://platform9.atlassian.net/browse/PMK-1186 for more information
    rm -f ${CNI_CONFIG_DIR}/10-weave.conf || echo "Either file not present or unable to delete. Continuing"
    rm -f ${CNI_CONFIG_DIR}/09-weave.conflist || echo "Either file not present or unable to delete. Continuing"
}

function delete_docker0_bridge_if_present()
{
   # Opportunistically delete docker0 bridge
   ip link set dev docker0 down || true
   ip link del docker0 || true
}

