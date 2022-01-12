#!/usr/bin/env bash
# Note no set -e; we want to ignore errors so tests can use this to
# clean up before running setup.sh.
nginx_dir=`dirname $0`
source $nginx_dir/wait_until.sh
kubectl_opts=$@
kubectl_cmd="$KUBECTL ${kubectl_opts}"
num_nodes=`$kubectl_cmd get nodes|grep " Ready "|wc -l`

function main()
{
    # idempotent, will retry if api reports an error with etcd
    echo "Tearing down nginx autoscaler example"
    wait_until "$kubectl_cmd delete --ignore-not-found -f $nginx_dir" 4 15

    echo Waiting for nginx pods to disappear
    wait_until 'no_pods_of_this_type app=nginx' 10 6

    echo "Waiting for newly created nodes to disappear"
    wait_until "wait_for_nodes_to_disappear" 30 660
}

function no_pods_of_this_type()
{
    local selector=$1
    [ `$kubectl_cmd get --selector=${selector} pods|tail -n+2|wc -l` == '0' ]
}

function wait_for_nodes_to_disappear()
{
    [ `$kubectl_cmd get nodes|grep " Ready "|wc -l` < $num_nodes  ]
}

main
