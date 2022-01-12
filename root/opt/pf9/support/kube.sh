#!/bin/bash

# Scripts in this directory are invoked by pf9-hostagent during support
# bundle generation. This script's output will be included in the bundle.

set -x
set -o allexport
export SHELLOPTS
# Note: some of these runtime commands can fail depending on whether the node
# is a master or worker, that's expected and non-fatal.

# OS version information
source /etc/os-release

# Change permissions to read runtime log folders
sudo chmod -R a+rX /var/lib/docker/containers
sudo chmod -R a+rX /var/lib/containerd
sudo chmod -R a+rX /var/lib/nerdctl

source /etc/pf9/kube.env
if [ -f /etc/pf9/kube_override.env ]; then
    source /etc/pf9/kube_override.env
fi

cd /opt/pf9/pf9-kube/
# Default configuration
source defaults.env
# Runtime specific functions
source runtime.sh

get_logs_runtime_components(){
        components=(etcd proxy bouncer)
        for comp_name in "${components[@]}"; do
                comp_log="$(pf9ctr_run_with_sudo inspect $comp_name | sudo /opt/pf9/pf9-kube/bin/jq -r '.[0].LogPath')"
                if [ ! -z "$comp_log" ]; then
                        # Collecting last two log files
                        cp $comp_log "/var/log/pf9/kube/$comp_name.log"
                        cp $comp_log.1 "/var/log/pf9/kube/$comp_name.log.1"
                        echo Collected $comp_name logs
                fi;
        done
        echo -------------------------------------------------
        pf9ctr_run_with_sudo ps -a
        echo -------------------------------------------------
        pf9ctr_run_with_sudo images
        echo -------------------------------------------------
}

ubuntu::get_apparmor_status(){
        app_armor="sudo apparmor_status"
        apparmor_log="/var/log/pf9/apparmor_status.log"
        echo "Running command: $app_armor" > $apparmor_log
        $app_armor >> $apparmor_log 2>&1
}

get_logs_runtime_daemon(){
        local runtime
        if [ "$RUNTIME" == "containerd" ]; then
            runtime=containerd.service
        else
            runtime=docker.service
        fi
        sudo journalctl -u $runtime > "/var/log/pf9/kube/runtime.log"
        gzip -f "/var/log/pf9/kube/runtime.log"
}

centos::get_logs_kubelet(){
        sudo journalctl -u pf9-kubelet.service > "/var/log/pf9/kube/kubelet.log"
        gzip -f "/var/log/pf9/kube/kubelet.log"
}

get_logs_k8s(){
        echo -------------------------------------------------
        k8s_components=(kube-scheduler kube-apiserver kube-controller)
        for k8s_comp_name in "${k8s_components[@]}"; do
            if [ $RUNTIME == "containerd" ]; then
                # nerdctl cannot fetch logs of containers not started by nerdctl
                collect_cri_container_logs $k8s_comp_name
                continue
            fi
            k8s_container_id=$(pf9ctr_run_with_sudo ps | grep $k8s_comp_name | awk '{print $1}')
            k8s_comp_log="$(pf9ctr_run_with_sudo inspect $k8s_container_id | sudo /opt/pf9/pf9-kube/bin/jq -r '.[0].LogPath')"
            if [ ! -z "$k8s_comp_log" ]; then
                cp $k8s_comp_log "/var/log/pf9/kube/$k8s_comp_name.log"
                cp $k8s_comp_log.1 "/var/log/pf9/kube/$k8s_comp_name.log.1"
                echo Collected $k8s_comp_name logs
            fi
        done
        echo -------------------------------------------------
}

get_logs_networking(){
        echo -------------------------------------------------
        networking_log="/var/log/pf9/networking.log"
        echo "collecting networking logs..." > $networking_log
        echo -------------------------------------------------
        commands=("ip a" "route -n" "brctl show" "sudo iptables --list -t nat" "netstat -anlp"
                  "getenforce" "cat /etc/resolv.conf" "cat /etc/hosts"
                 )
        for command_name in "${commands[@]}"; do
                echo "-------------------------------------------------" >> $networking_log
                echo "Running command: $command_name" >> $networking_log 2>&1
                $command_name >> $networking_log 2>&1
        done
}

get_kube_env(){
        /bin/cat /etc/pf9/kube.env > "/var/log/pf9/kube/kube.env"
}

collect_direct_runtime_container_logs(){
        if [ -z "$1" ]
        then
                echo "No network backend specified"
                return
        else
                echo "Network backend is: \"$1\""
        fi

        if [ -z "$2" ]
        then
                return
        fi

        container_id=$2
        log="$(pf9ctr_run_with_sudo inspect $container_id | sudo /opt/pf9/pf9-kube/bin/jq -r '.[0].LogPath')"
        if [ ! -z "$log" ]; then
                cp $log "/var/log/pf9/kube/$1.log"
                if [ -f $log.1 ]; then
                    cp $log.1 "/var/log/pf9/kube/$1.log.1"
                fi
                echo "Collected $1 logs"
        fi;
}

collect_cri_container_logs(){
    comp_name=$1
    # nerdctl cannot fetch logs of containers not started by nerdctl
    container_id=$(sudo /opt/pf9/pf9-kube/bin/crictl -r unix://$CONTAINERD_SOCKET ps | grep $comp_name | awk '{print $1}')
    sudo /opt/pf9/pf9-kube/bin/crictl -r unix://$CONTAINERD_SOCKET logs $container_id > /var/log/pf9/kube/$comp_name.log 2>&1
}

get_cni_logs(){
    if [ "$RUNTIME" == "containerd" ]; then
        declare -A networks=(
            ["flannel"]="flannel"
            # only component names needed. Log collection function will figure out the container ID.
            ["calico_controller"]="calico-kube-controllers"
            ["calico_node"]="calico-node"
        )
    else
        declare -A networks=(
            ["flannel"]="flannel"
            ["calico_controller"]="$(pf9ctr_run_with_sudo ps | grep k8s_calico-kube-controllers | awk '{print $1}')"
            ["calico_node"]="$(pf9ctr_run_with_sudo ps | grep k8s_calico-node | awk '{print $1}')"
        )
    fi

    for backend in "${!networks[@]}"; do
        if [[ "$RUNTIME" == "docker" || "$backend" == "flannel" ]]; then
            collect_direct_runtime_container_logs "$backend" "${networks[$backend]}";
        else
            collect_cri_container_logs "${networks[$backend]}"
        fi
    done
}

get_keepalived_conf(){
        sudo /bin/cat /etc/keepalived/keepalived.conf > "/var/log/pf9/kube/keepalived.conf"
}

get_keepalived_log(){
        sudo journalctl -u keepalived.service > "/var/log/pf9/kube/keepalived.log"
        gzip -f "/var/log/pf9/kube/keepalived.log"
}

get_kubelet_config() {
        sudo chown -R pf9:pf9group /var/opt/pf9/kube/kubelet-config
        sudo chmod -R +r /var/opt/pf9/kube/kubelet-config
        cp -r /var/opt/pf9/kube/kubelet-config /var/log/pf9/kube
}

function_list=(
        get_logs_k8s
        get_logs_runtime_components
        get_logs_networking
        get_kube_env
        get_cni_logs
        get_keepalived_conf
        get_keepalived_log
        get_kubelet_config
)

echo "From /etc/os-release: ID=$ID VERSION_ID=$VERSION_ID"

case $ID in
        ubuntu)
                function_list+=(
                        ubuntu::get_apparmor_status
                        get_logs_runtime_daemon
                        )
                ;;
        centos|rhel)
                function_list+=(
                        get_logs_runtime_daemon
                        centos::get_logs_kubelet
                )
                ;;
        *)
                echo "Unknown OS: $ID $VERSION_ID. Assuming systemd support."
                function_list+=(get_logs_runtime_daemon)
                ;;
esac

export -f collect_cri_container_logs # Not exported along with others since this is a helper function
# Exporting each function as a subprocess
# and invoking those with timeout
for func in "${function_list[@]}"; do
        export -f $func
        timeout -s KILL 50 bash -c $func
done
