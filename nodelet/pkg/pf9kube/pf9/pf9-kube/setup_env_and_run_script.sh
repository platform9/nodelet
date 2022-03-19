#!/bin/bash

# This script is responsible to setting up the environment by sourcing relevant
# files (for bash functions and variables) for running the task scripts.
CONFIG_DIR=${CONFIG_DIR:-/etc/pf9}
DATE_FORMAT="+%Y-%m-%d %H:%M:%S"
LOG_FILE="/var/log/pf9/kube/kube.log"
source $CONFIG_DIR/kube.env
# This is useful for dev & test. For example, on a test system I typically
# have /opt/pf9/pf9-kube/utils.sh symlinked to utils.sh in git repo, which
# does not have __KUBERNETES_VERSION__ substituted, so I use this extra file
# to define KUBERNETES_VERSION explicitly, or else things break.      -leb
if [ -e "$CONFIG_DIR/kube_extra_opts.env" ]
then
    source $CONFIG_DIR/kube_extra_opts.env
fi
# Added to allow, specifying special flags like PF9_MANAGED_DOCKER which lets
# isv customers bypass setting up of docker on the hosts as part of pf9-kube role.
# Note: docker0 will come up with 172.17.0.0/16 or 172.18.0.0/16 or the next available /16 CIDR
# Customer needs to make sure that any cluster's Service/Container CIDR do not overlap with docker CIDR
if [ -e "$CONFIG_DIR/kube_override.env" ]
then
    source $CONFIG_DIR/kube_override.env
fi

# based on http://unix.stackexchange.com/a/26729
function prefix_timestamp()
{
    while IFS= read -r line;
        do printf '[%s] %s\n' "$(date "$DATE_FORMAT")" "$line";
    done
}

function do_cmd()
{
    echo --- $@ at "$(date "$DATE_FORMAT")" --- | tee -a $LOG_FILE
    if [ $# -eq 3 ] && [ "$3" == "--debug" ]; then
        export DEBUG=true
    fi
    $@ 2>&1 | prefix_timestamp | tee -a $LOG_FILE
    return ${PIPESTATUS[0]}
}

# $0 - This script
# $1 - The script to run. Only allow scripts to be run from /opt/pf9/pf9-kube/phases
# $2 - The argument to pass to script that will be run.
#      Valid values are - start, stop, status, can_run_status and name
valid_files="/opt/pf9/pf9-kube/phases/.*\.sh"
if [[ ! $1 =~ $valid_files ]]; then
    echo "Cannot execute [$1] file with this script."
    exit 1
fi
valid_operations="^(name|status|start|stop|can_run_status)$"
if [[ ! $2 =~ $valid_operations ]]; then
    echo "Invalid operation [$2]"
    exit 1
fi

cmd_no_logs="^(name|can_run_status)$"
if [[ $2 =~ $cmd_no_logs ]]; then
    # "name" and "can_run_status" operations are special. It only prints the name of
    # the task and if it has a status check, this will be recorded in nodelet datastructure.
    # Don't print additional debugging information to simplify the string parsing in nodelet.
    export DEBUG=false
    $1 $2
else
    cd /opt/pf9/pf9-kube/
    source utils.sh

    # Export proxy settings to make them available to all phase scripts.
    ensure_http_proxy_configured >> /dev/null
    export NODE_NAME_TYPE=$(get_node_name_type)
    export NODE_IP=$(ip_address_of_default_gw_nic)
    export NODE_NAME=$(get_node_name)
    export HOSTNAME=`hostname`

    do_cmd $1 $2 $3
fi
