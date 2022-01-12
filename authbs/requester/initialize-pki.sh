#!/usr/bin/env bash
set -euo pipefail

CMD_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
. ${CMD_DIR}/easyrsa.env

function setup_easyrsa()
{
    echo "Setting up easyrsa"
    pushd "$CMD_DIR"
    if [ ! -f easy-rsa.tar.gz ]; then
        echo "Downloading easyrsa"
        curl -L -O https://storage.googleapis.com/kubernetes-release/easy-rsa/easy-rsa.tar.gz
    fi
    tar xzf easy-rsa.tar.gz > /dev/null 2>&1
    popd
}

if [ ! -f "$EASYRSA_CMD" ]; then setup_easyrsa; fi
