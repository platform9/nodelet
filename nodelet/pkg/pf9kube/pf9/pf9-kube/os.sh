source /etc/os-release

if [[ "$ID" == "ubuntu" ]]; then
    if [[ "$VERSION_ID" == "16.04" || "$VERSION_ID" == "18.04" || "$VERSION_ID" == "20.04" || "$VERSION_ID" == "22.04" ]]; then
        source os_ubuntu.sh
        export OS_FAMILY="ubuntu"
        export OS_VERSION="${VERSION_ID}"
    else
        echo "Unknown Ubuntu version: ${VERSION_ID}"
        exit 1
    fi
elif [[ "$ID" == "centos" || "$ID" == "rhel" || "$ID" == "rocky" ]]; then
    source os_centos.sh
    export OS_FAMILY="centos"
    if [[ "$VERSION_ID" =~ 9.* ]]; then
        export OS_VERSION="9.x"
    elif [[ "$VERSION_ID" =~ 8.* ]]; then
        export OS_VERSION="8.x"
    elif [[ "$VERSION_ID" =~ 7.* ]]; then
        export OS_VERSION="7.x"
    else
        echo "Unknown CentOS/RHEL version: ${VERSION_ID}"
        exit 1
    fi
else
    echo "Unknown OS: ${ID}"
    exit 1
fi

export KEEPALIVED_PACKAGE_DIR="/opt/pf9/pf9-kube/keepalived_packages"

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
    local k8s_registry="${K8S_PRIVATE_REGISTRY:-registry.k8s.io}"
    local pause_img="${k8s_registry}/pause:3.6"
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
    sandbox_image = "$pause_img"
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
EOF

if [ "$CONTAINERD_CGROUP" = "systemd" ]; then
    cat >> "/etc/containerd/config.toml" <<EOF
     [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc.options]
        SystemdCgroup = true
EOF
else
   cat >> "/etc/containerd/config.toml" <<EOF
     [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc.options]
        SystemdCgroup = false
EOF
fi
 pf9ctr_restart
}

function set_containerd_sock_permissions_pf9()
{
    setfacl -m user:pf9:rwx /run/containerd/containerd.sock || chown pf9:pf9group /run/containerd/containerd.sock
}

function configure_containerd()
{
    load_containerd_kernel_modules
    set_containerd_sysctl_params
    containerd_config
    set_containerd_sock_permissions_pf9
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
    # tr ',' ' ' - replaces commas with one space
    # sed 's| |", |g' - convert space to 'double-quote comma space' ('", ')
    # sed 's|http|"http|g' - prepend http character with double quote
    # sed 's|$|"|' - append a trailing double quote to close out last string
    # sed 's|^|[|' - prepend starting square bracket '['
    # sed 's|$|]|' - append a closing square bracket ']'
    DOCKER_REGISTRY_MIRRORS=$(echo "$REGISTRY_MIRRORS" | tr ',' ' ' | sed 's| |", |g' | sed 's|http|"http|g' | sed 's|$|"|' | sed 's|^|[|' | sed 's|$|]|')

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

function ensure_keepalived_installed()
{
  IS_KEEPALIVED_INSTALLED=-1
  check_keepalived_installed
  if [ $IS_KEEPALIVED_INSTALLED == 0 ]; then
    echo "Expected Keepalived is not found, so (re)installing"
    install_keepalived
  fi
}

function check_keepalived_installed()
{
  local keepalived_status=$(hash keepalived 2>/dev/null; echo $?)
  if [ $keepalived_status -eq 0 ]; then
    echo "Keepalived found, checking version"
    local keepalived_version_installed=$(keepalived --version 2>&1 >/dev/null | head -1 | cut -d " " -f 2)
    echo "keepalived installed version = ${keepalived_version_installed}"
    local keepalived_version_expected=$(get_expected_keepalived_version)
    if [ ${keepalived_version_installed} == ${keepalived_version_expected} ]; then
      echo "Expected Keepalived version ${keepalived_version_expected} is installed"
      IS_KEEPALIVED_INSTALLED=1
      return
    fi
    echo "Keepalived version ${keepalived_version_expected} expected but ${keepalived_version_installed} is found"
    IS_KEEPALIVED_INSTALLED=0
    return
  fi

  echo "Keepalived is not installed"
  IS_KEEPALIVED_INSTALLED=0
  return
}