#!/usr/bin/env bash

source cert_utils.sh
source defaults.env
source runtime.sh

# FIXME: etcdctl currently uses api server's client certs to connect to etcd
# ... may want to generate its own client certs at some point
etcdctl_tls_flags="--cacert /certs/etcdctl/etcd/ca.crt "\
"--cert /certs/etcdctl/etcd/request.crt "\
"--key /certs/etcdctl/etcd/request.key"

etcdctlv2_tls_flags="--ca-file /certs/etcdctl/etcd/ca.crt "\
"--cert-file /certs/etcdctl/etcd/request.crt "\
"--key-file /certs/etcdctl/etcd/request.key"

# FIXME: mount only cert directories required by etcdctl
etcdctl_volume_flags="-v /etc/ssl:/etc/ssl "\
"-v /etc/pki:/etc/pki -v /etc/pf9/kube.d/certs:/certs"

function ensure_etcd_data_stored_on_host()
{
    echo "Ensuring etcd data is stored on host"

    if ! pf9ctr_run \
         inspect etcd >/dev/null; then
        echo "Skipping; etcd container does not exist"
        return
    fi
}

function write_etcd_env()
{
    local etcd_conf_dir=$1
    local node_endpoint=$2

    echo "Deriving local etcd environment"
    echo -e "$ETCD_ENV" > "$etcd_conf_dir/etcd.env"
}

function etcd_running()
{
    [ '$(pf9ctr_run ps | grep -w "etcd" | awk "NR==1{print $1}")' ]
}

# FIXME: client port 4001 is deprecated, retire it in favor of 2379
# FIXME: use https between peers (PMK-20)
function ensure_etcd_running()
{
    local node_endpoint=$1

    mkdir -p "$ETCD_DATA_DIR"
    chmod 0700 "$ETCD_DATA_DIR"

    local etcd_log_level="info"
    if [[ "${DEBUG}" == 'true' ]]; then
        etcd_log_level="debug"
    fi

    # ETCD_LOG_LEVEL: --debug flag and ETCD_DEBUG to be deprecated in v3.5
    # ETCD_LOGGER: default logger capnslog to be deprecated in v3.5, using zap
    # ETCD_ENABLE_V2: Need this for flannel's compatibility with etcd v3.4.14
    local run_opts="--net=host \
        --detach=true \
        --volume /etc/ssl:/etc/ssl \
        --volume /etc/pki:/etc/pki \
        --volume /etc/pf9/kube.d/certs/etcd:/certs/etcd \
        --volume /etc/pf9/kube.d/certs/apiserver:/certs/apiserver \
        --volume ${ETCD_DATA_DIR}:/var/etcd/data \
        -e ETCD_LOG_LEVEL=${etcd_log_level} \
        -e ETCD_LOGGER=zap \
        -e ETCD_ENABLE_V2=true \
        -e ETCD_PEER_CLIENT_CERT_AUTH=true"
    local container_name="etcd"
    local gcr_registry="${GCR_PRIVATE_REGISTRY:-gcr.io}"
    ETCD_CONTAINER_IMG=`echo "${ETCD_CONTAINER_IMG}" | sed "s|gcr.io|${gcr_registry}|g"`
    local container_cmd="/usr/local/bin/etcd"
    local container_cmd_args=""
    if [ -n "${EXTRA_OPT_ETCD_FLAGS}" ] ; then
        container_cmd_args="${container_cmd_args} ${EXTRA_OPT_ETCD_FLAGS}"
    fi

    if [[ ! -e /etc/pki ]]; then
        echo "Creating /etc/pki ."
        mkdir /etc/pki
    fi

    local etcd_conf_dir="/etc/pf9/kube.d/etcd/"
    mkdir -p "$etcd_conf_dir"
    write_etcd_env "$etcd_conf_dir" "$node_endpoint"
    run_opts="${run_opts} --env-file ${etcd_conf_dir}/etcd.env"
    if [ -n "${ETCD_HEARTBEAT_INTERVAL}" ]; then
        run_opts="${run_opts} -e ETCD_HEARTBEAT_INTERVAL=${ETCD_HEARTBEAT_INTERVAL}"
    fi
    if [ -n "${ETCD_ELECTION_TIMEOUT}" ]; then
        run_opts="${run_opts} -e ETCD_ELECTION_TIMEOUT=${ETCD_ELECTION_TIMEOUT}"
    fi
    #
    # TODO
    # PMK-3665: Customise ETCD in platform9 managed kubernetes cluster
    # 
    # The flexibility of customizing ETCD with the help of environment variables
    # needs support from DU side as well if we want it to be truly customizable
    # at the time of cluster creation or at the time of cluster update.
    # For now, we are only checking for the following two environment variables
    # that can be provided via override enironment file at /etc/pf9/kube_override.env
    # on all master nodes. This will make such customizations persistent across node
    # reboots and upgrades.

    # Default snapshot count is set to 100000 from ETCD v3.2 onwards as compared
    # 10000 in earlier versions.
    #
    # If ETCD is getting OOM Killed, this could be one of the possible
    # reasons. ETCD retains all the snapshots in memory so that new nodes joining
    # the ETCD cluster or slow nodes can catch up.
    #
    # Provide an override environment variable in /etc/pf9/kube_override.env
    # set to a lower value exported under name ETCD_SNAPSHOT_COUNT
    # This also results in lower WAL files or write action log files which may
    # consume huge disk space if the snapshot count is high.
    if [ -n "${ETCD_SNAPSHOT_COUNT}" ]; then
        run_opts="${run_opts} -e ETCD_SNAPSHOT_COUNT=${ETCD_SNAPSHOT_COUNT}"
    fi

    # Default max DB size for ETCD is set to 2.1 GB
    # Incase the max DB size is reached, ETCD stops responding to any get/put/watch
    # calls resulting into k8s cluster control plane going down.
    #
    # One of the reasons why this can happen is due to huge amount of older revisions
    # of key values in ETCD database. Although auto compation happens in ETCD every 5 mins,
    # if during these 5 minutes, there are frequent writes happening on the cluster,
    # the revisions pile up during those 5 minutes and even though compaction happens every
    # 5 mins, the space claimed by DB is not released back to the system. In order to release
    # the space, one needs to defrag ETCD manually.
    #
    # If you are expecting intensive writes over a period of 5 mins, it is best to increase
    # the default quota bytes for DB and set it to a higher value, max can be 8GB
    #
    # Provide an override environment variable in /etc/pf9/kube_override.env
    # set to value in bytes, exported under name ETCD_QUOTA_BACKEND_BYTES
    if [ -n "${ETCD_QUOTA_BACKEND_BYTES}" ]; then
        run_opts="${run_opts} -e ETCD_QUOTA_BACKEND_BYTES=${ETCD_QUOTA_BACKEND_BYTES}"
    fi

    # One can control the frequency and extent of compaction using following two environment
    # variables:
    # a) ETCD_AUTO_COMPACTION_MODE
    # b) ETCD_AUTO_COMPACTION_RETENTION
    #
    # ETCD_AUTO_COMPACTION_MODE: which can be set to 'periodic' or 'revision'
    #                            default value is periodic.
    #
    #              periodic can be used if you want to retain key value revisions from the
    #              last time window specified in ETCD_AUTO_COMPACTION_RETENTION env variable.
    #              e.g. 1h or 30m 
    #
    #              revision can be used if you want to retains last n revisions of key values.
    #              You can specify the value in in ETCD_AUTO_COMPACTION_RETENTION env variable.
    if [ -n "${ETCD_AUTO_COMPACTION_MODE}" ]; then
        run_opts="${run_opts} -e ETCD_AUTO_COMPACTION_MODE=${ETCD_AUTO_COMPACTION_MODE}"
    fi
    if [ -n "${ETCD_AUTO_COMPACTION_RETENTION}" ]; then
        run_opts="${run_opts} -e ETCD_AUTO_COMPACTION_RETENTION=${ETCD_AUTO_COMPACTION_RETENTION}"
    fi

    ensure_fresh_container_running $socket "${run_opts}" "${container_name}" "${ETCD_CONTAINER_IMG}" "${container_cmd}" "${container_cmd_args}"

    # Wait for etcd to be up
    # Need to grep the result of 'cluster-health' command because it returns
    # zero even when the local instance is reachable but one or more cluster
    # members is unhealthy. We want to wait for all members to be healthy.

    # IAAS-6826 : Loop waiting for etcd cluster to initialize.
    #             If during this time, etcd exits unexpectedly, then restart it.
    local ok=''
    local retries=${EXTRA_OPT_ETCD_RETRIES:-90}
    for i in `seq 1 $retries`
    do
        sleep 10
        if ! container_running ${socket} etcd ; then
            local timestamp=`date`
            local logfile="/var/log/pf9/kube/etcd-${timestamp}.log"
            pf9ctr_run logs etcd > "${logfile}" 2>&1
            echo 'Restarting failed etcd'
            # Post host name to slack channel #iaas-6826
            # FIXME: remove or make dev-only before GA (IAAS-6849)
            if [ -n "${EXTRA_OPT_SLACK_DEBUG_URL}" ]; then
                local data=`base64 -w 0 "${logfile}"`
                local du_fqdn=`grep host= /etc/pf9/hostagent.conf | grep -v localhost | cut -d= -f2`
                curl -sX POST -d "payload={\"username\":\"${du_fqdn}-`hostname`\", \"text\":\"${data}\"}" \
                    ${EXTRA_OPT_SLACK_DEBUG_URL}
                echo posted log file to slack
            fi
            ensure_fresh_container_running $socket "${run_opts}" "${container_name}" "${ETCD_CONTAINER_IMG}" "${container_cmd}" "${container_cmd_args}"
            continue
        fi
        if pf9ctr_run \
              run ${etcdctl_volume_flags} \
              --rm --net=host ${ETCD_CONTAINER_IMG} \
              etcdctl --endpoints 'https://localhost:4001' ${etcdctl_tls_flags} endpoint health ; then
            ok=yes
            break;
        fi
        echo "Waiting for healthy etcd cluster."
    done

    if [ -z "${ok}" ] ; then
        echo 'timed out waiting for etcd initialization'
        timestamp=`date`
        pf9ctr_run logs etcd > "/var/log/pf9/kube/etcd-$timestamp.log" 2>&1
        return 1
    fi
}

# write the latest etcd version from ETCD_VERSION env variable
# that is pushed in defaults.env
function write_etcd_version_to_file()
{
    local ETCDVERSION_FILE=/var/opt/pf9/etcd_version
    echo "Writing etcd version: ${ETCD_VERSION} in ${ETCDVERSION_FILE} file"
    echo "${ETCD_VERSION}" > ${ETCDVERSION_FILE}
}

function ensure_etcd_destroyed()
{
    ensure_container_destroyed $socket etcd
}

function ensure_role_binding()
{
    ${KUBECTL} version
    local role_binding="${CONF_DST_DIR}/rolebindings/"
    # Delete and re-create the specified resource, when PATCH encounters conflict and has retried for 5 times.
    ${KUBECTL} apply --force -f $role_binding
}


function ensure_dashboard_secret()
{
    ${KUBECTL_SYSTEM} create ns kubernetes-dashboard \
    --dry-run -o yaml | ${KUBECTL_SYSTEM} apply -f -

    # Add the custom TLS configuration to the "kubernetes-dashboard-certs",
    # which is expected to be present by the kubernetes-dashboard deployment.
    ${KUBECTL_SYSTEM} create secret generic kubernetes-dashboard-certs \
      --namespace kubernetes-dashboard \
      --from-file=dashboard.crt="${CONF_DST_DIR}/certs/dashboard/request.crt" \
      --from-file=dashboard.key="${CONF_DST_DIR}/certs/dashboard/request.key" \
      --dry-run -o yaml | ${KUBECTL_SYSTEM} apply -f -
}

function ensure_appcatalog()
{
    local appcatalog="${CONF_DST_DIR}/appcatalog"
    ${KUBECTL} apply -R -f ${appcatalog}
}

function kustomize_config()
{
    # Create a patch file for persisting custom flags of containers
    # running inside master pods if it doesn't exist
    if [[ ! -f /opt/pf9/.custom_api_args.yaml ]]; then
        touch /opt/pf9/.custom_api_args.yaml
    fi

    echo "---" > /opt/pf9/.custom_api_args.yaml
    echo "# For flags in kube-controller-manager" >> /opt/pf9/.custom_api_args.yaml
    echo "#- op: add" >> /opt/pf9/.custom_api_args.yaml
    echo "#  path: /spec/containers/0/command/-" >> /opt/pf9/.custom_api_args.yaml
    echo "#  value: \"--my-custom-controller-arg1=arg1\"" >> /opt/pf9/.custom_api_args.yaml
    echo "# For flags in kube-apiserver" >> /opt/pf9/.custom_api_args.yaml
    echo "#- op: add" >> /opt/pf9/.custom_api_args.yaml
    echo "#  path: /spec/containers/1/command/-" >> /opt/pf9/.custom_api_args.yaml
    echo "#  value: \"--my-custom-apiserver-arg2=arg2\"" >> /opt/pf9/.custom_api_args.yaml
    echo "# For flags in kube-scheduler" >> /opt/pf9/.custom_api_args.yaml
    echo "#- op: add" >> /opt/pf9/.custom_api_args.yaml
    echo "#  path: /spec/containers/2/command/-" >> /opt/pf9/.custom_api_args.yaml
    echo "#  value: \"--my-custom-scheduler-arg3=arg3\"" >> /opt/pf9/.custom_api_args.yaml

    local controller_manager_flags=$(echo $CONTROLLER_MANAGER_FLAGS | sed "s/,\-\-/ --$1/g" | sed 's/, / /g' | sed 's/,*$//g')
    for controller_manager_flag in $controller_manager_flags; do
        echo "- op: add" >> /opt/pf9/.custom_api_args.yaml
        echo "  path: /spec/containers/0/command/-" >> /opt/pf9/.custom_api_args.yaml
        echo "  value: \"$controller_manager_flag\"" >> /opt/pf9/.custom_api_args.yaml
    done

    # Include NodeRestriction by default
    # https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#noderestriction
    if [[ $API_SERVER_FLAGS != *"--enable-admission-plugins"* ]]; then
        API_SERVER_FLAGS="${API_SERVER_FLAGS},--enable-admission-plugins=NodeRestriction"
    fi
    local api_server_flags=$(echo $API_SERVER_FLAGS | sed "s/,\-\-/ --$1/g" | sed 's/, / /g' | sed 's/,*$//g')
    for api_server_flag in $api_server_flags; do
        if [[ ( $api_server_flag == *"--enable-admission-plugins"* ) && ( $api_server_flag != *"NodeRestriction"* ) ]]; then
            api_server_flag="${api_server_flag},NodeRestriction"
        fi
        echo "- op: add" >> /opt/pf9/.custom_api_args.yaml
        echo "  path: /spec/containers/1/command/-" >> /opt/pf9/.custom_api_args.yaml
        echo "  value: \"$api_server_flag\"" >> /opt/pf9/.custom_api_args.yaml
    done

    local scheduler_flags=$(echo $SCHEDULER_FLAGS | sed "s/,\-\-/ --$1/g" | sed 's/, / /g' | sed 's/,*$//g')
    for scheduler_flag in $scheduler_flags; do
        echo "- op: add" >> /opt/pf9/.custom_api_args.yaml
        echo "  path: /spec/containers/2/command/-" >> /opt/pf9/.custom_api_args.yaml
        echo "  value: \"$scheduler_flag\"" >> /opt/pf9/.custom_api_args.yaml
    done

    # Update the cloud provider specific kustomize script to point to the right OS specific base file
    # File to patch ${CONF_SRC_DIR}/masterconfig/overlays/${CLOUD_PROVIDER_TYPE}/kustomization.yaml
    local kustomization_to_patch="${CONF_SRC_DIR}/masterconfig/overlays/${CLOUD_PROVIDER_TYPE}/kustomization.yaml"
    sed -i "s|__OS_FAMILY__|${OS_FAMILY}|g" $kustomization_to_patch

    # PMK-1793 : Replacing sed scripts which existed in a futuristic dystopian universe
    #            where the space time fabric was split and kustomize was never invented!
    ${KUSTOMIZE_BIN} build --load_restrictor none "${CONF_SRC_DIR}/masterconfig/overlays/${CLOUD_PROVIDER_TYPE}" -o "${CONF_SRC_DIR}/master.yaml"
}

# prepare_conf_files processes the configuration files from $CONF_SRC_DIR
# (./conf) and copies them to the $CONF_DST_DIR, keeping the same relative paths.
function prepare_conf_files()
{
    local master_pod="master.yaml"
    local kube_scheduler="configs/kube-scheduler.yaml"
    local dashboard="addons/dashboard"
    local monocular_api_cm=appcatalog/monocular/api/monocular-api-cm.yaml
    local monocular_api_deploy=appcatalog/monocular/api/monocular-api-deploy.yaml
    local monocular_api_svc=appcatalog/monocular/api/monocular-api-svc.yaml
    local tiller_deploy=appcatalog/tiller/tiller-deploy.yaml
    local tiller_svc=appcatalog/tiller/tiller-svc.yaml
    local toleration_patch="toleration_patch.yaml"

    # pf9-sentry files
    local pf9_sentry_namespace="addons/pf9-sentry/pf9-sentry-namespace.yaml"
    local pf9_sentry_serviceaccount="addons/pf9-sentry/pf9-sentry-serviceaccount.yaml"
    local pf9_sentry_clusterrole="addons/pf9-sentry/pf9-sentry-clusterrole.yaml"
    local pf9_sentry_clusterrolebinding="addons/pf9-sentry/pf9-sentry-clusterrolebinding.yaml"
    local pf9_sentry_deployment="addons/pf9-sentry/pf9-sentry-deployment.yaml"
    local pf9_sentry_service="addons/pf9-sentry/pf9-sentry-service.yaml"

    # pf9-addon-operator files
    local pf9_addon_operator_crd="addons/pf9-addon-operator/pf9-addon-operator-crd.yaml"
    local pf9_addon_operator_namespace="addons/pf9-addon-operator/pf9-addon-operator-namespace.yaml"
    local pf9_addon_operator_rbac="addons/pf9-addon-operator/pf9-addon-operator-rbac.yaml"
    local pf9_addon_operator_deployment="addons/pf9-addon-operator/pf9-addon-operator-deployment.yaml"

    # Image registries
    local quay_registry="${QUAY_PRIVATE_REGISTRY:-quay.io}"
    local k8s_registry="${K8S_PRIVATE_REGISTRY:-k8s.gcr.io}"
    local gcr_registry="${GCR_PRIVATE_REGISTRY:-gcr.io}"
    local docker_registry="${DOCKER_PRIVATE_REGISTRY}"

    # If docker_registry is empty, we need to also remove the leading `/` in the image URL.
    # Otherwise, the URL will look like `/platform9/pf9-sentry:1.0.0`, which is invalid.
    local docker_registry_filter="__DOCKER_REGISTRY__"
    if [[ -z ${docker_registry} ]]; then
        docker_registry_filter="__DOCKER_REGISTRY__\/"
    fi

    # Configure image registries
    find ${CONF_SRC_DIR} -name "*.yaml" -print0 | while IFS= read -r -d '' file; do
        sed -i \
        -e "s|__QUAY_REGISTRY__|${quay_registry}|g" \
        -e "s|__K8S_REGISTRY__|${k8s_registry}|g" \
        -e "s|__GCR_REGISTRY__|${gcr_registry}|g" \
        -e "s|${docker_registry_filter}|${docker_registry}|g" \
        ${file}
    done


    #                 file        substitution func
    prepare_conf_file $master_pod prepare_master_pod

    prepare_conf_file $kube_scheduler
    prepare_conf_file $pf9_sentry_namespace
    prepare_conf_file $pf9_sentry_serviceaccount
    prepare_conf_file $pf9_sentry_clusterrole
    prepare_conf_file $pf9_sentry_clusterrolebinding
    prepare_conf_file $pf9_sentry_deployment
    prepare_conf_file $pf9_sentry_service

    prepare_conf_file $pf9_addon_operator_crd
    prepare_conf_file $pf9_addon_operator_namespace
    prepare_conf_file $pf9_addon_operator_rbac
    prepare_conf_file $pf9_addon_operator_deployment

    prepare_conf_file $monocular_api_cm
    prepare_conf_file $monocular_api_deploy
    prepare_conf_file $monocular_api_svc
    prepare_conf_file $tiller_deploy
    prepare_conf_file $tiller_svc
    prepare_conf_file $toleration_patch

    local authn_webhook_kubeconfig="authn/webhook-config.yaml"
    env_vars_from_cert AUTHN_WEBHOOK authn_webhook
    prepare_conf_file $authn_webhook_kubeconfig prepare_authn_webhook_kubeconfig
}

function prepare_kubeconfigs_master_only()
{
    # generate kubeconfigs for kube_scheduler and kube_controller_manager only on master
    local kube_controller_manager="kubeconfigs/kube-controller-manager.yaml"
    local kube_scheduler="kubeconfigs/kube-scheduler.yaml"

    #                  var prefix               certs_dir
    env_vars_from_cert KUBE_CONTROLLER_MANAGER  kube-controller-manager/apiserver
    env_vars_from_cert KUBE_SCHEDULER           kube-scheduler/apiserver

    apiserver_host="localhost:"${K8S_API_PORT}

    #                 file                     substitution command
    prepare_conf_file $kube_controller_manager "prepare_kube_controller_manager_kubeconfig ${apiserver_host}"
    prepare_conf_file $kube_scheduler          "prepare_kube_scheduler_kubeconfig ${apiserver_host}"
}

function prepare_authn_webhook_kubeconfig()
{
    sed -e "s|__AUTHN_WEBHOOK_URL__|https://${AUTHN_WEBHOOK_ADDR}/v1|g" \
        -e "s/__AUTHN_WEBHOOK_CA_CERT_BASE64__/${AUTHN_WEBHOOK_CA_CERT_BASE64}/g"
}

function prepare_master_pod()
{
    # https://kubernetes.io/docs/reference/access-authn-authz/node/
    AUTHZ_MODE="RBAC,Node"
    if [ $KEYSTONE_ENABLED == 'true' ]; then
        # RUNTIME_CONFIG is comma-separated list of key=value pairs. It is
        # defined in the pf9-kube role configuration. Here, we prepend the
        # runtime configuration required by the token webhook authenticator.
        if [ -z "$RUNTIME_CONFIG" ]; then
            RUNTIME_CONFIG="authentication.k8s.io/v1beta1=true"
        else
            RUNTIME_CONFIG="authentication.k8s.io/v1beta1=true,${RUNTIME_CONFIG}"
        fi
        AUTHN_WEBHOOK_CACHE_TTL="0s"
        AUTHN_WEBHOOK_CONFIG_FILE="/srv/kubernetes/authn/webhook-config.yaml"
    else
        AUTHN_WEBHOOK_CACHE_TTL=""
        AUTHN_WEBHOOK_CONFIG_FILE=""
    fi
    if [ -z "$RUNTIME_CONFIG" ]; then
        RUNTIME_CONFIG="scheduling.k8s.io/v1alpha1=true"
    else
        RUNTIME_CONFIG="scheduling.k8s.io/v1alpha1=true,${RUNTIME_CONFIG}"
    fi

    CLOUD_PROVIDER="$CLOUD_PROVIDER_TYPE"
    if [ "$CLOUD_PROVIDER_TYPE" == "local" ]; then
        CLOUD_PROVIDER=""
    fi

    if [[ "$CLOUD_PROVIDER_TYPE" == "openstack" || "$CLOUD_PROVIDER_TYPE" == "azure" ]]; then
        CLOUD_CFG_FILE="/srv/kubernetes/cloud-config"
    else
        CLOUD_CFG_FILE=""
    fi

    ALLOCATE_NODE_CIDRS=""
    CLUSTER_CIDR=""

    if [ "${PF9_NETWORK_PLUGIN}" == "canal" ] || [ "${PF9_NETWORK_PLUGIN}" == "calico" ]; then
        ALLOCATE_NODE_CIDRS="true"
        CLUSTER_CIDR="${CONTAINERS_CIDR}"
    fi

    DEBUG_LEVEL=2

    if [ "$DEBUG" == "true" ]; then
        DEBUG_LEVEL=8
    fi

    sed -e "s/__KUBERNETES_VERSION__/${KUBERNETES_VERSION}/g" \
        -e "s|__SERVICES_CIDR__|${SERVICES_CIDR}|g" \
        -e "s|__PRIVILEGED__|${PRIVILEGED}|g" \
        -e "s|__AUTHZ_MODE__|${AUTHZ_MODE}|g" \
        -e "$(sub_or_delete_line __CLOUD_PROVIDER__ ${CLOUD_PROVIDER})" \
        -e "$(sub_or_delete_line __CLOUD_CONFIG__ ${CLOUD_CFG_FILE})" \
        -e "$(sub_or_delete_line __RUNTIME_CONFIG__ ${RUNTIME_CONFIG})" \
        -e "$(sub_or_delete_line __AUTHN_WEBHOOK_CACHE_TTL__ ${AUTHN_WEBHOOK_CACHE_TTL})" \
        -e "$(sub_or_delete_line __AUTHN_WEBHOOK_CONFIG_FILE__ ${AUTHN_WEBHOOK_CONFIG_FILE})" \
        -e "$(sub_or_delete_line __ALLOCATE_NODE_CIDRS__ ${ALLOCATE_NODE_CIDRS})" \
        -e "$(sub_or_delete_line __CLUSTER_CIDR__ ${CLUSTER_CIDR})" \
        -e "s|__HTTP_PROXY__|${HTTP_PROXY}|g" \
        -e "s|__HTTPS_PROXY__|${HTTPS_PROXY}|g" \
        -e "s|__NO_PROXY__|${NO_PROXY}|g" \
        -e "s|__http_proxy__|${http_proxy}|g" \
        -e "s|__https_proxy__|${https_proxy}|g" \
        -e "s|__no_proxy__|${no_proxy}|g" \
        -e "s|__APISERVER_STORAGE_BACKEND__|${APISERVER_STORAGE_BACKEND}|g" \
        -e "s|__K8S_API_PORT__|${K8S_API_PORT}|g" \
        -e "s|__DEBUG_LEVEL__|${DEBUG_LEVEL}|g"
}

function prepare_metrics_api()
{
    sed -e "s/--deprecated-kubelet-completely-insecure.*$/--tls-cipher-suites=TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256/" \
        -e "/--kubelet-port/d"

}

function populate_cert_command_map_master()
{
    # the client cert etcdctl uses to talk to etcd
    cert_path_to_params_map["etcdctl/etcd"]="--cn=etcdctl --cert_type=client"
    # the server cert etcd presents to clients
    cert_path_to_params_map["etcd/client"]="--cn=etcd --cert_type=server --sans=${trimmed_etcd_sans} "
    # the server cert that etcd presents to peers
    cert_path_to_params_map["etcd/peer"]="--cn=etcd --cert_type=peer --sans=${trimmed_etcd_sans} "
    # the client cert flannel uses to talk to etcd
    cert_path_to_params_map["flannel/etcd"]="--cn=flannel --cert_type=client"
    # the server cert kube-apiserver presents to clients
    cert_path_to_params_map["apiserver"]="--cn=apiserver --cert_type=server --sans=${trimmed_apiserver_sans} --needs_svcacctkey=true "
    # the client cert kube-apiserver uses to talk to etcd
    cert_path_to_params_map["apiserver/etcd"]="--cn=apiserver --cert_type=client"
    # the client cert kube-apiserver uses to talk to etcd
    cert_path_to_params_map["kubelet/apiserver"]="--cn=kubelet --cert_type=client"
    # the server cert kubelet presents to kube-apiserver (not used currently)
    cert_path_to_params_map["kubelet/server"]="--cn=kubelet --cert_type=server --sans=${trimmed_kubelet_sans} "
    # the client cert kube-proxy uses to talk to kube-apiserver
    cert_path_to_params_map["kube-proxy/apiserver"]="--cn=kube-proxy --cert_type=client"
    # the client cert kube-controller-manager uses to talk to kube-apiserver
    cert_path_to_params_map["kube-controller-manager/apiserver"]="--cn=system:kube-controller-manager --cert_type=client --org=system:kube-controller-manager"
    # the client cert kube-scheduler uses to talk to kube-apiserver
    cert_path_to_params_map["kube-scheduler/apiserver"]="--cn=system:kube-scheduler --cert_type=client --org=system:kube-scheduler"
    # the client cert the admin user uses to talk to kube-apiserver
    cert_path_to_params_map["admin"]="--cn=admin --cert_type=client --org=system:masters"
    # the server cert the authn webhook presents to kube-apiserver
    cert_path_to_params_map["authn_webhook"]="--cn=admin --cert_type=server --sans=${trimmed_auth_webhook}"
    # the server cert the kubernetes dashboard presents to users
    cert_path_to_params_map["dashboard"]="--cn=dashboard  --cert_type=server --sans=${trimmed_dashboard_sans}"
    # the client cert calico uses to talk to kube-apiserver
    cert_path_to_params_map["calico/etcd"]="--cn=calico --cert_type=client"
    # the client cert the kube api aggregator uses to talk to kube-apiserver
    cert_path_to_params_map["aggregator"]="--cn=aggregator --cert_type=client"


}
# Request signed certs concurrently, then wait for all requests to be
# fulfilled
function prepare_certs()
{
    local node_name=$1
    local node_name_type=$2
    local node_ip=$3
    ensure_certs_dir_backedup "startup"

    local apiserver_sans="\
        IP:${API_SERVICE_IP}, \
        $(master_ip_type):${MASTER_IP}, \
        IP:127.0.0.1, \
        IP:::1, \
        ${node_name_type}:${node_name}, \
        DNS:${MASTER_NAME}, \
        DNS:localhost, \
        DNS:kubernetes, \
        DNS:kubernetes.default, \
        DNS:kubernetes.default.svc, \
        DNS:kubernetes.default.svc.${DNS_DOMAIN}"

    local dashboard_sans="\
        DNS:kubernetes-dashboard, \
        DNS:kubernetes-dashboard.default, \
        DNS:kubernetes-dashboard.default.svc, \
        DNS:kubernetes-dashboard.default.svc.${DNS_DOMAIN}"

    # DNS/IP:MASTER_IP SANs is necessary if Calico networking is used
    local etcd_sans="\
        IP:127.0.0.1, \
        IP:::1, \
        IP:${node_ip}, \
        DNS:localhost, \
        $(master_ip_type):${MASTER_IP},\
        ${node_name_type}:${node_name}"

    local kubelet_sans="\
        IP:${node_ip}, \
        ${node_name_type}:${node_name}"

    if [ -n "$EXTERNAL_DNS_NAME" ]; then
        apiserver_sans="${apiserver_sans}, DNS:${EXTERNAL_DNS_NAME}"
        etcd_sans="${etcd_sans}, DNS:${EXTERNAL_DNS_NAME}"
        # the apiserver could be running inside of a kubernetes namespace
        # (for e.g. in the masterless case), in which case its service is
        # addressable via the cluster short name (which automatically gets
        # appended with the full namespace-specific domain name via resolv.conf)
        # by clients running inside of the cluster.
        clustname=$(echo ${EXTERNAL_DNS_NAME} | cut -d'.' -f1)
        apiserver_sans="${apiserver_sans}, DNS:${clustname}"
    fi

    if [ -n "$PUBLIC_IP" ]; then
        apiserver_sans="${apiserver_sans}, IP:${PUBLIC_IP}"
    fi

    if [ "${MASTER_VIP_ENABLED}" == "true" ]; then
        apiserver_sans="${apiserver_sans}, IP:${MASTER_IP}"
        etcd_sans="${etcd_sans}, IP:${MASTER_IP}"
    fi

    trimmed_etcd_sans=$(trim_sans "$etcd_sans")
    trimmed_apiserver_sans=$(trim_sans "$apiserver_sans")
    trimmed_kubelet_sans=$(trim_sans "$kubelet_sans")
    trimmed_auth_webhook=$(trim_sans "$AUTHN_WEBHOOK_SANS")
    trimmed_dashboard_sans=$(trim_sans "$dashboard_sans")

    init_pki

    (
        tmp_dir=$(mktemp -d --tmpdir=/tmp authbs-certs.XXXX)

        populate_cert_command_map_master

        if run_certs_requests; then
            echo 'All Certs generated successfully'
            return;
        else
            echo "Failed to generate certificates even after ${MAX_CERTS_RETRIES} retries"
            return 1
        fi
    )
}

function master_ip_type()
{
    ./ip_type ${MASTER_IP}
}

#
# Enter the container and modify its /etc/hosts to map
# $KEYSTONE_DOMAIN to localhost.  comms and switcher then route this
# to keystone on the DU.
#
# Modifying /etc/hosts in a container is not in general safe.
# However, in this case, since we set hostNetwork=True in the master
# pod spec, docker and kubernetes leave /etc/hosts unmodified for the
# containers in the master pod.  If we find this to be brittle, we
# could try using dnsmasq instead.
#
function ensure_keystone_dns_mapped()
{
    if ! keystone_dns_mapped; then
        map_keystone_dns
    fi
}

function keystone_dns_mapped()
{
    local apiserver_docker_id=$(local_apiserver_docker_id)
    pf9ctr_run \
        exec $apiserver_docker_id \
            grep "${KEYSTONE_ETCDHOSTS_ENTRY}" /etc/hosts
    return $?
}

function map_keystone_dns()
{
    local apiserver_docker_id=$(local_apiserver_docker_id)
    # get container to resolve $KEYSTONE_DOMAIN to localhost
    pf9ctr_run \
        exec $apiserver_docker_id \
            sh -c "echo ${KEYSTONE_ETCDHOSTS_ENTRY} >> /etc/hosts"
    echo "Appended ${KEYSTONE_ETCDHOSTS_ENTRY} to /etc/hosts in apiserver (docker id: $apiserver_docker_id)"
}

# Find local apiserver docker id (FIXME IAAS-5946)
function local_apiserver_docker_id()
{
    pf9ctr_run --namespace k8s.io ps | grep -w "kube-apiserver" | awk 'NR==1{print $1}'
}

function local_apiserver_running()
{
    # Attempts to connect to the k8s api server's bind-address on port
    # user defined port. Port 443 by default.
    # from http://stackoverflow.com/a/19866239
    &>/dev/null timeout 1s bash -c "echo "" > /dev/tcp/0.0.0.0/${K8S_API_PORT}"
}

function ensure_authn_webhook_image_available()
{
    load_image_from_file $socket "$AUTHN_WEBHOOK_IMAGE_TARBALL"
}

function ensure_authn_webhook_running()
{
    local run_opts="--net=host \
        --detach=true \
        --volume ${CERTS_DIR}/authn_webhook/:/certs:ro"
    local container_name="$AUTHN_WEBHOOK_CTR_NAME"
    local container_img="$AUTHN_WEBHOOK_IMAGE"
    local container_cmd="/bouncerd"
    local container_cmd_args="--ca-file /certs/ca.crt \
        --cert-file /certs/request.crt \
        --key-file /certs/request.key \
        $AUTHN_WEBHOOK_ADDR \
        $AUTHN_WEBHOOK_KEYSTONE_URL \
        $CLUSTER_PROJECT_ID"

    # Instrumentation for bug INF-764 (slow auth webhook requests)
    if [ -n "${BOUNCER_SLOW_REQUEST_WEBHOOK}" ] ; then
        local du_fqdn=`grep host= /etc/pf9/hostagent.conf | grep -v localhost | cut -d= -f2`
        run_opts="${run_opts} --env BOUNCER_SLOW_REQUEST_WEBHOOK=${BOUNCER_SLOW_REQUEST_WEBHOOK}"
        run_opts="${run_opts} --env DU_FQDN=${du_fqdn}"
        run_opts="${run_opts} --env HOST_NAME=$(hostname)"
        run_opts="${run_opts} --env CLUSTER_ID=${CLUSTER_ID}"
    fi
    ensure_fresh_container_running $socket "${run_opts}" "${container_name}" "${container_img}" "${container_cmd}" "${container_cmd_args}"
}

function ensure_authn_webhook_stopped()
{
    ensure_container_destroyed $socket "$AUTHN_WEBHOOK_CTR_NAME"
}

function authn_webhook_listening()
{
    curl --silent \
        --max-time 5 \
        --cacert "${CERTS_DIR}/authn_webhook/ca.crt" \
        "https://${AUTHN_WEBHOOK_ADDR}/healthz" \
        &>/dev/null
}

function add_tolerations_and_affinity()
{
    local deployment_path=$1
    local patch_file="${CONF_DST_DIR}/toleration_patch.yaml"
    ${KUBECTL_SYSTEM} patch -f ${deployment_path} -p "$(cat ${patch_file})"
}

function taint_node()
{
    local node_name=$1
    wait_until "${KUBECTL} taint node ${node_name} --overwrite node-role.kubernetes.io/master=true:NoSchedule" 5 20
}


# TODO: This function is invoked only by tackboard/configure_metallb.sh. This should be removed
# once we make corresponding changes in qbert update API call. Ideally metallb configmap should
# only be updated by the addon operator. Keeping this function to minimize build failures.
function configure_metallb()
{
    local metallb_config="${CONF_SRC_DIR}/addons/metallb_conf.yaml"
cat <<EOF > ${metallb_config}
apiVersion: v1
kind: ConfigMap
metadata:
  namespace: metallb-system
  name: config
data:
  config: |
    address-pools:
    - name: default
      protocol: layer2
      addresses:
EOF

    # Handle multiple address pools (PMK-1662)
    OLD_IFS=${IFS}
    IFS=',';
    pools=(${METALLB_CIDR})
    for pool in ${pools[@]};
    do
        # xargs removes the leading and trailing white spaces
        pool=`echo ${pool}| xargs`
cat <<EOF >> ${metallb_config}
      - ${pool}
EOF
    done
    IFS=${OLD_IFS}

    ${KUBECTL_SYSTEM} apply -f ${metallb_config}
}

function create_keepalived_healthcheck_script()
{

CURL=/usr/bin/curl
ADMIN_CERTS=${CONF_DST_DIR}/certs/admin

cat <<EOF > ${VRRP_HEALTH_CHECK_SCRIPT}
#!/bin/bash
set -x
${CURL} https://127.0.0.1:${K8S_API_PORT}/healthz --cacert ${ADMIN_CERTS}/ca.crt --key ${ADMIN_CERTS}/request.key --cert ${ADMIN_CERTS}/request.crt --fail > /dev/null 2>&1
EOF

chown -R ${PF9_USER}:${PF9_GROUP} ${VRRP_HEALTH_CHECK_SCRIPT}
chmod +x ${VRRP_HEALTH_CHECK_SCRIPT}

}

function ensure_keepalived_configured()
{

create_keepalived_healthcheck_script

cat <<EOF > ${MASTER_VIP_KEEPALIVED_CONF_FILE}
vrrp_script chk_apiserver {
    script ${VRRP_HEALTH_CHECK_SCRIPT}
    interval ${VRRP_HEALTH_CHECK_INTERVAL}
    fall ${VRRP_HEALTH_CHECK_FALL}
    rise ${VRRP_HEALTH_CHECK_RISE}
    user ${PF9_USER}
}

global_defs {
    enable_script_security
    script_user ${PF9_USER}
    vrrp_garp_master_refresh 10
    vrrp_garp_master_refresh_repeat 2
}

vrrp_instance K8S_APISERVER {
    interface ${MASTER_VIP_IFACE}
    state BACKUP
    virtual_router_id ${MASTER_VIP_VROUTER_ID}
    nopreempt

    authentication {
        auth_type AH
        auth_pass m@st3rv1p
    }

    virtual_ipaddress {
        ${MASTER_IP}
    }
    track_script {
        chk_apiserver
    }

}
EOF

}

function start_keepalived()
{
    # Make sure keepalived is set not to start automaticaly
    systemctl disable keepalived
    systemctl start keepalived
}

function stop_and_remove_keepalived()
{
    systemctl stop keepalived
    rm -f ${MASTER_VIP_KEEPALIVED_CONF_FILE} || echo "Couldn't remove keepalived conf file or file does not exist"
}

function keepalived_running()
{
    systemctl status keepalived
}

function post_upgrade_cleanup()
{
    local heapster="${CONF_DST_DIR}/addons/heapster/heapster.yaml"
    local heapster_rbac="${CONF_DST_DIR}/addons/heapster/heapster_rbac.yaml"

    if heapster_running; then
        if [ -f ${heapster} ]; then
            ${KUBECTL} delete --ignore-not-found -f ${heapster}
        fi
        if [ -f ${heapster_rbac} ]; then
            ${KUBECTL} delete --ignore-not-found -f ${heapster_rbac}
        fi
    fi

    local old_metrics_server_deployment="metrics-server-v0.2.1"
    ${KUBECTL} delete --ignore-not-found deployment/${old_metrics_server_deployment}

}

function post_upgrade_monitoring_fix()
{
    # Check if monitoring has been previously installed, if not return
    local csv=$(${KUBECTL_SYSTEM} get csv prometheusoperator.0.37.0 -n pf9-operators --ignore-not-found -o jsonpath='{.metadata.name}')
    if [[ ! $csv == "prometheusoperator.0.37.0" ]]; then
        echo "Monitoring not installed, no post upgrade fix required"
        return
    else
        echo "Monitoring found installed, fixing ownership of objects in pf9-monitoring"
    fi

    # Need to create configmap because we are upgrading from an older pf9-kube version 
    # where the new owner configmap was not present
    ${KUBECTL_SYSTEM} create configmap monitoring-owner -n pf9-monitoring \
    --dry-run -o yaml | ${KUBECTL_SYSTEM} apply -f -

    # Patch ownership of all objects deployed by monhelper
    # Without this objects don't come up in pf9-monitoring ns, see PMK-3663
    # Change ownership by patching the ClusterServerVersion with new monhelper image
    local csv_patch="${CONF_SRC_DIR}/csv_patch.yaml"
    ${KUBECTL_SYSTEM} -n pf9-operators patch csv prometheusoperator.0.37.0 --type merge --patch "$(cat ${csv_patch})"
}

function heapster_running()
{
    local heapster_deployment="heapster-v1.5.0"
    ${KUBECTL} get deployment -n kube-system ${heapster_deployment} &> /dev/null
}

function ensure_pf9_sentry()
{
    local pf9_sentry="${CONF_DST_DIR}/addons/pf9-sentry"
    ${KUBECTL_SYSTEM} apply -f "${pf9_sentry}/pf9-sentry-namespace.yaml"
    ${KUBECTL_SYSTEM} apply -f "${pf9_sentry}/pf9-sentry-serviceaccount.yaml"
    ${KUBECTL_SYSTEM} apply -f "${pf9_sentry}/pf9-sentry-clusterrole.yaml"
    ${KUBECTL_SYSTEM} apply -f "${pf9_sentry}/pf9-sentry-clusterrolebinding.yaml"
    ${KUBECTL_SYSTEM} apply -f "${pf9_sentry}/pf9-sentry-deployment.yaml"
    ${KUBECTL_SYSTEM} apply -f "${pf9_sentry}/pf9-sentry-service.yaml"
}

function delete_cluster_autoscaler_post_upgrade()
{
    if [[ "$CLOUD_PROVIDER_TYPE" == 'aws' ]]; then
        ${KUBECTL_SYSTEM} -n pf9-addons delete addon ${CLUSTER_ID}-cluster-auto-scaler-aws --ignore-not-found
    elif [[ "$CLOUD_PROVIDER_TYPE" == 'azure' ]]; then
        ${KUBECTL_SYSTEM} -n pf9-addons delete addon ${CLUSTER_ID}-cluster-auto-scaler-azure --ignore-not-found
    fi
}

function ensure_pf9_addon_operator()
{
    ensure_dashboard_secret

    local pf9_addon_operator="${CONF_DST_DIR}/addons/pf9-addon-operator"
    ${KUBECTL_SYSTEM} apply -f "${pf9_addon_operator}/pf9-addon-operator-crd.yaml"
    ${KUBECTL_SYSTEM} apply -f "${pf9_addon_operator}/pf9-addon-operator-namespace.yaml"
    ${KUBECTL_SYSTEM} apply -f "${pf9_addon_operator}/pf9-addon-operator-rbac.yaml"
    local addon_operator="${pf9_addon_operator}/pf9-addon-operator-deployment.yaml"
    local du_fqdn=`grep host= /etc/pf9/hostagent.conf | grep -v localhost | cut -d= -f2`

    cat $addon_operator | sed -e "s/__CLUSTER_ID__/${CLUSTER_ID}/g"  \
    | sed -e "s/__PROJECT_ID__/${CLUSTER_PROJECT_ID}/" \
    | sed -e "s/__DU_FQDN__/${du_fqdn}/" \
    | sed -e "s/__CLOUD_PROVIDER_TYPE__/${CLOUD_PROVIDER_TYPE}/" \
    | sed -e "s/__QUAY_REGISTRY__/${QUAY_PRIVATE_REGISTRY}/" \
    | sed -e "s/__K8S_REGISTRY__/${K8S_PRIVATE_REGISTRY}/" \
    | sed -e "s/__GCR_REGISTRY__/${GCR_PRIVATE_REGISTRY}/" \
    | sed -e "s/__DOCKER_REGISTRY__/${DOCKER_PRIVATE_REGISTRY}/" \
    | sed -e "s#__HTTP_PROXY__#${HTTP_PROXY}#" \
    | sed -e "s#__HTTPS_PROXY__#${HTTPS_PROXY}#" \
    | sed -e "s#__NO_PROXY__#${NO_PROXY}#" \
    | ${KUBECTL_SYSTEM} apply -f -

    ensure_addon_secret
}


function ensure_addon_secret()
{
    LITERALS="--from-literal=dnsIP=${DNS_IP} "

    if [[ "$CLOUD_PROVIDER_TYPE" == 'azure' ]]; then
        local client_id=`cat /etc/pf9/kube.d/cloud-config| /opt/pf9/pf9-kube/bin/jq -r '.aadClientID'`
        local trimmed_client_id=$(trim_sans "$client_id" | base64)
        local client_secret=`cat /etc/pf9/kube.d/cloud-config| /opt/pf9/pf9-kube/bin/jq -r '.aadClientSecret'`
        local trimmed_client_secret=$(trim_sans "$client_secret" | base64)
        local resource_group=`cat /etc/pf9/kube.d/cloud-config| /opt/pf9/pf9-kube/bin/jq -r '.resourceGroup'`
        local trimmed_resource_group=$(trim_sans "$resource_group" | base64)
        local subscription_id=`cat /etc/pf9/kube.d/cloud-config| /opt/pf9/pf9-kube/bin/jq -r '.subscriptionId'`
        local trimmed_subscription_id=$(trim_sans "$subscription_id" | base64)
        local tenant_id=`cat /etc/pf9/kube.d/cloud-config| /opt/pf9/pf9-kube/bin/jq -r 'tenantID'`
        local trimmed_tenant_id=$(trim_sans "$tenant_id" | base64)

        LITERALS=$LITERALS" --from-literal=clientID=${trimmed_client_id} \
        --from-literal=clientSecret=${trimmed_client_secret} \
        --from-literal=resourceGroup=${trimmed_resource_group} \
        --from-literal=subscriptionID=${trimmed_subscription_id} \
        --from-literal=tenantID=${trimmed_tenant_id} "
    fi

    ${KUBECTL_SYSTEM} -n pf9-addons create secret generic addon-config ${LITERALS} \
    --dry-run -o yaml | ${KUBECTL_SYSTEM} apply -f -
}

function is_eligible_for_etcd_backup()
{
    local exitCode=0
    local ETCDVERSION_FILE=/var/opt/pf9/etcd_version
    if [ ! -f ${ETCDVERSION_FILE} ]; then
        # writing etcd version to a file during start_master instead of stop_master,
        # During the upgrade new package sequence is : status --> stop --> start
        # Due to above, cannot rely on writing version during stop, as that will lead
        # to false assumption.
        # With this, backup and raft check shall happen once during both fresh install
        # and upgrade
        write_etcd_version_to_file >/dev/null 2>&1
        return ${exitCode}
    else
        local OLD_ETCD_VERSION=$(<"${ETCDVERSION_FILE}")
        # no backup done if etcd version are the same
        if [ ${OLD_ETCD_VERSION} == ${ETCD_VERSION} ]; then
            exitCode=1
        else
            # when etcd version is a mismatch, that indicates upgrade
            # perform backup and raft check and update the etcd version to most recent
            write_etcd_version_to_file >/dev/null 2>&1
        fi
    fi
    return ${exitCode}
}

function ensure_etcd_data_backup()
{
    local exitCode=0

    # etcdctl v3
    ETCDCTL_BIN=/opt/pf9/pf9-kube/bin/etcdctl
    local ETCD_BACKUP_DIR=/var/opt/pf9/kube/etcd/etcd-backup
    local ETCD_BACKUP_LOC=${ETCD_BACKUP_DIR}/etcdv3_backup.db

    # || rc=$? is used to prevent exit due to errexit shopt
    # checks if data directory present
    if [ -d ${ETCD_DATA_DIR}/member ]; then

        # etcd v3 data
        if [ -d "${ETCD_BACKUP_DIR}" ]; then
            echo "${ETCD_BACKUP_DIR} already present"
        else
            echo "creating ${ETCD_BACKUP_DIR}"
            mkdir -p ${ETCD_BACKUP_DIR}
        fi

        if [ -f ${ETCD_BACKUP_LOC} ]; then
            echo "cleaning existing etcdv3 backup and taking a new backup"
            rm -rf ${ETCD_BACKUP_LOC}
        fi

        cp -aR ${ETCD_DATA_DIR}/member/snap/db ${ETCD_BACKUP_LOC} && rc=$? || rc=$?
        if [ $rc -ne 0 ]; then
            echo "etcdv3 backup failed"
            exitCode=1
        else
            echo "etcdv3 backup success"
        fi
    else
        echo "etcd ${ETCD_DATA_DIR} directory not found. skipping etcd data backup"
    fi

    return ${exitCode}
}


function ensure_etcd_cluster_status()
{
    local exitCode=0
    local ETCD_RAFT_CHECKER=/opt/pf9/pf9-kube/bin/etcd_raft_checker

    if [ -f "${ETCD_RAFT_CHECKER}" ]; then
        ${ETCD_RAFT_CHECKER} && rc=$? || rc=$?
        if [ $rc -ne 0 ]; then
            echo "etcd cluster status not ok"
            exitCode=1
        else
            echo "etcd cluster status ok"
        fi
    else
        # continue with the rest of the upgrade even if binary is missing.
        # flag it as ERROR in log. Failing here will be premature with a benefit
        # of doubt that etcd cluster still can be healthy
        echo "ERROR: etcd-raft-checker binary missing; could not check raft indices"
    fi
    return ${exitCode}
}

function ensure_dns()
{
    local coredns_template="${CONF_SRC_DIR}/networkapps/coredns.yaml"
    local coredns_file="${CONF_SRC_DIR}/networkapps/coredns-applied.yaml"
    local k8s_registry="${K8S_PRIVATE_REGISTRY:-k8s.gcr.io}"

    # Replace configuration values in calico spec with user input
    sed -e "s|__DNS_IP__|${DNS_IP}|g" \
        -e "s|__K8S_REGISTRY__|${k8s_registry}|g" \
        < ${coredns_template} > ${coredns_file}
    # Apply daemon set yaml
    ${KUBECTL_SYSTEM} apply -f ${coredns_file}
}
