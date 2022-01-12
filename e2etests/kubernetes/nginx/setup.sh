#!/usr/bin/env bash
set -e
if [ -n "$DEBUG" ]; then
    set -x
fi

nginx_dir=`dirname $0`
source $nginx_dir/wait_until.sh
kubectl_opts=$@
kubectl_cmd="$KUBECTL ${kubectl_opts}"
num_nodes=`$kubectl_cmd get nodes|grep "Ready"|wc -l`

function main()
{
    stage
    deploy
    test
}

function stage()
{
    staging_dir=$(mktemp -d)
    trap "rm -rf $staging_dir" EXIT
    cp -r ${nginx_dir}/* "$staging_dir"
}

function deploy()
{
    echo "Setting up nginx deployment"
    wait_until "$kubectl_cmd create ns nginx"
    wait_until "$kubectl_cmd apply -f $staging_dir" 4 15

    echo "Waiting for nginx deployment"
    wait_until "two_running_nginx_pods" 10 90
}

function test()
{
    echo "Scaling nginx deployment"
    wait_until "$kubectl_cmd patch deployment nginx -n nginx --patch '{\"spec\": {\"replicas\": 15}}'" 10 90

    echo "Waiting for nginx deployment to reflect updated replicas"
    wait_until "fifteen_nginx_pods" 10 90

    echo "Waiting for k8s cluster to reflect newly created nodes due to autoscaling"
    wait_until "wait_for_new_nodes" 30 660

    echo "Waiting for 15 running nginx pods"
    wait_until "fifteen_running_nginx_pods" 30 180
}

function two_running_nginx_pods()
{
    [ `$kubectl_cmd get --selector='app=nginx' pods -n nginx|grep Running|wc -l` == '2' ]
}

function fifteen_nginx_pods()
{
    [ `$kubectl_cmd get --selector='app=nginx' pods -n nginx|grep nginx| wc -l` == '15' ]
}

function fifteen_running_nginx_pods()
{
    [ `$kubectl_cmd get --selector='app=nginx' pods -n nginx|grep Running| wc -l` == '15' ]
}

function wait_for_new_nodes()
{
    [ `$kubectl_cmd get nodes|grep " Ready "|wc -l` > $num_nodes ]
}

main
