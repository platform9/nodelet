#!/bin/sh

if [ -z $KUBECTL ]
then
    export KUBECTL="/opt/pf9/qbert/bin/kubectl"
fi

if [ "$#" -lt 3 ]
then
    echo "Usage: $0 <example-name> <setup|teardown> <kubectl opts>"
    echo "Example: $0 guestbook setup --server=http://1.2.3.4:5678/"
    echo "Example: $0 guestbook setup --kubeconfig=/tmp/kubeconfig.yaml"
    exit 1
fi

example=$1
command=$2
shift 2
kubectl_opts=$@

$(dirname $0)/$example/$command.sh $kubectl_opts
