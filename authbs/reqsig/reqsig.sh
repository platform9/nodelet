#!/usr/bin/env bash
# Note: if run as root, the files extracted from the certs tarball will
# have the permissions set by the signer (run as root, tar enforces the
# --preserve-permissions flag)
set -euo pipefail

if [ $# == 0 ]; then
    echo "Usage: $0 <proxy url> <cert type> </path/to/csr> </path/to/certs.tgz> <cluster id> <sans> <needs_svcacctkey(true|false)>"
    exit 1
fi

proxy_url=$1
cert_type=$2
csr_path=$3
certs_path=$4
cluster_id=$5
sans=${6:-""}
needs_svcacctkey=${7:-"false"}

host_id="unknown_host_id"
if [ -e /etc/pf9/host_id.conf ] ; then
    host_id=`grep host_id /etc/pf9/host_id.conf | cut -d' ' -f3`
fi

curl "${proxy_url}/${cluster_id}/${cert_type}/" \
    --fail \
    -X POST \
    -H "Subject-Alt-Names: ${sans}" \
    -H "Node-Id: ${host_id}" \
    -H "Needs-Service-Account-Key: ${needs_svcacctkey}" \
    --data-binary @${csr_path} \
    -o "$certs_path"
