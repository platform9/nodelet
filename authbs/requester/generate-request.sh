#!/usr/bin/env bash
set -euo pipefail

if [ $# == 0 ]; then
    echo "Usage: $0 <name> </path/to/csr> </path/to/key> <pki_dir> [sans]"
    exit 1
fi

CMD_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
. ${CMD_DIR}/easyrsa.env

sans=""
org=""

while test $# -gt 0; do
   opt="${1%%=*}"
   val="${1#*=}"
   case "$opt" in
    --name)      name="$val" ;;
    --csr_file)  csr_path=$val ;;
    --key_file)  key_path="$val" ;;
    --pki_dir)   export EASYRSA_PKI="$val" ;;
    --sans)      sans="$val" ;;
    --org)       org="$val" ;;
    *) break ;;
   esac
   shift
done

if [ ! -d "$EASYRSA_PKI" ]; then
    echo creating $EASYRSA_PKI
    echo "yes" | $EASYRSA_CMD init-pki
fi

filename=$RANDOM
cmd_args=" --batch --req-cn=$name"
if [ -n "$sans" ]; then
    cmd_args+=" --subject-alt-name=${sans}"
fi
if [ -n "$org" ]; then
    cmd_args+=" --req-email= --dn-mode=org --req-org=${org}"
fi
cmd_args+=" gen-req $filename nopass"

$EASYRSA_CMD $cmd_args

mv "${EASYRSA_PKI}/reqs/${filename}.req" "$csr_path"
mv "${EASYRSA_PKI}/private/${filename}.key" "$key_path"
