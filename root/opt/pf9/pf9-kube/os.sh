source /etc/os-release

if [[ "$ID" == "ubuntu" ]]; then
    if [[ "$VERSION_ID" == "16.04" || "$VERSION_ID" == "18.04" || "$VERSION_ID" == "20.04" ]]; then
        source os_ubuntu.sh
        export OS_FAMILY="ubuntu"
    else
        echo "Unknown Ubuntu version: ${VERSION_ID}"
        exit 1
    fi
elif [[ "$ID" == "centos" || "$ID" == "rhel" ]]; then
    source os_centos.sh
    export OS_FAMILY="centos"
else
    echo "Unknown OS: ${ID}"
    exit 1
fi

function ensure_runtime_installed_and_stopped()
{
    ensure_runtime_repo_installed
    ensure_runtime_essentials_installed

    if runtime_running; then
        runtime_stop
    fi
}

function ensure_runtime_repo_installed()
{
    install_runtime_repo
}

function ensure_runtime_essentials_installed()
{
    if ! runtime_essentials_installed; then
        echo "Install runtime essentials.."
        install_runtime_essentials
    fi
}

function configure_docker()
{
    configure_docker_storage
    if [ "$pf9_kube_http_proxy_configured" = "true" ]; then
        configure_docker_http_proxy
    fi
    write_docker_daemon_json
}

function containerd_config()
{
    echo "Generating config.toml"
    mkdir -p /etc/containerd
    cat > "/etc/containerd/config.toml" <<EOF
version = 2
root = "/var/lib/containerd"
state = "/run/containerd"
plugin_dir = ""
disabled_plugins = []
required_plugins = []
oom_score = 0
[plugins]
  [plugins."io.containerd.grpc.v1.cri"]
    [plugins."io.containerd.grpc.v1.cri".registry]
      [plugins."io.containerd.grpc.v1.cri".registry.mirrors]
        [plugins."io.containerd.grpc.v1.cri".registry.mirrors."platform9.io"]
          endpoint = ["https://dockermirror.platform9.io"]
        [plugins."io.containerd.grpc.v1.cri".registry.mirrors."docker.io"]
          endpoint = ["https://dockermirror.platform9.io", "https://registry-1.docker.io"]
  [plugins."io.containerd.grpc.v1.cri".containerd]
    snapshotter = "overlayfs"
  [plugins."io.containerd.grpc.v1.cri".containerd.runtimes]
    [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc]
      runtime_type = "io.containerd.runc.v2"
      [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc.options]
        SystemdCgroup = true
EOF
    pf9ctr_restart
}


function configure_containerd()
{
    load_containerd_kernel_modules
    set_containerd_sysctl_params
    containerd_config
    if [ "$pf9_kube_http_proxy_configured" = "true" ]; then
        configure_containerd_http_proxy
    fi
}

function load_containerd_kernel_modules()
{
    mkdir -p /etc/modules-load.d/
    modprobe overlay
    modprobe br_netfilter
    cat <<EOF | sudo tee /etc/modules-load.d/containerd.conf
overlay
br_netfilter
EOF
}

function set_containerd_sysctl_params()
{
    mkdir -p /etc/sysctl.d/
    cat <<EOF | sudo tee /etc/sysctl.d/pf9-kubernetes-cri.conf
net.bridge.bridge-nf-call-iptables  = 1
net.ipv4.ip_forward                 = 1
net.bridge.bridge-nf-call-ip6tables = 1
EOF
    sysctl --system
}

function configure_runtime()
{
    if [ "$RUNTIME" == "containerd" ]; then
        configure_containerd
    else
        configure_docker
    fi
}

function write_docker_daemon_json()
{
    if [ -f "$DOCKER_DAEMON_JSON" ]; then
        cp "$DOCKER_DAEMON_JSON" "$DOCKER_DAEMON_JSON.orig"
    fi

    # Parses a comma-separated string of docker registry mirrors and converts
    # it in into an array of mirrors. For example,
    # REGISTRY_MIRRORS="https://mirror1.io,https://mirror2.io"
    # is converted to
    # DOCKER_REFISTRY_MIRRORS="[\"https://mirror1.io\", \"https://mirror2.io\"]"
    # which is then populated in docker's daemon.json file
    DOCKER_REGISTRY_MIRRORS=$(echo "$REGISTRY_MIRRORS" | /opt/pf9/python/bin/python -c 'import sys; print(str(sys.stdin.read().strip().split(",")))')
    DOCKER_REGISTRY_MIRRORS=$(echo ${DOCKER_REGISTRY_MIRRORS} | sed "s/'/\"/g" )

    mkdir -p /etc/docker
    prepare_docker_daemon_json \
        < "${CONF_SRC_DIR}/daemon.json" \
        > /etc/docker/daemon.json
}

function prepare_docker_daemon_json()
{
    sed -e "s|__DOCKER_GRAPH__|${DOCKER_GRAPH}|g" \
        -e "s|__DOCKER_SOCKET_GROUP__|${DOCKER_SOCKET_GROUP}|g" \
        -e "s|__DOCKER_CGROUP__|${DOCKER_CGROUP}|g" \
        -e "s|__DOCKER_LOG_DRIVER__|${DOCKER_LOG_DRIVER}|g" \
        -e "s|__DOCKER_LOG_MAX_SIZE__|${DOCKER_LOG_MAX_SIZE}|g" \
        -e "s|__DOCKER_LOG_MAX_FILE__|${DOCKER_LOG_MAX_FILE}|g" \
        -e "s|__DOCKER_STORAGE_DRIVER__|${DOCKER_STORAGE_DRIVER}|g" \
        -e "s|__DOCKER_STORAGE_OPTS__|${DOCKER_STORAGE_OPTS}|g" \
        -e "s|__DOCKER_DEBUG_FLAG__|${DEBUG}|g" \
        -e "s|__DOCKER_LIVE_RESTORE_ENABLED__|${DOCKER_LIVE_RESTORE_ENABLED}|g" \
        -e "s|__DOCKER_REGISTRY_MIRRORS__|${DOCKER_REGISTRY_MIRRORS}|g"
}
