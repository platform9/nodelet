#@IgnoreInspection BashAddShebang

service="docker"
cli="/usr/bin/docker"
socket="$DOCKER_SOCKET"

function pf9ctr_status()
{
    systemctl status $service
}

function pf9ctr_enable()
{
    systemctl enable $service containerd
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
    $cli -H unix://$socket "$@"
}

function pf9ctr_run_with_sudo()
{
    sudo $cli -H unix://$socket "$@"
}