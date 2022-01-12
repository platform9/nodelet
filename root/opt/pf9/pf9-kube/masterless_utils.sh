#!/usr/bin/env bash
source defaults.env
source runtime.sh

function get_apiserver_endpoint_ip()
{
    ${KUBECTL} get -o jsonpath='{.subsets[0].addresses[0].ip}' ep kubernetes
}

function setup_veth()
{
    local ipaddr=$1
    ip link delete pa-proxy-veth0 || true
    ip link add pa-proxy-veth0 type veth peer name pa-proxy-veth1 || return 1
    ip addr add "${ipaddr}/32" dev pa-proxy-veth0 || return 1
    ip link set pa-proxy-veth0 up
}

function teardown_veth()
{
    ip link set pa-proxy-veth0 down || true
    ip link delete pa-proxy-veth0 || true
}

function start_pod_to_apiserver_proxy()
{
    if ! wait_until "ip=$(get_apiserver_endpoint_ip)" 5 5 ; then
        echo "failed to get apiserver endpoint ip"
        return 1
    fi
    echo "apiserver internal endoint ip is ${ip}"

    # FIXME: KPLAN-72: detect whether this apiserver IP address conflicts
    # with another interface or overlaps a subnet accessible by the host.
    if ! setup_veth "${ip}"; then
        echo "failed to set up pa-proxy-veth0"
        return 1
    fi
    local run_opts="-d --net host"
    local container_name="pa-proxy"

    # If DOCKER_PRIVATE_REGISTRY is empty, we need to also remove the leading `/` in the image URL.
    # Otherwise, the URL will look like `/platform9/pa-haproxy:latest`, which is invalid.
    # FIXME: KPLAN-73: use a versioned tag instead of :latest
    #                  or package and bundle it like bouncer
    local container_img="${DOCKER_PRIVATE_REGISTRY}/platform9/pa-proxy:latest"
    if [[ -z ${DOCKER_PRIVATE_REGISTRY} ]]; then
        container_img="platform9/pa-proxy:latest"
    fi

    local container_cmd=""
    local container_cmd_args="-bind ${ip} -port 8443 -dest ${EXTERNAL_DNS_NAME}"

    ensure_fresh_container_running ${socket} "${run_opts}" "${container_name}" "${container_img}" "${container_cmd}" "${container_cmd_args}"
}

function init_masterless_worker_if_necessary()
{
    if [ "${MASTERLESS_ENABLED}" == "true" ]; then
        start_pod_to_apiserver_proxy
    fi
}

function teardown_masterless_worker_if_necessary()
{
    if [ "${MASTERLESS_ENABLED}" == "true" ]; then
        ensure_container_stopped_or_nonexistent ${socket} pa-proxy
        teardown_veth
    fi
}
