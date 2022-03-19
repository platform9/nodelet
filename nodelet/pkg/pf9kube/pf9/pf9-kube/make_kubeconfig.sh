#!/bin/bash
set -e
cd "$(dirname "$0")"

username="$1"
if [ -z "${username}" ]; then
    echo "user name is missing"
    exit 1
fi

clustername="$2"
if [ -z "${clustername}" ]; then
    echo "cluster name is missing"
    exit 1
fi

uuid="$3"
if [ -z "${uuid}" ]; then
    echo "request uuid is missing"
    exit 1
fi

source /etc/pf9/kube.env
source utils.sh

# Create kubeconfig for kubectl to write to, and ensure it is destroyed on exit
KUBECONFIG=$(mktemp --tmpdir=/tmp kubeconfig.XXXX)
trap "rm -f ${KUBECONFIG}" EXIT
make_kubeconfig "$username" "$clustername" "$KUBECONFIG"

CONTENT_TYPE="content-type: application/octet-stream"
url=http://localhost:9080/tackboard/
NO_PROXY=localhost curl -sX POST --data-binary @${KUBECONFIG} -H "${CONTENT_TYPE}" -H "uuid: ${uuid}" ${url}
