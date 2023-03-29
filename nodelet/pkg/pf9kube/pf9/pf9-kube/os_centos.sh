source defaults.env
source runtime.sh

PWD=$(pwd)
PF9_TE_FILE="$PWD/pf9.te"

DOCKER_PACKAGE="docker-ce"
DOCKER_CLI="docker-ce-cli"
CONTAINERD_PACKAGE="containerd.io"

if [[ "$VERSION_ID" =~ ^8.* ]]; then
    DOCKER_PACKAGE_VERSION="3:20.10.6-3.el8"
    DOCKER_CLI_VERSION="1:20.10.6-3.el8"
    CONTAINERD_PACKAGE_VERSION="1.4.12-3.1.el8"
else
    DOCKER_PACKAGE_VERSION="19.03.11-3.el7"
    DOCKER_CLI_VERSION="19.03.11-3.el7"
    CONTAINERD_PACKAGE_VERSION="1.4.12-3.1.el7"
fi



is_selinux_installed() {
    # Checks the existence of `getenforce` to confirm
    # whether SELinux is installed on the host
    local getenforce=$(command -v getenforce)
    if [ $getenforce ] && [ $(getenforce) != "Disabled"  ]
    then
        return 0
    else
        return 1
    fi
}

create_pf9_selinux_te_file() {
    # Create a SElinux type enforcement file that is later
    # installed as a policy module
    cat<<EOF > ${PF9_TE_FILE}
module pf9 1.0;
require {
    type usr_t;
    type useradd_t;
    type tmp_t;
    type keepalived_t;
    type ifconfig_t;
    type initrc_var_log_t;
    class file write;
    class dir { read write };
    class capability { setgid setuid dac_override };
}
#============= useradd_t ==============
allow useradd_t usr_t:dir write;
#============= keepalived_t ==============
allow keepalived_t self:capability { setgid setuid dac_override };
allow keepalived_t tmp_t:dir { read write };
#============= ifconfig_t ==============
allow ifconfig_t initrc_var_log_t:file write;
EOF
}

# execute_checkmodule creates a SELinux module from a type enforcement file.
execute_checkmodule() {
    local checkmodule=$(command -v "checkmodule")
    if [ $checkmodule ] && [ -f $PF9_TE_FILE ] ; then
        # Convert type enforcement file into a module
        checkmodule -M -m -o ./pf9.mod ./pf9.te >/dev/null
    fi
}

# execute_semodule_package creates a SELinux policy module package from
# module file to be included in the package
execute_semodule_package() {
    local semodule_package=$(command -v "semodule_package")
    if [ $semodule_package ] && [ -f ./pf9.mod ] ; then
        # Compile the policy module
         semodule_package -o ./pf9.pp -m ./pf9.mod
    fi
}

#execute_semodule installs the pf9 SELinux policy module.
execute_semodule() {
    local semodule=$(command -v "semodule")
    if [ $semodule ] && [ -f ./pf9.pp ] ; then
        # Install the policy module
        semodule -i ./pf9.pp
    fi
}

build_and_install_pf9_selinux_policy() {
    execute_checkmodule
    execute_semodule_package
    execute_semodule

    # Clean up the files
    rm -f ./pf9.te ./pf9.mod ./pf9.pp || true
}

function configure_docker_storage()
{
    # check if theere is existind docker daemon.json
    if [ -f /etc/docker/daemon.json ]; then
        echo "existing docker daemon.json found, reading the storage opts from the existing configuration"
        DOCKER_STORAGE_DRIVER=`cat /etc/docker/daemon.json | /opt/pf9/pf9-kube/bin/jq '.["storage-driver"]' | xargs`
        DOCKER_STORAGE_OPTS=`cat /etc/docker/daemon.json | /opt/pf9/pf9-kube/bin/jq '.["storage-opts"]' | xargs`
        if [ "$DOCKER_STORAGE_DRIVER" == "null" ]; then
            # use default value
            DOCKER_STORAGE_DRIVER="overlay2"
        fi

        if [ "$DOCKER_STORAGE_OPTS" == "null" ]; then
            DOCKER_STORAGE_OPTS="[]"
        fi
    fi
}

function runtime_running()
{
    pf9ctr_is_active
}

function runtime_start()
{
    remove_unused_docker_drop_in
    if [ "$pf9_kube_http_proxy_configured" = "true" ]; then
        echo "docker configuration: http proxy configuration detected; injecting env vars using a drop-in file"
        configure_docker_http_proxy
    else
        echo "docker configuration: http proxy configuration not detected"
    fi
    ensure_docker_vg_activated
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
    yum repolist | grep -qs "Docker CE Stable"
}

function install_runtime_repo()
{
    # Add the repository key
    rpm --import ${DOCKER_CENTOS_REPO_KEY}

    # use $'' to make sure \n are interpreted correctly
    local docker_repo_string=$'[dockerrepo]\nname=Docker CE Stable - \$basearch\n'

    if [ ! -z $DOCKER_CENTOS_REPO_URL ]; then
         # Install the repository
    cat > /etc/yum.repos.d/docker.mirrors <<EOF
$DOCKER_CENTOS_REPO_URL
https://download.docker.com/linux/centos/7/\$basearch/stable
EOF
        # use $'' to make sure \n are interpreted correctly
        docker_repo_string+=$'mirrorlist=file:///etc/yum.repos.d/docker.mirrors\n'
    else
        # use the default configuration
        docker_repo_string+=$'baseurl=https://download.docker.com/linux/centos/7/\$basearch/stable\nenabled=1\ngpgcheck=1\n'
    fi

    # Install the repository
    echo "$docker_repo_string" > /etc/yum.repos.d/docker.repo
}

function docker_essentials_installed()
{
    if ! rpm -q "$DOCKER_PACKAGE-$DOCKER_PACKAGE_VERSION"; then
        return 1
    fi
    if ! rpm -q "$DOCKER_CLI-$DOCKER_CLI_VERSION"; then
        return 1
    fi
    return 0
}

function containerd_essentials_installed()
{
    if ! rpm -q "$CONTAINERD_PACKAGE-$CONTAINERD_PACKAGE_VERSION"; then
        return 1
    fi
    return 0
}

function remove_incompatible_packages()
{
    for pkg in \
        docker \
        docker-engine \
        docker-selinux \
        docker-client \
        docker-common \
        docker-forward-journald \
        container-selinux \
        ; do
        if rpm -q ${pkg} &> /dev/null ; then
            echo removing ${pkg}
            yum -y erase ${pkg}
        fi
    done
}

function runtime_essentials_installed()
{
    if [ "$RUNTIME" != "containerd" ]; then
        docker_essentials_installed
        local returncode=$?
        if [ $returncode == 1 ]; then
            return $returncode
        fi
    fi

    containerd_essentials_installed
    local returncode=$?
    if [ $returncode == 1 ]; then
        return $returncode
    fi
}

function install_docker_essentials()
{
    if ! rpm -q "$DOCKER_CLI-$DOCKER_CLI_VERSION"
    then
        # Docker CLI may be a different version
        echo "Removing $DOCKER_CLI. It is not at expected version."
        yum -y erase "$DOCKER_CLI"
    fi

    if ! rpm -q "$DOCKER_PACKAGE-$DOCKER_PACKAGE_VERSION" || ! rpm -q "$CONTAINERD_PACKAGE-$CONTAINERD_PACKAGE_VERSION"
    then
        # This condition is needed in case docker needs a downgrade. See
        # PMK-1293 for details on the scenario
        echo "Removing $DOCKER_PACKAGE and $CONTAINERD_PACKAGE. It is not at expected version."
        yum -y erase "$DOCKER_PACKAGE" "$CONTAINERD_PACKAGE"
    fi
    yum -y install "$DOCKER_CLI-$DOCKER_CLI_VERSION" "$DOCKER_PACKAGE-$DOCKER_PACKAGE_VERSION" "$CONTAINERD_PACKAGE-$CONTAINERD_PACKAGE_VERSION"
}

function install_containerd_essentials()
{
    if ! rpm -q "$CONTAINERD_PACKAGE-$CONTAINERD_PACKAGE_VERSION"
    then
        echo "Removing $CONTAINERD_PACKAGE. It is not at expected version."
        yum -y erase "$CONTAINERD_PACKAGE"
    fi
    yum -y install "$CONTAINERD_PACKAGE-$CONTAINERD_PACKAGE_VERSION"

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
    rpm -q bridge-utils &> /dev/null || yum -y install bridge-utils
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
    if is_selinux_installed
    then
        create_pf9_selinux_te_file || true
        build_and_install_pf9_selinux_policy || true
    fi
}

function os_specific_kubelet_setup()
{
    local kubelet_args="$1"
    _generate_kubelet_systemd_unit "$kubelet_args"
    systemctl daemon-reload
    # Master component containers won't stay running unless selinux is in permissive mode
    # set selinux mode to Permissive when already it is in Enforcing mode. Ignore if the mode is Disabled by default
    # ret=`getenforce`
    # if [ "${ret}" == "Enforcing" ]; then
    #    sed -i s/SELINUX=enforcing/SELINUX=permissive/g /etc/selinux/config
    #    # set selinux mode to Permissive(0)
    #    setenforce Permissive
    #fi
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

# Activates the docker volume group, in case auto-activation failed. Returns an
# error only if both the docker volume group and the LVM tool required to
# manually activate it are present, but manual activation fails. Otherwise logs
# a message. Note: LVM tools must be run with superuser privileges.
#
# Workaround for IAAS-7398.
function ensure_docker_vg_activated()
{
    if [ ! -x "$VGS" ]; then
        echo "activate docker volume group: LVM tool vgs is unavailable"
        return
    fi
    if ! "$VGS" "$DOCKER_VOLUME_GROUP"; then
        echo "activate docker volume group: volume group '${DOCKER_VOLUME_GROUP}' not found"
        return
    fi
    if [ ! -x "$VGCHANGE" ]; then
        echo "activate docker volume group: volume group found, but LVM tool vgchange is unavailable"
        return
    fi
    # Succeeds if volume group is already activated
    "$VGCHANGE" --activate y "$DOCKER_VOLUME_GROUP"
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

# FIXME Remove after 3.1
function remove_unused_docker_drop_in()
{
    if [ -f "${DOCKER_DROPIN_DIR}/00-pf9-limits.conf" ]; then
        rm -f "${DOCKER_DROPIN_DIR}/00-pf9-limits.conf"
    fi
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

function install_keepalived()
{
  echo "Removing keepalived"
  # remove keepalived
  yum erase -y keepalived

  echo "Installing keepalived"
  # install keepalived
  if [[ "$ID" == "centos" ]]; then
    yum install -y $KEEPALIVED_PACKAGE_DIR/keepalived-2.1.3-1.el7.x86_64.rpm
  elif [[ "$ID" == "rhel" ]]; then
    yum install -y $KEEPALIVED_PACKAGE_DIR/keepalived-2.1.5-9.el8.x86_64.rpm
  fi
}