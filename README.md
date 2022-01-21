## NODELET

## Compiling and creating OS packages

1. Install the build pre-reqs
   ```
   sudo apt-get update
   sudo apt-get install ruby-dev rpm build-essential docker.io -y
   sudo gem i fpm -f
   curl -O https://dl.google.com/go/go1.17.1.linux-amd64.tar.gz
   sudo tar -C /usr/local -xzf go1.17.1.linux-amd64.tar.gz
   mkdir -p ~/go; echo "export GOPATH=$HOME/go" >> ~/.bashrc
   echo "export PATH=$PATH:$HOME/go/bin:/usr/local/go/bin" >> ~/.bashrc
   source ~/.bashrc
   ```
2. Compile and create rpm/deb packages
   ```
   make agent-deb # to create a deb package
   make agent-rpm # to create a rpm package
   ```
## Installation steps

This section contains instructions on creating a single-node cluster using nodelet. Instructions for multi-node cluster and multi-master clusters will be added shortly. 
1. Create pf9 user and group
   ```
   mkdir -p /opt/pf9/home
   groupadd pf9group
   useradd -d /opt/pf9/home -G pf9group pf9
   ```
2. Create the necessary config files. Replace the IP address of the node.
   ```
   mkdir -p /etc/pf9/nodelet /etc/pf9/kube.d
   touch /etc/pf9/nodelet/config_sunpike.yaml /etc/pf9/kube_resmgr.env
   ```
   ```
   # Contents of /etc/pf9/nodelet/config_sunpike.yaml
   ALLOW_WORKLOADS_ON_MASTER: "true"
   API_SERVER_FLAGS: ""
   APISERVER_STORAGE_BACKEND: etcd3
   APP_CATALOG_ENABLED: "false"
   AUTHZ_ENABLED: "true"
   BOUNCER_SLOW_REQUEST_WEBHOOK: ""
   CALICO_IPIP_MODE: Always
   CALICO_IPV4: autodetect
   CALICO_IPV4_BLOCK_SIZE: "26"
   CALICO_IPV4_DETECTION_METHOD: first-found
   CALICO_IPV6: none
   CALICO_IPV6_DETECTION_METHOD: first-found
   CALICO_IPV6POOL_BLOCK_SIZE: "116"
   CALICO_IPV6POOL_CIDR: ""
   CALICO_IPV6POOL_NAT_OUTGOING: "false"
   CALICO_NAT_OUTGOING: "true"
   CALICO_ROUTER_ID: hash
   CLOUD_PROVIDER_TYPE: local
   CLUSTER_ID: cbe813ef-8a68-4af5-bc7d-7242e3ec4c5d
   CLUSTER_PROJECT_ID: 373d078433b8422490fdfcd96d406805
   CNI_BRIDGE: cni0
   CONTAINERS_CIDR: 10.20.0.0/22
   CONTROLLER_MANAGER_FLAGS: ""
   CPU_MANAGER_POLICY: none
   DEBUG: "true"
   DEPLOY_KUBEVIRT: "false"
   DEPLOY_LUIGI_OPERATOR: "false"
   DOCKER_CENTOS_REPO_URL: ""
   DOCKER_LIVE_RESTORE_ENABLED: "true"
   DOCKER_PRIVATE_REGISTRY: ""
   DOCKER_ROOT: /var/lib
   DOCKER_UBUNTU_REPO_URL: ""
   DOCKERHUB_ID: ""
   DOCKERHUB_PASSWORD: ""
   ENABLE_CAS: "false"
   ENABLE_PROFILE_AGENT: "true"
   ETCD_DATA_DIR: /var/opt/pf9/kube/etcd/data
   ETCD_DISCOVERY_URL: ""
   ETCD_ELECTION_TIMEOUT: "1000"
   ETCD_ENV: |-
     ETCD_NAME=be0324eb-f74b-4eeb-8437-19ad9a3307f4
     ETCD_STRICT_RECONFIG_CHECK=true
     ETCD_INITIAL_CLUSTER_TOKEN=cbe813ef-8a68-4af5-bc7d-7242e3ec4c5d
     ETCD_INITIAL_CLUSTER_STATE=new
     ETCD_INITIAL_CLUSTER=be0324eb-f74b-4eeb-8437-19ad9a3307f4=https://<NODE IP ADDRESS>:2380
     ETCD_INITIAL_ADVERTISE_PEER_URLS=https://<NODE IP ADDRESS>:2380
     ETCD_LISTEN_PEER_URLS=https://<NODE IP ADDRESS>:2380
     ETCD_ADVERTISE_CLIENT_URLS=https://<NODE IP ADDRESS>:4001
     ETCD_LISTEN_CLIENT_URLS=https://0.0.0.0:4001,http://127.0.0.1:2379
     ETCD_DATA_DIR=/var/etcd/data
     ETCD_CERT_FILE=/certs/etcd/client/request.crt
     ETCD_KEY_FILE=/certs/etcd/client/request.key
     ETCD_TRUSTED_CA_FILE=/certs/etcd/client/ca.crt
     ETCD_PEER_KEY_FILE=/certs/etcd/peer/request.key
     ETCD_PEER_CERT_FILE=/certs/etcd/peer/request.crt
     ETCD_PEER_TRUSTED_CA_FILE=/certs/etcd/peer/ca.crt
     ETCD_CLIENT_CERT_AUTH=true
     ETCD_DEBUG=false
     ETCD_CIPHER_SUITES=TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256
   ETCD_HEARTBEAT_INTERVAL: "100"
   ETCD_VERSION: ""
   EXTERNAL_DNS_NAME: <NODE IP ADDRESS>
   EXTRA_OPTS: ""
   FELIX_IPV6SUPPORT: "false"
   FLANNEL_IFACE_LABEL: ""
   FLANNEL_PUBLIC_IFACE_LABEL: ""
   GCR_PRIVATE_REGISTRY: ""
   HOSTID: be0324eb-f74b-4eeb-8437-19ad9a3307f4
   IPV6_ENABLED: "false"
   K8S_API_PORT: "443"
   K8S_PRIVATE_REGISTRY: ""
   KEYSTONE_DOMAIN: kubernetes-keystone.platform9.horse
   KEYSTONE_ENABLED: "true"
   KUBE_PROXY_MODE: ipvs
   KUBE_SERVICE_STATE: "true"
   KUBELET_CLOUD_CONFIG: ""
   MASTER_IP: <NODE IP ADDRESS>
   MASTER_VIP_ENABLED: "false"
   MASTER_VIP_IFACE: ""
   MASTER_VIP_PRIORITY: ""
   MASTER_VIP_VROUTER_ID: ""
   MASTERLESS_ENABLED: "false"
   MAX_NUM_WORKERS: "0"
   METALLB_CIDR: ""
   METALLB_ENABLED: "false"
   MIN_NUM_WORKERS: "0"
   MTU_SIZE: "1440"
   OS_AUTH_URL: ""
   OS_PASSWORD: ""
   OS_PROJECT_DOMAIN_NAME: ""
   OS_PROJECT_NAME: ""
   OS_REGION: ""
   OS_USER_DOMAIN_NAME: ""
   OS_USERNAME: ""
   PF9_NETWORK_PLUGIN: calico
   PRIVILEGED: "true"
   QUAY_PRIVATE_REGISTRY: ""
   REGISTRY_MIRRORS: https://dockermirror.platform9.io/
   RESERVED_CPUS: ""
   ROLE: master
   RUNTIME: containerd
   RUNTIME_CONFIG: ""
   SCHEDULER_FLAGS: ""
   SERVICES_CIDR: 10.21.0.0/22
   TOPOLOGY_MANAGER_POLICY: none
   USE_HOSTNAME: "false"
   STANDALONE: "true"
   DOCKER_ROOT: /var/lib/docker
   ```
   
   ```
   # Contents of /etc/pf9/kube_resmgr.env
   export ALLOW_WORKLOADS_ON_MASTER="true"
   export APISERVER_STORAGE_BACKEND="etcd3"
   export API_SERVER_FLAGS=""
   export APP_CATALOG_ENABLED="false"
   export AUTHZ_ENABLED="true"
   export BOUNCER_SLOW_REQUEST_WEBHOOK=""
   export CALICO_IPIP_MODE="Always"
   export CALICO_IPV4="autodetect"
   export CALICO_IPV4_BLOCK_SIZE="26"
   export CALICO_IPV4_DETECTION_METHOD="first-found"
   export CALICO_IPV6="none"
   export CALICO_IPV6POOL_BLOCK_SIZE="116"
   export CALICO_IPV6POOL_CIDR=""
   export CALICO_IPV6POOL_NAT_OUTGOING="false"
   export CALICO_IPV6_DETECTION_METHOD="first-found"
   export CALICO_NAT_OUTGOING="true"
   export CALICO_ROUTER_ID="hash"
   export CLOUD_PROVIDER_TYPE="local"
   export CLUSTER_ID="cbe813ef-8a68-4af5-bc7d-7242e3ec4c5d"
   export CLUSTER_PROJECT_ID="373d078433b8422490fdfcd96d406805"
   export CNI_BRIDGE="cni0"
   export CONTAINERS_CIDR="10.20.0.0/22"
   export CONTROLLER_MANAGER_FLAGS=""
   export CPU_MANAGER_POLICY="none"
   export DEBUG="false"
   export DEPLOY_KUBEVIRT="false"
   export DEPLOY_LUIGI_OPERATOR="false"
   export DOCKERHUB_ID=""
   export DOCKERHUB_PASSWORD=""
   export DOCKER_CENTOS_REPO_URL=""
   export DOCKER_LIVE_RESTORE_ENABLED="true"
   export DOCKER_PRIVATE_REGISTRY=""
   export DOCKER_ROOT="/var/lib"
   export DOCKER_UBUNTU_REPO_URL=""
   export ENABLE_CAS="false"
   export ENABLE_PROFILE_AGENT="true"
   export ETCD_DATA_DIR="/var/opt/pf9/kube/etcd/data"
   export ETCD_DISCOVERY_URL=""
   export ETCD_ELECTION_TIMEOUT="1000"
   export ETCD_ENV="ETCD_NAME=be0324eb-f74b-4eeb-8437-19ad9a3307f4\nETCD_STRICT_RECONFIG_CHECK=true\nETCD_INITIAL_CLUSTER_TOKEN=cbe813ef-8a68-4af5-bc7d-7242e3ec4c5d\nETCD_INITIAL_CLUSTER_STATE=new\nETCD_INITIAL_CLUSTER=be0324eb-f74b-4eeb-8437-19ad9a3307f4=https://<NODE IP ADDRESS>:2380\nETCD_INITIAL_ADVERTISE_PEER_URLS=https://<NODE IP ADDRESS>:2380\nETCD_LISTEN_PEER_URLS=https://<NODE IP ADDRESS>:2380\nETCD_ADVERTISE_CLIENT_URLS=https://<NODE IP ADDRESS>:4001\nETCD_LISTEN_CLIENT_URLS=https://0.0.0.0:4001,http://127.0.0.1:2379\nETCD_DATA_DIR=/var/etcd/data\nETCD_CERT_FILE=/certs/etcd/client/request.crt\nETCD_KEY_FILE=/certs/etcd/client/request.key\nETCD_TRUSTED_CA_FILE=/certs/etcd/client/ca.crt\nETCD_PEER_KEY_FILE=/certs/etcd/peer/request.key\nETCD_PEER_CERT_FILE=/certs/etcd/peer/request.crt\nETCD_PEER_TRUSTED_CA_FILE=/certs/etcd/peer/ca.crt\nETCD_CLIENT_CERT_AUTH=true\nETCD_DEBUG=false\nETCD_CIPHER_SUITES=TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"
   export ETCD_HEARTBEAT_INTERVAL="100"
   export ETCD_VERSION=""
   export EXTERNAL_DNS_NAME="<NODE IP ADDRESS>"
   export EXTRA_OPTS=""
   export FELIX_IPV6SUPPORT="false"
   export FLANNEL_IFACE_LABEL=""
   export FLANNEL_PUBLIC_IFACE_LABEL=""
   export GCR_PRIVATE_REGISTRY=""
   export HOSTID="be0324eb-f74b-4eeb-8437-19ad9a3307f4"
   export IPV6_ENABLED="false"
   export K8S_API_PORT="443"
   export K8S_PRIVATE_REGISTRY=""
   export KEYSTONE_DOMAIN="kubernetes-keystone.platform9.horse"
   export KEYSTONE_ENABLED="true"
   export KUBELET_CLOUD_CONFIG=""
   export KUBE_PROXY_MODE="ipvs"
   export KUBE_SERVICE_STATE="true"
   export MASTERLESS_ENABLED="false"
   export MASTER_IP="<NODE IP ADDRESS>"
   export MASTER_VIP_ENABLED="false"
   export MASTER_VIP_IFACE=""
   export MASTER_VIP_PRIORITY=""
   export MASTER_VIP_VROUTER_ID=""
   export MAX_NUM_WORKERS="0"
   export METALLB_CIDR=""
   export METALLB_ENABLED="false"
   export MIN_NUM_WORKERS="0"
   export MTU_SIZE="1440"
   export OS_AUTH_URL=""
   export OS_PASSWORD=""
   export OS_PROJECT_DOMAIN_NAME=""
   export OS_PROJECT_NAME=""
   export OS_REGION=""
   export OS_USERNAME=""
   export OS_USER_DOMAIN_NAME=""
   export PF9_NETWORK_PLUGIN="calico"
   export PRIVILEGED="true"
   export QUAY_PRIVATE_REGISTRY=""
   export REGISTRY_MIRRORS="https://dockermirror.platform9.io/"
   export RESERVED_CPUS=""
   export ROLE="master"
   export RUNTIME="containerd"
   export RUNTIME_CONFIG=""
   export SCHEDULER_FLAGS=""
   export SERVICES_CIDR="10.21.0.0/22"
   export TOPOLOGY_MANAGER_POLICY="none"
   export USE_HOSTNAME="false"
   export STANDALONE="true"
   ```
3. Install the rpm or deb according to your OS. Currently we only support CentOS 7.8, CentOS 7.9, Ubuntu 18 and Ubuntu 20. Support for other OS and creating a OS independent nodelet binary is in-progress.
   ```
   yum install <RPM>
   OR
   apt install <DEB>
   ```
4. Create a symlink to python3. (We are actively working on removing this dependency)
   ```
   mkdir -p /opt/pf9/python/bin
   ln -s `which python3` /opt/pf9/python/bin/python
   ```
4. Start the nodelet service
   ```
   /opt/pf9/nodelet/nodeletd --disable-sunpike
   ```

