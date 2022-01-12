#!/usr/bin/env bash
set -e
if [ -n "$DEBUG" ]; then
    set -x
fi
guestbook_dir=`dirname $0`
source $guestbook_dir/wait_until.sh
kubectl_opts=$@
kubectl_cmd="$KUBECTL ${kubectl_opts}"

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
    cp -r ${guestbook_dir}/* "$staging_dir"

    # Use guestbook app images from our dockerhub repo. GCR pulls have not been reliable at times
    sed -i 's#k8s.gcr.io#platform9#' "${staging_dir}/redis-master-deployment.yaml"
    sed -i 's#gcr.io/google_samples#platform9#' "${staging_dir}/redis-slave-deployment.yaml"
    sed -i 's#gcr.io\/google-samples#platform9#' "${staging_dir}/frontend-deployment.yaml"

    if [ -n "$CLOUD_PROVIDER_TYPE" ]; then
        echo "Preparing frontend-service.yaml for '$CLOUD_PROVIDER_TYPE' cloud provider type"
        case "$CLOUD_PROVIDER_TYPE" in
            aws)
                # idempotent
                sed -i 's/  # \(type: LoadBalancer\)/  \1/' "${staging_dir}/frontend-service.yaml"
                ;;
        esac
    fi
}

function deploy()
{
    # idempotent, will retry if api reports an error with etcd
    echo "Setting up kubernetes guestbook example"
    wait_until "$kubectl_cmd apply -f $staging_dir" 4 15

    echo Waiting for redis master
    wait_until "one_running_redis_master" 10 90

    echo Waiting for redis slaves
    wait_until "two_running_redis_slaves" 10 90

    echo Waiting for frontend pods
    wait_until "three_running_frontend_pods" 10 90

    echo Waiting for busybox pod
    wait_until "$kubectl_cmd get pods busybox-0 | grep Running" 10 90
}

function test()
{
    echo Testing frontend
    # Expect 200 OK; do not persist the result to a file
    wait_until "$kubectl_cmd exec busybox-0 -- wget --output=/dev/null http://frontend/" 10 36

    echo Testing KubeDNS resolution of frontend service
    wait_until "$kubectl_cmd exec busybox-0 nslookup frontend" 10 36

    if [[ "${CLOUD_PROVIDER_TYPE}" = aws ]]; then
        echo Waiting for frontend service to be available via AWS ELB
        wait_until "frontend_listens_at_elb" 10 60
    fi
}


function three_running_frontend_pods()
{
    kubectl_out=$($kubectl_cmd get --selector='tier=frontend,app=guestbook' pods)
    echo "$kubectl_out"
    [ $(echo "$kubectl_out" | grep Running | wc -l) == '3' ]
}

function one_running_redis_master()
{
    kubectl_out=$($kubectl_cmd get --selector='app=redis,role=master,tier=backend' pods)
    echo "$kubectl_out"
    [ $(echo "$kubectl_out" | grep Running | wc -l) == '1' ]
}

function two_running_redis_slaves()
{
    kubectl_out=$($kubectl_cmd get --selector='app=redis,role=slave,tier=backend' pods)
    echo "$kubectl_out"
    [ $(echo "$kubectl_out" | grep Running | wc -l) == '2' ]
}

function frontend_listens_at_elb()
{
    lb_endpoint=$($kubectl_cmd get services frontend -o='jsonpath={.status.loadBalancer.ingress[0].hostname}')
    rval=$?
    echo "$lb_endpoint"
    if [ "$rval" != 0 -o -z "$lb_endpoint" ]; then
        echo "No LB endpoint found"
        return 1
    fi
    wget --timeout=10 -O- "$lb_endpoint"
}

main
