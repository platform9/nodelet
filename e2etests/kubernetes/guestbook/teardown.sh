#!/usr/bin/env bash
# Note no set -e; we want to ignore errors so tests can use this to
# clean up before running setup.sh.
guestbook_dir=`dirname $0`
source $guestbook_dir/wait_until.sh
kubectl_opts=$@
kubectl_cmd="$KUBECTL ${kubectl_opts}"

function main()
{
    # idempotent, will retry if api reports an error with etcd
    echo "Tearing down kubernetes guestbook example"
    wait_until "$kubectl_cmd delete --ignore-not-found -f $guestbook_dir" 4 15

    echo Waiting for frontend pods to disappear
    wait_until 'no_pods_of_this_type tier=frontend,app=guestbook' 10 6

    echo Waiting for redis master pods to disappear
    wait_until 'no_pods_of_this_type app=redis,role=master,tier=backend' 10 6

    echo Waiting for redis slave pods to disappear
    wait_until 'no_pods_of_this_type app=redis,role=slave,tier=backend' 10 6

    echo Waiting for busybox pod to disappear
    wait_until "! $kubectl_cmd describe pod busybox-0 &> /dev/null" 10 12
}

function no_pods_of_this_type()
{
    local selector=$1
    kubectl_out=$($kubectl_cmd get --selector=${selector} pods)
    echo "$kubectl_out"
    [ $(echo "$kubectl_out"|tail -n+2|wc -l) == '0' ]
}

main
