
#@IgnoreInspection BashAddShebang
source /etc/os-release

service="containerd"
cli="/opt/pf9/pf9-kube/bin/nerdctl"
crictl="/opt/pf9/pf9-kube/bin/crictl"
socket="$CONTAINERD_SOCKET"

function pf9ctr_status()
{
    systemctl status $service
}

function pf9ctr_enable()
{
    systemctl enable $service
}

function pf9ctr_start()
{
    systemctl start $service
}

function pf9ctr_restart()
{
    systemctl restart $service
}

function pf9ctr_is_active()
{
    systemctl is-active $service
}

function pf9ctr_stop()
{
    systemctl stop $service
}

function pf9ctr_run()
{
    if [ "$pf9_kube_http_proxy_configured" = "true" ]; then
        http_proxy=$http_proxy https_proxy=$https_proxy HTTP_PROXY=$HTTP_PROXY HTTPS_PROXY=$HTTPS_PROXY no_proxy=$no_proxy NO_PROXY=$NO_PROXY $cli -n k8s.io --cgroup-manager=$CONTAINERD_CGROUP -H unix://$socket "$@"
    else
        $cli -n k8s.io --cgroup-manager=$CONTAINERD_CGROUP -H unix://$socket "$@"
    fi
}

function pf9ctr_crictl()
{
    $crictl -r unix://$socket "$@"
}

function pf9ctr_run_with_sudo()
{
    sudo $cli -n k8s.io --cgroup-manager=$CONTAINERD_CGROUP -H unix://$socket "$@"
}