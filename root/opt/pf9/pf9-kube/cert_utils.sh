#!/usr/bin/env bash

source utils.sh

CMD_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cacert=/etc/pf9/kube.d/rootCA.crt
cakey=/etc/pf9/kube.d/rootCA.key
openssl_ca_conf=/etc/pf9/kube.d/openssl_ca.conf

function download_missing_certs() {
    #input parameter is array of missing certs to try.
    missing_cert_paths=$1

    local cert_lpid
    for try_cert_path in "${missing_cert_paths[@]}"
    do
        params_for_cmnd=${cert_path_to_params_map["$try_cert_path"]}
        cert $params_for_cmnd "--certs_dir=$try_cert_path" &
        cert_lpid="$!"
        cert_pids_to_path_map[${cert_lpid}]="$try_cert_path"
    done
}

function run_certs_requests() {
    echo 'Inside run_certs_requests'
    declare -a missing_cert_paths

    missing_cert_paths=( ${!cert_path_to_params_map[@]} )

    retries=0
    retry_needed=false

    #internally trying for MAX_CERTS_RETRIES times.
    while [ "$retries" -lt ${MAX_CERTS_RETRIES} ]
    do
            #Defining a map to be used when running commands to map pids to certpaths
            declare -A cert_pids_to_path_map

            download_missing_certs $missing_cert_paths
            #clear missing_cert_paths
            unset missing_cert_paths

            for pid in "${!cert_pids_to_path_map[@]}";
            do
                if wait $pid; then
                    loc_path=${cert_pids_to_path_map[$pid]}
                    mkdir -p "${CERTS_DIR}/${loc_path}/"
                    cp -a "${tmp_dir}/${loc_path}"/* "${CERTS_DIR}/${loc_path}/"
                else
                    retry_needed=true
                    echo "Cert missed in this round: ${cert_pids_to_path_map[$pid]}"
                    missing_cert_paths=("${missing_cert_paths[@]}" "${cert_pids_to_path_map[$pid]}")
                fi
            done

            if [ "$retry_needed" == true ]; then
                retry_needed=false
                retries=$((retries+1))
                unset cert_pids_to_path_map
                echo "Retrying again internally"

            else
                #successful
                ensure_dir_readable_by_pf9 $CERTS_DIR
                touch $CERTS_DIR/.done
                chown -R ${PF9_USER}:${PF9_GROUP} $CERTS_DIR/.done
                return
            fi
    done

    return 1

}

function init_pki() {
    # TODO, we will want to mock this out at some point, so we can
    # build these RPMs independently of the DU/Qbert.
    echo "yes" | ${CMD_DIR}/bin/requester/initialize-pki.sh
}

function vault_get_svc_acct_key() {

    curl --silent \
    -H "X-Vault-Token: $VAULT_TOKEN" \
    ${VAULT_ADDR}/v1/secret/$CLUSTER_ID \
    > $certs_dir/svcacct.json

    cat $certs_dir/svcacct.json | /opt/pf9/pf9-kube/bin/jq -r '.data.service_account_key' > ${certs_dir}/svcacct.key

}

function vault_sign_csr() {

    local vault_role=$1
    local csr_filepath=$2
    local cert_file=$3
    local ca_file=$4

    echo "Signing certificate request for $vault_role..."

    # Escape newline chars \n as \\n for JSON string formatting.
    local csr=$(awk 'NF {sub(/\r/, ""); printf "%s\\n",$0;}' $csr_filepath)

    curl --silent \
    -d "{\"csr\":\"$csr\"}" \
    -H "X-Vault-Token: $VAULT_TOKEN" \
    ${VAULT_ADDR}/v1/pmk-ca-${CLUSTER_ID}/sign/${vault_role} \
    > $certs_dir/request.json

    cat $certs_dir/request.json | extract_vault_json certificate > $cert_file
    cat $certs_dir/request.json | extract_vault_json issuing_ca > $ca_file
}

function cert() {
    local name
    local cert_type
    local certs_dir
    local sans
    local needs_svcacctkey
    local org
    local certs_dir_loc

    while test $# -gt 0; do
        opt="${1%%=*}"
        val="${1#*=}"
        case "$opt" in
        --cn)         name="$val";;
        --cert_type)  cert_type=$val;;
        --certs_dir)  certs_dir_loc="$val" ;;
            --sans)       sans="$val" ;;
        --needs_svcacctkey) needs_svcacctkey="$val" ;;
        --org)        org="$val" ;;
        *) break ;;
        esac
        shift
    done

    certs_dir="${tmp_dir}/${certs_dir_loc}"
    mkdir -p "$certs_dir"

    local csr_file="${certs_dir}/request.csr"
    local key_file="${certs_dir}/request.key"
    local certs_file="${certs_dir}/certs.tgz"
    local log_file="${certs_dir}/request.log"
    local pki_dir="${certs_dir}/pki"

    # Generate certificate request.
    ${CMD_DIR}/bin/requester/generate-request.sh --name="$name" --csr_file="$csr_file" --key_file="$key_file" --pki_dir="$pki_dir" --sans="$sans" --org="$org" &> $log_file

    if [ "${needs_svcacctkey}" == "true" ]; then
        vault_get_svc_acct_key
    fi

    local cert_file="${certs_dir}/request.crt"
    local ca_file="${certs_dir}/ca.crt"

    # common name for kube-scheduler and kube-controller-manager will have "system:" prefix that needs to be removed as vault
    # doesn't like it for certificate name
    if [[ "$name" == "system:"* ]]; then
        name=`echo $name | sed -e 's/system://'`
    fi

    if [ $STANDALONE == "true" ]; then
        create_root_ca_if_needed
        if [ "${needs_svcacctkey}" == "true" ]; then
            cp $cakey ${certs_dir}/svcacct.key
        fi
        self_sign_csr $name $csr_file $cert_file $ca_file $sans
    else
        vault_sign_csr "$name-$cert_type" "$csr_file" "$cert_file" "$ca_file"
    fi

    # Validate presence or absence of service account key.
    if [ "${needs_svcacctkey}" == "true" ] && [ ! -e ${certs_dir}/svcacct.key ] ; then
        echo "Service account key missing"
        exit 1
    elif [ "${needs_svcacctkey}" != "true" ] && [ -e ${certs_dir}/svcacct.key ] ; then
        echo "Unexpected service account key"
        exit 1
    fi

    # Verify that the certificate is signed by a CA.
    if ! openssl verify -CAfile "$ca_file" "$cert_file"; then
        echo "Certificate is not signed by CA"
        exit 1
    fi

    # Verify that the certificate matches the private key.
    local cert_mod=$(openssl x509 -noout -modulus -in "${certs_dir}/request.crt" | openssl md5)
    local key_mod=$(openssl rsa -noout -modulus -in "${key_file}" | openssl md5)
    echo "$cert_mod $key_mod"
    if [ "$cert_mod" != "$key_mod" ]; then
        echo "Private key does not match certificate"
        exit 1
    fi
}

function create_root_ca_if_needed()
{
    if ! [ -f $cacert ]; then
        cat <<EOF > $openssl_ca_conf
[ req ]
default_md = sha256
prompt = no
req_extensions = req_ext
distinguished_name = req_distinguished_name
[ req_distinguished_name ]
commonName = kubernetes
[ req_ext ]
keyUsage=critical,digitalSignature,keyEncipherment
extendedKeyUsage=critical,serverAuth,clientAuth
subjectAltName = @alt_names
[ alt_names ]
IP.0 = $NODE_IP
IP.1 = 127.0.0.1
DNS.1 = localhost
EOF
        openssl req -x509 -sha256 -days 3650 -newkey rsa:2048 -keyout $cakey -out $cacert -config $openssl_ca_conf -nodes
    fi
}

function self_sign_csr()
{
    name=$1
    csr=$2
    cert=$3
    ca=$4
    sans=$5

    if [ "x$sans" == "x" ]; then
        sans="DNS:$name"
    fi
    dir=`dirname $csr`
    openssl_temp_conf=$dir/openssl_$name.conf
    echo -e "[v3_req]\nkeyUsage=critical,digitalSignature,keyEncipherment\nextendedKeyUsage=critical,serverAuth,clientAuth\nsubjectAltName=$sans" > $openssl_temp_conf
    openssl x509 -req -CA $cacert -CAkey $cakey -in $csr -out $cert -days 365 -CAcreateserial -extensions v3_req -extfile $openssl_temp_conf
    cp $cacert $ca
}
