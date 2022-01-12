#!/bin/sh

#
# Run kubernetes tests against a cluster by specifying its master host UUID.
# NOTE: Must be on run on the DU.
#

if [ "$#" -lt 3 ]
then
    echo "Usage: $0 <master_host_uuid> <test-name> <setup|teardown> [kubectl opts]"
    echo "Example: $0 guestbook setup --kubeconfig=/tmp/kubeconfig.yaml"
    echo "NOTE: do not set the --server option, since it's set by this script"
    exit 1
fi

if ! systemctl status pf9-forwarder &> /dev/null; then
    echo "Requires pf9-forwarder service to run. (Are you running on the DU?)"
    exit 2
fi

masterHostUuid=$1
example=$2
command=$3
shift 3
kubectl_opts=$@

cd `dirname $0`
./kubetest.sh $example $command ${masterHostUuid} ${kubectl_opts}
