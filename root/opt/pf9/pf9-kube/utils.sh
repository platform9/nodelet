#!/usr/bin/env bash

# Default configuration
source defaults.env

# OS specific functions
source os.sh
source wait_until.sh
source runtime.sh

# declaring global variables for managing certs
declare -A cert_path_to_params_map



# Process extra options.
# Example: if EXTRA_OPTS is defined as FOO=BAR,JANE=BOB
# then the following will be defined:
#  export EXTRA_OPT_FOO=BAR
#  export EXTRA_OPT_JANE=BOB
# Reference:
# http://stackoverflow.com/questions/918886/how-do-i-split-a-string-on-a-delimiter-in-bash
IFS=, read -ra extra_opts <<< "${EXTRA_OPTS}"
for x in ${extra_opts[@]}; do export EXTRA_OPT_$x; done

# Because multiple IP addresses can be assigned to an interface,
# it is conventional to apply distinct labels to assigned IPs,
# so we lookup by label to find a unique assigned IP.
function ipv4_address_of_interface_label()
{
    local iface_label=$1

    local ip=`ip addr show label "$iface_label" | grep -Po 'inet \K[\d.]+'`
    if [ -n "$ip" ]; then
        echo $ip
        return 0
    fi

    echo 1>&2 "No IPv4 address found for interface label $iface_label"
    return 2
}

function ip_address_of_default_gw_nic()
{
    ipv4_ip=`ipv4_address_of_default_gw_nic`
    local ret=$?
    if [ $ret -eq 0 ]; then
        if [ "$ipv4_ip" != "127.0.0.1" ]; then
            echo "$ipv4_ip"
            return ${ret}
        fi
    fi

    ipv6_ip=`ipv6_address_of_default_gw_nic`
    local ret=$?
    if [ $ret -ne 0 ]; then
        echo "127.0.0.1"
        return 0
    fi
    echo "$ipv6_ip"
    return ${ret}
}

function default_gw_nic()
{
    local default_itf=""
    default_itf=`default_v4_gw_nic`

    if [ -z ${default_itf} ]; then
        default_itf=`default_v6_gw_nic`
    fi

    echo "$default_itf"
    return 0
}

function default_v4_gw_nic()
{
    local itf_identifier="V4_INTERFACE"
    local itf_file=/var/opt/pf9/kube_interface_v4
    local cached_itf=""
    local itf=""
    if [ -f $itf_file ]; then
        cached_itf=$(grep "$itf_identifier" $itf_file | awk -F ' ' '{print $2}')
    fi

    if [ -z ${cached_itf} ]; then
        itf=`ip route | grep "default via" | awk '{print $5}'`
    else
        itf=$cached_itf
    fi

    echo "$itf"
    return 0
}

function default_v6_gw_nic()
{
    local itf_identifier="V6_INTERFACE"
    local itf_file=/var/opt/pf9/kube_interface_v6
    local cached_itf=""
    local itf=""
    if [ -f $itf_file ]; then
        cached_itf=$(grep "$itf_identifier" $itf_file | awk -F ' ' '{print $2}')
    fi
    if [ -z ${cached_itf} ]; then
        itf=`ip -6 route | grep "default via" | awk '{print $5}'`
    else
        itf=$cached_itf
    fi

    echo "$itf"
    return 0
}

function ipv6_address_of_default_gw_nic()
{
    local itf=`default_v6_gw_nic`

    local ip=`ip addr show ${itf}|grep 'inet6 '| grep 'global'| awk '{print $2}'| head -n1 | cut -d/ -f1`
    if [ -n "$ip" ]; then
        echo " $itf_identifier $itf" > $itf_file
        echo $ip
        return 0
    fi
    echo 1>&2 No physical interface IPv6 addresses
    return 2
}

function ipv4_address_of_default_gw_nic()
{
    local itf=`default_v4_gw_nic`

    if ! [ -z ${itf} ]; then
        local ip=`ip addr show ${itf}|grep 'inet '|awk '{print $2}'| head -n1 | cut -d/ -f1`
        if [ -n "$ip" ]; then
            echo " $itf_identifier $itf" > $itf_file
            echo $ip
            return 0
        fi
    fi
    echo 1>&2 No default gateway available or physical interface IPv4 addresses
    return 2
}

function ip_addresses_of_nics()
{
    local default_itf=`default_gw_nic`
    if [ -z ${default_itf} ]; then
        echo 1>&2 No default route interface found
        return 1
    fi

    local default_ip=`ip addr show ${default_itf}|grep 'inet '|awk '{print $2}'| head -n1 | cut -d/ -f1`
    if [ "$default_ip" == "" ];then
        default_ip=`ip addr show ${default_itf}|grep 'inet6 '|awk '{print $2}'| head -n1 | cut -d/ -f1`
    fi


    local phys_interfaces=(`ls -l /sys/class/net | grep "\->" | grep -v virtual | awk '{print $9}'`)
    local virtual_interfaces=(`ls -l /sys/class/net | grep "\->" | grep virtual | awk '{print $9}'`)
    # To ensure physical nics if any are listed before virtual nics
    local itfs=( "${phys_interfaces[@]}" "${virtual_interfaces[@]}" )
    if [ "${#itfs[*]}" == "0" ] ; then
        echo 1>&2 No interfaces found
        return 1
    fi
    local out="{ "
    local separator=''
    local iface
    local ip
    local ipvs_iface_prefix="kube-ipvs"
    if [ "$default_ip" != "" ];then
        out="${out}${separator} \"default\":\"$default_itf\""
        separator=','
        out="${out}${separator} \"$default_itf\":\"$default_ip\""
    fi
    for iface in ${itfs[@]}; do
        if [ ${iface} == "lo" ] || [ ${iface} == $default_itf ] || [[ ${iface} == $ipvs_iface_prefix* ]]; then
            continue
        fi
        ip=`ip addr show ${iface}|grep 'inet '|awk '{print $2}'|cut -d/ -f1`
        if [ "$ip" != "" ]; then
            out="${out}${separator} \"$iface\":\"$ip\""
            separator=','
        fi
    done
    out="$out }"
    echo $out
}

function get_node_name_type()
{
    if [ "${CLOUD_PROVIDER_TYPE}" == "aws" ] || [ "${CLOUD_PROVIDER_TYPE}" == "azure" ] || [ "${IPV6_ENABLED}" == "true" ] || [[ $CLOUD_PROVIDER_TYPE == "local" && $USE_HOSTNAME == "true" ]]; then
        echo DNS
    else
        echo IP
    fi
}
function get_node_name()
{
    if [ "${CLOUD_PROVIDER_TYPE}" == "openstack" ]; then
        # Workaround for PMK-993
        result=$(hostname -s)
        # Returns lowercase hostname
        echo ${result,,}
    elif [[ "${CLOUD_PROVIDER_TYPE}" == "aws" ]]; then
        result=$(curl --silent --show-error http://169.254.169.254/latest/meta-data/local-hostname)
        # Returns lowercase local-hostname
        echo ${result,,}
    elif [ "${CLOUD_PROVIDER_TYPE}" == "azure" ] || [ "${IPV6_ENABLED}" == "true" ] || [[ $CLOUD_PROVIDER_TYPE == "local" && $USE_HOSTNAME == "true" ]]; then
        result=$(hostname)
        # Returns lowercase hostname
        echo ${result,,}
    else
        ip_address_of_default_gw_nic
    fi
}

function label_node()
{
    # The label allows pods/users to identify the role of the node (master/worker)
    local node_name=$1
    local node_role=$2
    # It can take a while for kubelet to register the node with the API server, so wait 20 seconds and try 5 times
    wait_until "${KUBECTL} label node ${node_name} --overwrite node-role.kubernetes.io/${node_role}=" 20 5
}

function get_container_logs(){
    local container_name=$1
    comp_log="$(pf9ctr_run inspect $container_name | /opt/pf9/pf9-kube/bin/jq '.[0].LogPath' )"
    current_time=$(date "+%Y.%m.%d-%H.%M.%S")
    if [ ! -z "$comp_log" ] && [ -f $comp_log ]; then
        cp $comp_log "/var/log/pf9/kube/$current_time-$container_name-docker-logs.log"
    fi;
    local comp_log_1=$comp_log
    comp_log_1+='.1'
    if [ ! -z "$comp_log_1" ] && [ -f $comp_log_1 ]; then
        cp $comp_log.1 "/var/log/pf9/kube/$current_time-$container_name-docker-logs.log.1"
    fi
    echo Collected $container_name docker logs
    echo -------------------------------------------------
    echo `sudo pf9ctr_run ps -a`
    echo -------------------------------------------------
    echo `sudo pf9ctr_run images`
    echo -------------------------------------------------
}

function container_running()
{
    if [ "$(pf9ctr_is_active)" != "active" ]; then
        echo "[RUNTIME-DAEMON-FAIL] runtime daemon is not running"
        return 1
    fi

    local socket_name=$1
    local container_name=$2

    if run_state=`pf9ctr_run inspect $container_name | /opt/pf9/pf9-kube/bin/jq '.[0].State.Running' `; then
        if [ "$run_state" == "true" ]; then
            return 0
        else
            echo [$container_name-DOCKER-NOT-RUNNING]
            get_container_logs $container_name
            echo [$container_name-DOCKER-INSPECT]
            echo `pf9ctr_run inspect $container_name`
        fi
    else
        echo [$container_name-DOCKER-FAIL]
    fi

    return 1
}

function ensure_fresh_container_running()
{
    local socket_name=$1
    local run_opts=$2
    local container_name=$3
    local container_img=$4
    local container_cmd=$5
    local container_cmd_args=$6

    ensure_container_destroyed $socket_name $container_name
    cmd="pf9ctr_run \
         run ${container_name:+--name ${container_name}} \
         ${run_opts} ${container_img} ${container_cmd} ${container_cmd_args}"

    # retry 2 times to handle spurious docker errors such as
    # https://github.com/docker/docker/issues/14048
    wait_until "${cmd}" 6 3
}

function ensure_container_stopped_or_nonexistent()
{
    local socket_name=$1
    local container_name=$2
    local run_state
    echo Ensuring $container_name is stopped or non-existent
    if run_state=`pf9ctr_run inspect $container_name | /opt/pf9/pf9-kube/bin/jq '.[0].State.Running' 2> /dev/null`; then
        if [ "$run_state" == "true" ]; then
            echo Stopping $container_name
            pf9ctr_run stop $container_name
        else
            echo $container_name is already stopped
        fi
    else
        echo $container_name does not exist -- ok
    fi
}

function stop_and_destroy_containers()
{
    local socket_name=$1
    local containers=$2
    echo "Stopping containers '$containers'"
    pf9ctr_run stop $containers
    echo "Destroying containers '$containers'"
    pf9ctr_run rm --force $containers
}

function ensure_container_destroyed()
{
    local socket_name=$1
    local container_name=$2
    echo "Ensuring container '$container_name' is destroyed"
    if pf9ctr_run inspect $container_name &> /dev/null; then
        stop_and_destroy_containers "$socket_name" "$container_name"
    fi
}

function get_container_ids_with_crictl()
{
    local cmd="pf9ctr_crictl"
    local container_stop_batch_count=$1
    cri_containers=$(pf9ctr_crictl ps -a | grep -v "POD ID" | awk '{print $1}' | head -n $container_stop_batch_count)
    echo "$cri_containers"
}

function stop_and_destroy_k8s_containers()
{
    local containers=$1
    local cmd
    if [ "$RUNTIME" == "containerd" ]; then
        cmd="pf9ctr_crictl"
    else
        cmd="pf9ctr_run"
    fi
    echo "Stopping containers '$containers'"
    $cmd stop $containers
    echo "Destroying containers '$containers'"
    $cmd rm --force $containers
}

function destroy_all_k8s_containers()
{
    local socket_name=$socket
    local container_stop_batch_count=${CONTAINER_STOP_BATCH_COUNT:=50}
    local container_list
    # Get all the k8s containers first
    while :
    do
        if [ "$RUNTIME" == "containerd" ]; then
            container_list=$(get_container_ids_with_crictl $container_stop_batch_count)
        else
            container_list=$(pf9ctr_run ps -a -q -n $container_stop_batch_count --filter name="^k8s_*" | tr '\n' ' ' )
        fi
        echo "container_list $container_list"
        if [ -z "${container_list}" ]; then
            break
        fi
        stop_and_destroy_k8s_containers "$container_list"
    done

    # Get the proxy container
    local proxy_container="$(pf9ctr_run inspect proxy | /opt/pf9/pf9-kube/bin/jq -r .[0].Id)"
    echo "container_list $proxy_container"
    if ! [ -z "${proxy_container}" ]; then
        stop_and_destroy_containers "$socket_name" "$proxy_container"
    fi
}


function load_image_from_file()
{
    local socket_name=$1
    local path=$2
    pf9ctr_run load --input "$path"
}

function kubelet_running()
{
    os_specific_kubelet_running
}

# IAAS-7212 https://github.com/docker/docker/issues/15912
# Delete the docker socket directory (e.g. /var/run/docker.sock/)
# which can be created by a race condition between dockerd and
# containers mounting the docker socket
function remove_runtime_sock_dir_if_present()
{
    local socket_path=$1
    if [ -d "${socket_path}" ]; then
        rmdir "${socket_path}"
    fi
}

function ensure_kubelet_stopped()
{
    if kubelet_running; then
        os_specific_kubelet_stop
    fi
}

# Assumes all env vars in the file catted are exported
function prepare_kubelet_bootstrap_config
{
    # On a master node, kubelet manages the master pod
    if [ "$ROLE" == "master" ]; then
        export STATIC_POD_PATH="/etc/pf9/kube.d/master.yaml"
    fi
    export CLIENT_CA_FILE="/etc/pf9/kube.d/certs/kubelet/server/ca.crt"
    export TLS_CERT_FILE="/etc/pf9/kube.d/certs/kubelet/server/request.crt"
    export TLS_PRIVATE_KEY_FILE="/etc/pf9/kube.d/certs/kubelet/server/request.key"
    export TLS_CIPHER_SUITES="[TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256]"
    mkdir -p ${KUBELET_CONFIG_DIR}
    ensure_dir_readable_by_pf9 ${KUBELET_CONFIG_DIR}

    cat <<EOF > ${KUBELET_BOOTSTRAP_CONFIG}
apiVersion: kubelet.config.k8s.io/v1beta1
kind: KubeletConfiguration
address: 0.0.0.0
authentication:
  anonymous:
    enabled: false
  webhook:
    enabled: true
  x509:
    clientCAFile: "${CLIENT_CA_FILE}"
authorization:
  mode: AlwaysAllow
clusterDNS:
- "${DNS_IP}"
clusterDomain: "${DNS_DOMAIN}"
cpuManagerPolicy: "${CPU_MANAGER_POLICY}"
topologyManagerPolicy: "${TOPOLOGY_MANAGER_POLICY}"
reservedSystemCPUs: "${RESERVED_CPUS}"
featureGates:
  DynamicKubeletConfig: true
maxPods: 200
readOnlyPort: 0
staticPodPath: "${STATIC_POD_PATH}"
tlsCertFile: "${TLS_CERT_FILE}"
tlsPrivateKeyFile: "${TLS_PRIVATE_KEY_FILE}"
tlsCipherSuites: ${TLS_CIPHER_SUITES}
cgroupDriver: systemd
EOF

    # Apiserver, controller-manager, and scheduler don't run on workers, so don't need staticPodPath (it spams pf9-kubelet journalctl logs)
    if [ "$ROLE" == "worker" ]; then
        sed -i '/staticPodPath*/d' ${KUBELET_BOOTSTRAP_CONFIG}
    fi

    if [ "x${ALLOW_SWAP:-}" == "xtrue" ]; then
        echo "failSwapOn: false" >> ${KUBELET_BOOTSTRAP_CONFIG}
    fi
}

function check_node_using_custom_configmap()
{
    local node_name=$1

    local config_map=$(${KUBECTL} get node ${node_name} -o=jsonpath="{@..configSource.configMap.name}")
    if [ $config_map == ${KUBELET_DEFAULT_MASTER_CONFIGMAP_NAME} ] || [ $config_map == ${KUBELET_DEFAULT_WORKER_CONFIGMAP_NAME} ]; then
        return
    fi
    echo "custom_dynamic_kubeconfig_used" > "${KUBELET_DYNAMIC_CONFIG_DIR}/.dynamic_config"
}

function ensure_node_using_dynamic_configmap()
{
    local node_name=$1

    # By default nodes do not have any values set for configSource, so if it's found, the node has been patched
    local node_patched=$(${KUBECTL} get node ${node_name} -o=jsonpath="{@..configSource}")
    if [[ ${node_patched} ]]; then
        return
    elif [ "$ROLE" == "master" ]; then
        ${KUBECTL} patch node ${node_name} -p "{\"spec\":{\"configSource\":{\"configMap\":{\"name\":\"${KUBELET_DEFAULT_MASTER_CONFIGMAP_NAME}\",\"namespace\":\"kube-system\",\"kubeletConfigKey\":\"kubelet\"}}}}"
    else
        ${KUBECTL} patch node ${node_name} -p "{\"spec\":{\"configSource\":{\"configMap\":{\"name\":\"${KUBELET_DEFAULT_WORKER_CONFIGMAP_NAME}\",\"namespace\":\"kube-system\",\"kubeletConfigKey\":\"kubelet\"}}}}"
    fi
}

# Creates a ConfigMap in kube-system namespace of cluster to be the default kubelet
# config for nodes $ROLE. The first node of type $ROLE to run this function
# will create it, others will check first if it's there, no-op if so.
function ensure_dynamic_kubelet_default_configmap()
{
    if [ "$ROLE" == "master" ]; then
        if ! &>/dev/null ${KUBECTL} get cm -n kube-system ${KUBELET_DEFAULT_MASTER_CONFIGMAP_NAME}; then
            ${KUBECTL} -n kube-system create configmap ${KUBELET_DEFAULT_MASTER_CONFIGMAP_NAME} --from-file=kubelet=${KUBELET_BOOTSTRAP_CONFIG} -o yaml
        else
            echo "${KUBELET_DEFAULT_MASTER_CONFIGMAP_NAME} ConfigMap already exists; not creating"
        fi
    else
        if ! &>/dev/null ${KUBECTL} get cm -n kube-system ${KUBELET_DEFAULT_WORKER_CONFIGMAP_NAME}; then
            # KUBELET_BOOTSTRAP_CONFIG modified if node is a WORKER to _not_ have staticPodPath, but path to config file the same
            ${KUBECTL} -n kube-system create configmap ${KUBELET_DEFAULT_WORKER_CONFIGMAP_NAME} --from-file=kubelet=${KUBELET_BOOTSTRAP_CONFIG} -o yaml
        else
            echo "${KUBELET_DEFAULT_WORKER_CONFIGMAP_NAME} ConfigMap already exists; not creating"
        fi
    fi
}

function ensure_kubelet_running()
{
    if kubelet_running; then
        return 0
    fi

    local node_name=$1
    local kubeconfig="/etc/pf9/kube.d/kubeconfigs/kubelet.yaml"
    local log_dir_path="/var/log/pf9/kubelet/"
    local k8s_registry="${K8S_PRIVATE_REGISTRY:-k8s.gcr.io}"
    local pause_img="${k8s_registry}/pause:3.2"

    prepare_kubelet_bootstrap_config

    mkdir -p $KUBELET_DATA_DIR
    local kubelet_args=" \
        --kubeconfig=${kubeconfig} \
        --enable-server \
        --network-plugin=cni \
        --cni-conf-dir=${CNI_CONFIG_DIR} \
        --cni-bin-dir=${CNI_BIN_DIR} \
        --log-dir=${log_dir_path} \
        --logtostderr=false \
        --config=${KUBELET_BOOTSTRAP_CONFIG} \
        --register-schedulable=false \
        --pod-infra-container-image=${pause_img} \
        --dynamic-config-dir=${KUBELET_DYNAMIC_CONFIG_DIR} \
        --cgroup-driver=systemd"
    
    # container-runtime: The container runtime to use. Possible values: docker, remote
    # container-runtime-endpoint: The endpoint of remote runtime service. Currently unix socket endpoint is supported on Linux
    #                             Examples: unix:///var/run/dockershim.sock or /run/containerd/containerd.sock
    # runtime-request-timeout: Timeout of all runtime requests except long running request - pull/logs/exec/attach.
    #                          When timeout exceeded, kubelet will cancel the request, throw out an error (Default: 2m0s)
    #                          extended the timeout to 15Mins
    # container-log-max-files & container-log-max-size : when kubelet is configured to use alternative container runtime like
    #                                                    containerd, kubelet is one that manages the log files and not the runtime.
    #                                                    These 2 options control the number of log files and their sizes.
    #                                                    Current default values 10 files of 10MB each per container which is the pf9 configuration for docker.
    #                                                    This can be overridden by adding CONTAINER_LOG_MAX_FILES and CONTAINER_LOG_MAX_SIZE to kube_override.env

    if [ "$RUNTIME" == "containerd" ]; then
        local container_log_max_files=${CONTAINER_LOG_MAX_FILES:-${DOCKER_LOG_MAX_FILE}}
        # Why not use DOCKER_LOG_MAX_SIZE variable? 
        # The formatting for docker config is 10m while kubelet expects 10Mi. To avoid implement string manipulation in bash just hardcoding
        # the same default as docker config for now.
        local container_log_max_size=${CONTAINER_LOG_MAX_SIZE:-"10Mi"}
        kubelet_args="${kubelet_args} \
                --container-runtime=remote \
                --runtime-request-timeout=15m \
                --container-runtime-endpoint=unix://${CONTAINERD_SOCKET} \
                --container-log-max-files=${container_log_max_files} \
                --container-log-max-size=${container_log_max_size}"
    fi

    # if CLOUD_PROVIDER_TYPE is not local i.e. AWS, Azure, etc. or if it is local but USE_HOSTNAME is not true then use the node_endpoint (IP address).
    if [ $CLOUD_PROVIDER_TYPE != "local" ] || [[ $CLOUD_PROVIDER_TYPE == "local" && $USE_HOSTNAME != "true" ]]; then
        # if --hostname-override is not specified hostname of the node is used by default
        # in case --hostname-override is specified along with cloud provider then cloud provider determines
        # the hostname
        # https://kubernetes.io/docs/reference/command-line-tools-reference/kubelet/
        kubelet_args+=" --hostname-override=${node_name//:/-}"
    fi

    # Make sure that node IP is selected based on the interface specified in kube_interface cache files when nodename is set as the hostname.
    if [[ $CLOUD_PROVIDER_TYPE == "local" && $USE_HOSTNAME == "true" ]]; then
        kubelet_args+=" --node-ip=${NODE_IP}"
    fi

    if [[ "${CLOUD_PROVIDER_TYPE}" == "aws" ]]; then
        kubelet_args+=" --cloud-provider=aws"
        if [[ "${ENABLE_CAS}" == "true" ]]; then
            local instance_id=$(fetch_aws_instance_id)
            local az=$(fetch_aws_az)
            local trimmed_instance_id=$(trim_sans "$instance_id")
            local trimmed_az=$(trim_sans "$az")
            kubelet_args+=" --provider-id=aws:///${trimmed_az}/${trimmed_instance_id}"
        fi
    elif [[ "${CLOUD_PROVIDER_TYPE}" == "openstack" ]]; then
        kubelet_args+=" --cloud-provider=openstack"
        kubelet_args+=" --cloud-config=${CLOUD_CONFIG_FILE}"
    elif [[ "${CLOUD_PROVIDER_TYPE}" == "azure" ]]; then
        kubelet_args+=" --cloud-provider=azure"
        kubelet_args+=" --cloud-config=${CLOUD_CONFIG_FILE}"
    fi

    if [[ "${DEBUG}" == "true" ]]; then
        kubelet_args+=" --v=8"
    else
        kubelet_args+=" --v=2"
    fi

    os_specific_kubelet_setup "$kubelet_args"
    echo 'Starting kubelet'
    os_specific_kubelet_start "$kubelet_args"
}

function fetch_aws_instance_id()
{
    cat '/var/lib/cloud/data/instance-id'
}

function fetch_aws_az()
{
    local CURL="curl --silent --fail --max-time 3"
    ${CURL} http://${AWS_METADATA_IP}/latest/meta-data/placement/availability-zone
}

function fetch_aws_region()
{
    local CURL="curl --silent --fail --max-time 3"
    ${CURL} http://${AWS_METADATA_IP}/latest/dynamic/instance-identity/document | awk -F'"' '/\"region\"/ { print $4 }'
}

function load_ipv6_kernel_modules()
{
    # Load kernel modules necessary for ipv6
    if [ $(modprobe ip6table_filter) -gt 0  ]; then
         echo Failed to enable ip6table_filter kernel module
         exit 1
    fi
    echo Kernel module ip6table_filter loaded
}

function load_ipvs_kernel_modules()
{
    # Load kernel modules necessary for ipvs kube-proxy
    if [[ $(modprobe ip_vs) -gt 0  ]]; then
         echo Failed to enable ip_vs kernel module
         exit 1
    fi
    echo Kernel module ip_vs loaded

    if [[ $(modprobe ip_vs_rr) -gt 0  ]]; then
         echo Failed to enable ip_vs_rr kernel module
         exit 1
    fi
    echo Kernel module ip_vs_rr loaded

    if [[ $(modprobe ip_vs_wrr) -gt 0  ]]; then
         echo Failed to enable ip_vs_wrr kernel module
         exit 1
    fi
    echo Kernel module ip_vs_wrr loaded

    if [[ $(modprobe ip_vs_sh) -gt 0  ]]; then
         echo Failed to enable ip_vs_sh kernel module
         exit 1
    fi
    echo Kernel module ip_vs_sh loaded


    if [[ $(modprobe nf_conntrack_ipv4) -gt 0  ]]; then
         echo Failed to enable nf_conntrack_ipv4 kernel module
         exit 1
    fi
    echo Kernel module nf_conntrack_ipv4 loaded
}

function ensure_proxy_running()
{
    local node_name=$1
    local kubeconfig="/etc/pf9/kube.d/kubeconfigs/kube-proxy.yaml"
    local kubeconfig_in_container="/etc/kubernetes/pf9/kube-proxy/$(basename $kubeconfig)"

    # kube-proxy relies on the bind address to infer ipv4/ipv6, so fixing that
    local bind_address="0.0.0.0"
    if [[ "${IPV6_ENABLED}" == "true" ]]; then
          bind_address="::"
          load_ipv6_kernel_modules
    fi

    local run_opts="--detach=true \
        --net=host \
        --privileged \
        --volume ${kubeconfig}:${kubeconfig_in_container}"

    local k8s_registry="${K8S_PRIVATE_REGISTRY:-k8s.gcr.io}"
    local container_name="proxy"
    local container_img="${k8s_registry}/kube-proxy:$KUBERNETES_VERSION"

    local container_cmd="kube-proxy"
    local container_cmd_args="--kubeconfig=${kubeconfig_in_container} \
                              --v=2 \
                              --hostname-override=${node_name//:/-} \
                              --proxy-mode ${KUBE_PROXY_MODE} \
                              --cluster-cidr ${CONTAINERS_CIDR} \
                              --bind-address ${bind_address}"

    if [ ! -z "${MAX_NAT_CONN}" ]; then
        container_cmd_args="${container_cmd_args} --conntrack-max-per-core ${MAX_NAT_CONN}"
    fi

    if [[ "${KUBE_PROXY_MODE}" == "ipvs" ]]; then
        echo "Using IPVS mode for kube-proxy"
        load_ipvs_kernel_modules

        # Enable strict ARP mode when using kube-proxy in IPVS mode
        container_cmd_args+=" --ipvs-strict-arp"
    else
        echo "Using iptables mode for kube-proxy"
    fi

    ensure_fresh_container_running $socket "${run_opts}" "${container_name}" "${container_img}" "${container_cmd}" "${container_cmd_args}"
}

# Check if swap space is disabled on both worker and master
# nodes. This will fail pf9-kube at early stage before
# configuring and starting kubelet.
function check_swap_disabled()
{
    if [ $(swapon -s | wc -l) -gt 0  ]; then
        if [ "x${ALLOW_SWAP:-}" != "xtrue" ]; then
            echo Swap is enabled
            exit 1
        fi
    fi
    echo Swap is disabled
}

# Check env vars required by both worker and master roles
# TODO(daniel): replace this with a `set -u` in the scripts
function check_required_params()
{
    if [ -z "$DOCKER_ROOT" ]; then
        echo DOCKER_ROOT not defined
        exit 1
    fi
    echo Docker root directory is set to $DOCKER_ROOT

    if [ -z "$ETCD_DATA_DIR" ]; then
        echo ETCD_DATA_DIR not defined
        exit 1
    fi
    echo etcd data directory is set to $ETCD_DATA_DIR
    if [ -z "$MASTER_IP" ]; then
        echo MASTER_IP not defined
        exit 1
    fi
    if [ "$MASTER_IP" == "0.0.0.0" ]; then
        echo 'MASTER_IP 0.0.0.0 is invalid '
        exit 1
    fi
    echo master IP is $MASTER_IP

    if [ "$CONTAINERS_CIDR" == "" ]; then
        echo CONTAINERS_CIDR not defined
        exit 1
    fi
    echo containers CIDR is $CONTAINERS_CIDR

    if [ "$SERVICES_CIDR" == "" ]; then
        echo SERVICES_CIDR not defined
        exit 1
    fi
    echo services CIDR is $SERVICES_CIDR
}

function set_sysctl_params()
{
    if (( $(sysctl -w net.ipv4.ip_forward=1 1>&2; echo $?) != 0 )); then
        echo "Failed to turn on IPv4 forwarding"
        exit 1
    fi
    echo "Turned on IPv4 forwarding"

    if [ "$IPV6_ENABLED" == "true" ]; then
        if (( $(sysctl -w net.ipv6.conf.all.forwarding=1 1>&2; echo $?) != 0 )); then
            echo "Failed to turn on IPv6 forwarding"
            exit 1
        fi

        if (( $(sysctl -w net.bridge.bridge-nf-call-iptables=1 1>&2; echo $?) != 0 )); then
            echo "Failed to enable calling iptables for bridge traffic"
            exit 1
        fi
        echo "IPv6 syctl parameters set"
    fi
}

function kubernetes_node_ready()
{
    local output=`${KUBECTL_SILENT} get nodes | grep -v NAME | awk '{print $1"|"$2"|"$3}'`
    local out="["
    for node in $output
    do
        IFS='|' read -r -a array <<< "$node"
        node_status="{\"node\": \"${array[0]}\",\"status\":\"${array[1]}\",\"role\": \"${array[2]}\"}"
        out="$out$node_status,"
    done
    out=${out::-1}
    out="$out]"
    echo $out
}

function kubernetes_api_available()
{
    ADMIN_CERTS=${CONF_DST_DIR}/certs/admin
    #https://github.com/kubernetes/kubernetes/pull/46589 for role bindings to appear
    #healthz is better indication for availability, use https
    if [ "$ROLE" == "master" ]; then
        api_endpoint=localhost
    else
        api_endpoint=`./ip_for_http "$MASTER_IP"`
    fi
    curl --silent https://${api_endpoint}:${K8S_API_PORT}/healthz --cacert ${ADMIN_CERTS}/ca.crt --key ${ADMIN_CERTS}/request.key --cert ${ADMIN_CERTS}/request.crt --fail
}

function cleanup_file_system_state()
{
    if [ "$ROLE" == "none" ]; then
        # Some pod tmpfs mounts (e.g. for serviceaccountkey token and ca cert)
        # aren't cleaned up when a pod's containers are destroyed but the pod
        # itself is not (i.e. during a soft pf9-kube stop with role!=none).
        # When a pod and its containers are restarted, apparently the mounts are not
        # recreated (this could be a bug, see below).
        # Therefore, explicitly unmount them only during full uninstall
        # (which implies complete destruction of all pods)
        # FIXME: investigate what happens after a reboot.
        #        The tmpfs mount could be lost, resulting in errors down the line.
        mount|grep "tmpfs on $KUBELET_DATA_DIR"|awk '{print $3}' |xargs --no-run-if-empty umount

        # Delete pod state stored outside of etcd
        rm -rf $KUBELET_DATA_DIR

        # Delete the dynamic config dir
        rm -rf $KUBELET_DYNAMIC_CONFIG_DIR

        # Delete etcd state
        rm -rf $ETCD_DATA_DIR

        # Delete current configuration
        rm -rf $CONF_DST_DIR/*
    fi
}

function prevent_auto_reattach()
{
    # IAAS-6840
    # Unconditionally delete the qbert metadata file to prevent re-auth
    rm -f /opt/pf9/hostagent/extensions/fetch_qbert_metadata
}

#FIXME(daniel): existence of node entry doesn't mean it's up
# (could be in NotReady state)
# Argument 1: node endpoint
function node_is_up()
{
    local node_name=$1
    ${KUBECTL} get node ${node_name} #1> /dev/null
}

#FIXME(daniel): existence of svc doesn't mean dns is up
function dns_is_up()
{
    ${KUBECTL_SYSTEM} get svc kube-dns #&> /dev/null
}

#FIXME(daniel): existence of svc doesn't mean service-lb is up
function service_loadbalancer_is_up()
{
    ${KUBECTL_SYSTEM} get ds service-loadbalancer #&> /dev/null
}

#FIXME(osoriano): existence of svcs doesn't mean app catalog is up
function appcatalog_is_up()
{
    ${KUBECTL_SYSTEM} get svc monocular-api-svc #&> /dev/null && \
    ${KUBECTL_SYSTEM} get svc tiller-deploy #&> /dev/null
}

#Checking for one should suffice as all are created from same yaml
function role_bindings_exist()
{
    ${KUBECTL} get clusterrolebinding admin-and-pf9-access #&> /dev/null
}

# Argument 1: path of config file relative to config source dir
# Argument 2: command/function to prepare config file for use
# Prepares config file for use and puts in the config directory.
# If any error is encountered, no config file is written.
function prepare_conf_file()
{
    local path=$1
    local sub_func=${2:-cat}
    # -o pipefail will be set inside subshell only
    (
        mkdir -p $(dirname "${CONF_DST_DIR}/${path}")
        set -o pipefail
        if ! cat "${CONF_SRC_DIR}/${path}" | $sub_func > "${CONF_DST_DIR}/${path}"; then
            rm -rf "${CONF_DST_DIR}/${path}"
            return 1
        fi
    )
}

# Prepares kubeconfigs: First, ensures env vars for embedding certs, etc. into
# kubeconfigs are defined, then prepares the kuebconfigs.
function prepare_kubeconfigs()
{
    local kubelet="kubeconfigs/kubelet.yaml"
    local kube_proxy="kubeconfigs/kube-proxy.yaml"
    local admin="kubeconfigs/admin.yaml"

    #                  var prefix               certs_dir
    env_vars_from_cert ADMIN                    admin
    env_vars_from_cert KUBELET                  kubelet/apiserver
    env_vars_from_cert KUBE_PROXY               kube-proxy/apiserver

    local apiserver_host=`./ip_for_http "$MASTER_IP"`
    # When running as a master, ensure kubelet, kube-proxy, and scripts talk to
    # the apiserver on localhost
    if [ "$ROLE" == "master" ]; then
        apiserver_host="localhost"
    fi

    apiserver_host=${apiserver_host}":"${K8S_API_PORT}

    #                 file                     substitution command
    prepare_conf_file $kubelet                 "prepare_kubelet_kubeconfig ${apiserver_host}"
    prepare_conf_file $kube_proxy              "prepare_kube_proxy_kubeconfig ${apiserver_host}"
    prepare_conf_file $admin                   "prepare_admin_kubeconfig ${apiserver_host}"
}

function prepare_rolebindings()
{
    if [ ! -d ${CONF_DST_DIR}/rolebindings ]; then
        mkdir ${CONF_DST_DIR}/rolebindings
    fi
    cp -rf ${CONF_SRC_DIR}/rolebindings/* ${CONF_DST_DIR}/rolebindings/
}

function prepare_admin_kubeconfig()
{
    local apiserver_host=$1

    sed -e "s/__ADMIN_CERT_BASE64__/${ADMIN_CERT_BASE64}/g" \
        -e "s/__ADMIN_KEY_BASE64__/${ADMIN_KEY_BASE64}/g" \
        -e "s/__CA_CERT_BASE64__/${ADMIN_CA_CERT_BASE64}/g" \
        -e "s/__APISERVER_HOST__/${apiserver_host}/g"
}

function prepare_kubelet_kubeconfig()
{
    local apiserver_host=$1

    sed -e "s/__KUBELET_CERT_BASE64__/${KUBELET_CERT_BASE64}/g" \
        -e "s/__KUBELET_KEY_BASE64__/${KUBELET_KEY_BASE64}/g" \
        -e "s/__CA_CERT_BASE64__/${KUBELET_CA_CERT_BASE64}/g" \
        -e "s/__APISERVER_HOST__/${apiserver_host}/g"
}

function prepare_kube_proxy_kubeconfig()
{
    local apiserver_host=$1

    sed -e "s/__KUBE_PROXY_CERT_BASE64__/${KUBE_PROXY_CERT_BASE64}/g" \
        -e "s/__KUBE_PROXY_KEY_BASE64__/${KUBE_PROXY_KEY_BASE64}/g" \
        -e "s/__CA_CERT_BASE64__/${KUBE_PROXY_CA_CERT_BASE64}/g" \
        -e "s/__APISERVER_HOST__/${apiserver_host}/g"
}

function prepare_kube_controller_manager_kubeconfig()
{
    local apiserver_host=$1

    sed -e "s/__KUBE_CONTROLLER_MANAGER_CERT_BASE64__/${KUBE_CONTROLLER_MANAGER_CERT_BASE64}/g" \
        -e "s/__KUBE_CONTROLLER_MANAGER_KEY_BASE64__/${KUBE_CONTROLLER_MANAGER_KEY_BASE64}/g" \
        -e "s/__CA_CERT_BASE64__/${KUBE_CONTROLLER_MANAGER_CA_CERT_BASE64}/g" \
        -e "s/__APISERVER_HOST__/${apiserver_host}/g"
}

function prepare_kube_scheduler_kubeconfig()
{
    local apiserver_host=$1

    sed -e "s/__KUBE_SCHEDULER_CERT_BASE64__/${KUBE_SCHEDULER_CERT_BASE64}/g" \
        -e "s/__KUBE_SCHEDULER_KEY_BASE64__/${KUBE_SCHEDULER_KEY_BASE64}/g" \
        -e "s/__CA_CERT_BASE64__/${KUBE_SCHEDULER_CA_CERT_BASE64}/g" \
        -e "s/__APISERVER_HOST__/${apiserver_host}/g"
}

# Reads cert, etc. into env vars that follow the PREFIX_{CERT,KEY,CA}_BASE64
# convention. These vars are used to embed the cert, etc. into a kubeconfig.
# Argument 1: the PREFIX string
# Argument 2: the subdir where the cert, etc. is found
# Either all env vars are set, or none are
function env_vars_from_cert()
{
    local prefix=$1
    local subdir=$2

    local certs_dir="${CERTS_DIR}/${subdir}"
    local key_file="${certs_dir}/request.key"
    local cert_file="${certs_dir}/request.crt"
    local ca_file="${certs_dir}/ca.crt"

    local cert_base64
    local key_base64
    local ca_base64
    cert_base64=`set -o pipefail; cat "$cert_file" | base64_encode`
    key_base64=`set -o pipefail; cat "$key_file" | base64_encode`
    ca_base64=`set -o pipefail; cat "$ca_file" | base64_encode`
    export "${prefix}_KEY_BASE64=${key_base64}"
    export "${prefix}_CERT_BASE64=${cert_base64}"
    export "${prefix}_CA_CERT_BASE64=${ca_base64}"
}

function base64_encode()
{
    base64 | tr -d '\r\n'
}

function delete_node_from_apiserver()
{
    local node_ip=$1
    if ! err=$(${KUBECTL} delete node "$node_ip" 2>&1 1>/dev/null); then
        echo "Warning: failed to delete node ${node_ip}: ${err}" >&2
    fi
}

function drain_node_from_apiserver()
{
    local node_ip=$1
    # We use a timeout because we have observed this command taking upward of 6 hours to exit.
    # See https://platform9.atlassian.net/browse/PMK-933
    # TODO: Figure out how to have master nodes talk to API FQDN or MASTER_IP for this call only
    # since during a detach, master nodes won't be able to talk to their own apiserver, thus leaving
    # the node as part of the cluster but with status 'NotReady'
    # We disable eviction here, which tells kubectl to not try to evict pods
    # from the node. This implicitly prevents the drain from being stuck on any
    # PDB quotas not being met.
    if ! err=$(${KUBECTL} drain "$node_ip" --ignore-daemonsets --disable-eviction --delete-local-data --force --timeout=5m 2>&1 1>/dev/null); then
        echo "Warning: failed to drain node ${node_ip}: ${err}" >&2
        return
    # Add KubeStackShutDown annotation to the node on successful node drain
    add_annotation_to_node ${node_ip} KubeStackShutDown
    fi
}

function uncordon_node()
{
    local node_name=$1
    # This will succeed even if the node is already uncordoned
    # Errors are fatal. Retry a few times since the API server may take
    # a while to come during initial cluster creation.
    wait_until "${KUBECTL} uncordon ${node_name}" 6 20
    # Observed that qbert azure ubuntu tests fail intermittently because
    # one of the master node does not get uncordoned even after kubectl uncordon
    # completes successfully. Check once again if node is uncordoned properly.
    # Next line should NOT require wait_until assuming that above the kubectl
    # completes
    node_uncordoned=$(${KUBECTL} describe node ${node_name} | grep 'Unschedulable:' | awk '{print $2}')
    if [ "$node_uncordoned" != "false" ]; then
        # This value will be one of 3 values -
        # true: Node is unschedulable i.e. uncordon failed; return 1
        # empty: Most probably api server was unreachable so cannot
        #        gaurantee that uncordon was successful; return 1
        # false: Node is NOT unschedulable i.e. uncordon successful; return 0
        echo "Warning: Node [$node_name] is still cordoned or cannot be fetched"
        return 1
    fi
    return 0
}

function ensure_runtime_stopped()
{
    if runtime_running; then
        runtime_stop
    else
        echo runtime service already stopped
    fi
}

function ensure_dir_readable_by_pf9()
{
    local dir=$1
    chown -R "${PF9_USER}:${PF9_GROUP}" "${dir}"
}

# If certs directory exists and backup succeeds, returns 0
# If certs directory exists and backup fails, returns 1
# If certs directory does not exist, returns 0
function ensure_certs_dir_backedup()
{
    local suffix=$1


    if [ -e "$CERTS_DIR" ]; then
        local certs_backup_dir="$CERTS_DIR.${suffix}.`date +%s`"
        echo backing up certs in "$CERTS_DIR" to "$certs_backup_dir"

        # See PMK-1490, due to a bug in previous change there may be some backed
        # up dirs where the user permissions are not pf9:pf9group. Do an
        # explicit chown to handle this.
        chown -R pf9:pf9group $(dirname $certs_backup_dir)/certs.* || true
        if ! cp -pr "$CERTS_DIR" "$certs_backup_dir"; then
            echo failed to back up certs in "$CERTS_DIR"
            return 1
        else
            return 0
        fi
    else
        echo certs dir "$CERTS_DIR" not found, not backing up
        return 0
    fi
}

function teardown_certs()
{
    if ! ensure_certs_dir_backedup "teardown" ; then
        echo Warning: failed to back up certs directory during teardown
    fi


    if [ -d "$CERTS_DIR" ]; then
        echo "Removing the certs directory"
        rm -rf "$CERTS_DIR" ;
    fi
}

# Returns sed expression to replace a pattern. If the replacement string is
# empty, returns a sed expression to delete any line where the pattern is
# found.
function sub_or_delete_line()
{
    local pattern=$1
    local replacement=$2
    if [ -z "$replacement" ]; then
        echo "/${pattern}/d"
    else
        echo "s|${pattern}|${replacement}|g"
    fi
}

# Writes a valid kubeconfiga to a file. The caller is responsible for creating
# and removing the file. The kubeconfig may include embedded client certs or a
# placeholder for a authentication token, depending on how the cluster is
# configured.
# Argument 1: username
# Argument 2: clustername
# Argument 3: path to kubeconfig file
function make_kubeconfig()
{
    local username="$1"
    local clustername="$2"
    local kubeconfig="$3"
    local kubectl="${KUBECTL_BIN} --kubeconfig=${kubeconfig}"
    local kube_server="$MASTER_IP"
    if [ -n "$EXTERNAL_DNS_NAME" ]; then
        kube_server="$EXTERNAL_DNS_NAME"
    fi
    is_v6=$(/opt/pf9/nodelet/nodeletd advanced is-v6 ${kube_server})
    if [ "$is_v6" != "false" ]; then
        kube_server="[${kube_server}]"
    fi
    if [[ "$kube_server" == "$MASTER_IP" && "$USE_HOSTNAME" == "true" && "$CLOUD_PROVIDER_TYPE" == "local" && "$MASTER_VIP_ENABLED" == "false" ]]; then
        kube_server=$HOSTNAME
    fi 
    if [ "$K8S_API_PORT" != "443" ]; then
        kube_server="${kube_server}:${K8S_API_PORT}"
    fi

    echo "Writing kubeconfig for user ${username} to ${kubeconfig}"

    # Configure cluster
    $kubectl config set-cluster "$clustername" \
        --embed-certs=true \
        --certificate-authority=/etc/pf9/kube.d/certs/admin/ca.crt \
        --server="https://${kube_server}"

    # Configure credentials
    if [ "$KEYSTONE_ENABLED" == "true" ]; then
        $kubectl config set-credentials "$username" \
            --token="__INSERT_BEARER_TOKEN_HERE__"
    else
        $kubectl config set-credentials "$username" \
            --embed-certs=true \
            --client-certificate=/etc/pf9/kube.d/certs/admin/request.crt \
            --client-key=/etc/pf9/kube.d/certs/admin/request.key
    fi

    # Configure context
    $kubectl config set-context default \
        --cluster="$clustername" \
        --namespace=default \
        --user="$username"

    # Set current context
    $kubectl config use-context default
}

# Appends items to the NO_PROXY/no_proxy env vars and exports them immediately
# Argument 1: Item to append
function add_no_proxy()
{
    local item_to_add=$1
    if [ -n "$NO_PROXY" ] || [ -n "$no_proxy" ]; then
        echo "http proxy: adding ${item_to_add} to the NO_PROXY/no_proxy env var"
        export NO_PROXY="${item_to_add},${NO_PROXY}"
        export no_proxy="${item_to_add},${no_proxy}"
    else
        echo "http proxy: no_proxy env vars not defined; defining"
        export NO_PROXY="${item_to_add}"
        export no_proxy="${item_to_add}"
    fi
}

# Ensure pf9-kube scripts and all processes the spawn have the http/s_proxy and
# no_proxy env vars defined in their environments. The following scenarios are
# covered:
# |------------------|------------------------|----------------------------
# | environment vars | pf9-comms proxy config |
# | already defined  | exists and is valid    | action
# |------------------|------------------------|----------------------------
# | true             | true                   | use pf9-comms proxy config
# | true             | false                  | use defined env vars
# | false            | true                   | use pf9-comms proxy config
# | false            | false                  | skip configuration
# |------------------|------------------------|----------------------------
function ensure_http_proxy_configured()
{
    if [ -z "$HTTP_PROXY" ] && [ -z "$http_proxy" ] && [ -z "$HTTPS_PROXY" ] && [ -z "$https_proxy" ] && [ ! -f "${PF9_COMMS_PROXY_CONF}" ]; then
        echo "http proxy: http/s_proxy env vars not defined, no pf9-comms proxy configuration; skipping configuration"
        export pf9_kube_http_proxy_configured="false"
        return
    fi

    local pf9_comms_http_proxy
    local kube_no_proxy

    if [ -f "${PF9_COMMS_PROXY_CONF}" ]; then
        pf9_comms_http_proxy="$(/opt/pf9/python/bin/python parse_pf9_comms_proxy_cfg.py "$PF9_COMMS_PROXY_CONF")"
        if [ -z "$pf9_comms_http_proxy" ]; then
            echo "http proxy: pf9-comms proxy configuration malformed; ignoring"
        fi
    fi

    if [ -n "$pf9_comms_http_proxy" ]; then
        if [ -n "$HTTP_PROXY" ] || [ -n "$http_proxy" ] || [ -n "$HTTPS_PROXY" ] || [ -n "$https_proxy" ]; then
            echo "http proxy: http/s_proxy env vars already defined; overriding with pf9-comms proxy configuration"
        else
            echo "http proxy: http/s_proxy env vars not defined; using pf9-comms proxy configuration"
        fi
        export HTTP_PROXY="$pf9_comms_http_proxy"
        export http_proxy="$pf9_comms_http_proxy"
        export HTTPS_PROXY="$pf9_comms_http_proxy"
        export https_proxy="$pf9_comms_http_proxy"
    else
        if [ -n "$HTTP_PROXY" ] || [ -n "$http_proxy" ] || [ -n "$HTTPS_PROXY" ] || [ -n "$https_proxy" ]; then
            echo "http proxy: http/s_proxy env vars already defined; using unmodified"
        fi
    fi

    add_no_proxy "127.0.0.1"
    add_no_proxy "::1"
    add_no_proxy "localhost"
    if [[ "${CLOUD_PROVIDER_TYPE}" = aws ]]; then
        add_no_proxy "$AWS_METADATA_IP"
    elif [[ "${CLOUD_PROVIDER_TYPE}" = openstack ]]; then
        add_no_proxy "$OPENSTACK_METADATA_IP"
    fi
    add_no_proxy "$(get_node_endpoint)"
    add_no_proxy "${EXTERNAL_DNS_NAME:+$EXTERNAL_DNS_NAME,}${MASTER_IP}"
    add_no_proxy ".${DNS_DOMAIN}"
    add_no_proxy "${CONTAINERS_CIDR}"
    add_no_proxy "${SERVICES_CIDR}"
    add_no_proxy ".svc,.svc.cluster.local"

    echo "http proxy: configuration complete:"

    export pf9_kube_http_proxy_configured="true"
}

function write_cloud_provider_config()
{
    if [[ -z "${KUBELET_CLOUD_CONFIG}" ]]; then
        echo "utils::write_cloud_provider_config: Env var KUBELET_CLOUD_CONFIG is empty, not writing a file"
    else
        echo "utils::write_cloud_provider_config: Writing kubelet cloud-config information for CLOUD_PROVIDER_TYPE: ${CLOUD_PROVIDER_TYPE} to path ${CLOUD_CONFIG_FILE}"
        echo $KUBELET_CLOUD_CONFIG | base64 --decode > ${CLOUD_CONFIG_FILE}
    fi
}

# See - https://platform9.atlassian.net/browse/PMK-1165
function set_iptable_forward_policy_allow()
{
    iptables -P FORWARD ACCEPT
}

#Remove newlines and spaces from sans
#Needed as sans are now passed as a named argument
function trim_sans()
{
    echo $1 | tr -d '\n' | sed 's/\ //g'
}

# first argument is the node name/ip and second argument is name of the annotation
function add_annotation_to_node()
{
    local node_identifier=$1
    local annotation=$2
    if ! err=$(${KUBECTL} annotate --overwrite node ${node_identifier} ${annotation}=true 2>&1 1>/dev/null ); then
            echo "Warning: failed to annotate node ${node_identifier}: ${err}" >&2
    fi
}

# first argument is the node name/ip and second argument is name of the annotation
function remove_annotation_from_node()
{
    local node_identifier=$1
    local annotation=$2
    if ! err=$(${KUBECTL} annotate --overwrite node ${node_identifier} ${annotation}- 2>&1 1>/dev/null ); then
            echo "Warning: failed to remove annotation ${annotation} from node ${node_identifier}: ${err}" >&2
    fi
}
