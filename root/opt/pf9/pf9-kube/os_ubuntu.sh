#@IgnoreInspection BashAddShebang
source defaults.env
source runtime.sh

DOCKER_PACKAGE="docker-ce"
DOCKER_PACKAGE_VERSION="5:19.03.11~3-0~ubuntu-$(lsb_release -cs)"
DOCKER_CLI="docker-ce-cli"
DOCKER_CLI_VERSION="5:19.03.11~3-0~ubuntu-$(lsb_release -cs)"
CONTAINERD_PACKAGE="containerd.io"
CONTAINERD_PACKAGE_VERSION="1.4.12-1"

export DEBIAN_FRONTEND=noninteractive

function configure_docker_storage()
{
    # Use the default storage driver
    DOCKER_STORAGE_DRIVER=""
    DOCKER_STORAGE_OPTS="[ ]"
}

function runtime_running()
{
    pf9ctr_is_active
}

function runtime_start()
{
    if [ "$pf9_kube_http_proxy_configured" = "true" ]; then
        echo "docker configuration: http proxy configuration detected; injecting env vars using a drop-in file"
        if [ "$RUNTIME" == "containerd" ]; then
            configure_containerd_http_proxy
        else
            configure_docker_http_proxy
        fi
    else
        echo "docker configuration: http proxy configuration not detected"
    fi
    remove_runtime_sock_dir_if_present $socket
    pf9ctr_enable
    pf9ctr_start
    wait_until "stat $socket &> /dev/null" 5 12

}

function runtime_stop()
{
    pf9ctr_stop
}

function runtime_repo_installed()
{
    if [ "$DOCKER_UBUNTU_REPO_URL" ]; then
        apt-cache policy | grep -q "$DOCKER_UBUNTU_REPO_URL"
    else 
        apt-cache policy | grep -q "https://download.docker.com/linux/ubuntu $(lsb_release -cs)/stable"
    fi
}

function install_runtime_repo()
{
    # Install pre-requisites to installing the repo
    apt-get -y update
    apt-get -y install apt-transport-https ca-certificates

    # Add the repository key
    apt-key add ${DOCKER_UBUNTU_REPO_KEY}

    # Install the repository
    echo "deb [arch=amd64] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" > /etc/apt/sources.list.d/docker.list
    
    # if the upstream repo is not available use the updated repo
    if [ "$DOCKER_UBUNTU_REPO_URL" ]; then
        echo "deb $DOCKER_UBUNTU_REPO_URL" >> /etc/apt/sources.list.d/docker.list
    fi

    # Fetch a list of packages from the installed repository
    apt-get -y update
}


function docker_essentials_installed()
{
    local installed_docker_package_version
    # Evaluates to an empty string if the package is not installed
    installed_docker_package_version="$(dpkg -s $DOCKER_PACKAGE 2> /dev/null|grep Version:|cut -d' ' -f2)"

    local installed_docker_cli_version
    installed_docker_cli_version="$(dpkg -s $DOCKER_CLI 2> /dev/null|grep Version:|cut -d' ' -f2)"

    [ "$installed_docker_package_version" == "$DOCKER_PACKAGE_VERSION" ] && [ "$installed_docker_cli_version" == "$DOCKER_CLI_VERSION" ]
}

function containerd_essentials_installed()
{
    local installed_containerd_version
    installed_containerd_version="$(dpkg -s $CONTAINERD_PACKAGE 2> /dev/null|grep Version:|cut -d' ' -f2)"

    [ "$installed_containerd_version" == "$CONTAINERD_PACKAGE_VERSION" ]
}


function runtime_essentials_installed()
{
    if [ "$RUNTIME" != "containerd" ]; then
        docker_essentials_installed
        local returncode=$?
        if [ $returncode != 0 ]; then
            return $returncode
        fi
    fi

    containerd_essentials_installed
    local returncode=$?
    if [ $returncode == 1 ]; then
        return $returncode
    fi
}

function remove_incompatible_packages()
{
    apt-get -y purge docker lxc-docker docker-engine
}

function install_docker_essentials()
{
    local CURRENT_DOCKERCLI_VERSION=`dpkg-query --showformat='${Version}' --show docker-ce-cli || true`
    if [ -n "${CURRENT_DOCKERCLI_VERSION}" ] && [ "$CURRENT_DOCKERCLI_VERSION" != "$DOCKER_CLI_VERSION" ]
    then
        # In case of the CLI is a different version
        echo "Remove docker-ce-cli due to version mismatch. Current: $CURRENT_DOCKERCLI_VERSION, Expected: $DOCKER_CLI_VERSION"
        apt-get -y purge $DOCKER_CLI
    fi
    local CURRENT_DOCKER_VERSION=`dpkg-query --showformat='${Version}' --show $DOCKER_PACKAGE || true`
    local CURRENT_CONTAINERD_VERSION=`dpkg-query --showformat='${Version}' --show $CONTAINERD_PACKAGE || true`
    if [ "$CURRENT_DOCKER_VERSION" != "$DOCKER_PACKAGE_VERSION" ] || [ "$CURRENT_CONTAINERD_VERSION" != "$CONTAINERD_PACKAGE_VERSION" ]
    then
        # This condition is needed in case docker needs a downgrade. See
        # PMK-1293 for details on the scenario
        echo "Remove docker-ce due to version mismatch. Current: $CURRENT_DOCKER_VERSION, Expected: $DOCKER_PACKAGE_VERSION"
        apt-get -y purge $DOCKER_PACKAGE $CONTAINERD_PACKAGE
    fi

    apt-get -y update
    apt-get -y install "$DOCKER_CLI=$DOCKER_CLI_VERSION" "$DOCKER_PACKAGE=$DOCKER_PACKAGE_VERSION" "$CONTAINERD_PACKAGE=$CONTAINERD_PACKAGE_VERSION"

}

function install_containerd_essentials()
{
    # Install Containerd
    local CURRENT_CONTAINERD_VERSION=`dpkg-query --showformat='${Version}' --show $CONTAINERD_PACKAGE || true`
    if [ -n "${CURRENT_CONTAINERD_VERSION}" ] && [ "$CURRENT_CONTAINERD_VERSION" != "$CONTAINERD_PACKAGE_VERSION" ]
    then
        # This condition is needed in case docker needs a downgrade. See
        # PMK-1293 for details on the scenario
        echo "Remove containerd due to version mismatch. Current: $CURRENT_CONTAINERD_VERSION, Expected: $CONTAINERD_PACKAGE_VERSION"
        apt-get -y purge $CONTAINERD_PACKAGE
    fi

    apt-get -y update
    apt-get -y install "$CONTAINERD_PACKAGE=$CONTAINERD_PACKAGE_VERSION"
}

function install_runtime_essentials()
{
    remove_incompatible_packages
    if [ "$RUNTIME" == "containerd" ]; then
        install_containerd_essentials
    else
        install_docker_essentials
    fi
}


function ensure_bridge_utils_installed()
{
    dpkg -s bridge-utils &> /dev/null || apt-get -y install bridge-utils
}

function _generate_kubelet_systemd_unit()
{
    echo "Generating runtime systemd unit for kubelet"
    local kubelet_args=$1
    cat /etc/pf9/kube.env | awk '{print $2}' > /etc/pf9/kubelet.env
    if [ "$pf9_kube_http_proxy_configured" = "true" ]; then
        echo "kubelet configuration: http proxy configuration detected; appending proxy env vars to /etc/pf9/kubelet.env"
        configure_kubelet_http_proxy
    else
        echo "kubelet configuration: http proxy configuration not detected"
    fi
    sed -e "s|__KUBELET_BIN__|${KUBELET_BIN}|g" \
        -e "s|__KUBELET_ARGS__|${kubelet_args}|g" \
        ${PF9_KUBELET_SYSTEMD_UNIT_TEMPLATE} > ${SYSTEMD_RUNTIME_UNIT_DIR}/pf9-kubelet.service
}

function configure_kubelet_http_proxy()
{
    local kubelet_env="/etc/pf9/kubelet.env"
    echo "HTTP_PROXY=$http_proxy" >> $kubelet_env
    echo "HTTPS_PROXY=$https_proxy" >> $kubelet_env
    echo "NO_PROXY=$no_proxy" >> $kubelet_env
    echo "http_proxy=$http_proxy" >> $kubelet_env
    echo "https_proxy=$https_proxy" >> $kubelet_env
    echo "no_proxy=$no_proxy" >> $kubelet_env
}

function os_specific_config()
{
    if systemd_resolved_running ; then
        ln -sf /run/systemd/resolve/resolv.conf /etc/resolv.conf
    fi

    if command -v ufw 2> /dev/null; then
        # disable ufw if it is available. On Ubuntu18, some systems
        # don't have ufw
        ufw disable
    fi
}

function os_specific_kubelet_setup()
{
    local kubelet_args="$1"
    _generate_kubelet_systemd_unit "$kubelet_args"
    systemctl daemon-reload
}

function os_specific_kubelet_start()
{
    systemctl start pf9-kubelet
}

function os_specific_kubelet_stop()
{
    systemctl stop pf9-kubelet
}

function os_specific_kubelet_running()
{
    systemctl is-active pf9-kubelet
}

function systemd_resolved_running()
{
    systemctl is-active systemd-resolved
}

function configure_docker_http_proxy()
{
    local override_cfg="${DOCKER_DROPIN_DIR}/00-pf9-proxy.conf"

    mkdir -p "$DOCKER_DROPIN_DIR"
    cat > "$override_cfg" <<EOF
[Service]
Environment="http_proxy=$http_proxy" "https_proxy=$https_proxy" "HTTP_PROXY=$HTTP_PROXY" "HTTPS_PROXY=$HTTPS_PROXY" "no_proxy=$no_proxy" "NO_PROXY=$NO_PROXY"
EOF

    systemctl daemon-reload
}

function configure_containerd_http_proxy()
{
    local override_cfg="${CONTAINERD_DROPIN_DIR}/00-pf9-proxy.conf"
    mkdir -p "$CONTAINERD_DROPIN_DIR"
    cat > "$override_cfg" <<EOF
[Service]
Environment="HTTP_PROXY=${HTTP_PROXY}"  
Environment="HTTPS_PROXY=${HTTPS_PROXY}"
Environment="NO_PROXY=${NO_PROXY}"
EOF
    systemctl daemon-reload
    systemctl restart containerd
}
