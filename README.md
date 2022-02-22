## NODELET

Nodelet is a Kubernetes lifecycle manager which can be used stand-alone or as part of a larger system (e.g. Cluster API). Nodelet is capable of performing the following tasks:
- Installing a Kubernetes stack on one or more nodes
- Configuring the cluster
- Configuring a set of core add-ons
- Upgrades

Nodelet has a fine-grain orchestration system wherein individual steps needed to configure the stack are logically separated and report independent status, making it easier to assess the health of the system and pinpoint failures.

Currently nodelet supports k8s 1.21 only. Support for other k8s versions is in progress.

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

This section contains instructions on creating a single master cluster using nodelet. Instructions for multi-master clusters will be added shortly.
1. Create the config directories -
   ```
   mkdir -p /etc/pf9/nodelet /etc/pf9/kube.d
   ```

2. Generate CA certificates that will be used for signing all the certificates for various k8s components. This step is optional when creating a single node cluster.
We are actively working on documenting a more streamlined way of generating and sharing certificates using Hashicorp Vault.

   a. Create a OpenSSL conf package
      ```
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
      IP.0 = <NODE IP ADDRESS>
      IP.1 = 127.0.0.1
      DNS.1 = localhost
      ```
   b. Generate CA using OpenSSL
      ```
      openssl req -x509 -sha256 -days 3650 -newkey rsa:2048 -keyout /etc/pf9/kube.d/rootCA.key -out /etc/pf9/kube.d/rootCA.crt -config <OpenSSL conf file> -nodes
      ```
   c. Copy `/etc/pf9/kube.d/rootCA.key` and `/etc/pf9/kube.d/rootCA.crt` to all the nodes. The location of these files must be same on all hosts i.e. `/etc/pf9/kube.d/rootCA.*`

3. Create the necessary config files. Replace the IP address of the node. Create /etc/pf9/nodelet/config_sunpike.yaml on master node with following contents -
   ```
   # Contents of /etc/pf9/nodelet/config_sunpike.yaml
   ALLOW_WORKLOADS_ON_MASTER: "true" # whether to allow workloads on master. Valid values are - "true" & "false"
   API_SERVER_FLAGS: "" # comma separated list of arguments to be provided to apiserver
   APISERVER_STORAGE_BACKEND: etcd3 
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
   CLUSTER_ID: cbe813ef-8a68-4af5-bc7d-7242e3ec4c5d # unique ID for each cluster. Must be modified when reusing a node from another cluster managed by nodelet.
   CNI_BRIDGE: cni0
   CONTAINERS_CIDR: 10.20.0.0/22 # container subnet
   CONTROLLER_MANAGER_FLAGS: "" # comma separated list of arguments to be provided to controller manager
   CPU_MANAGER_POLICY: none
   DEBUG: "true"
   DOCKER_CENTOS_REPO_URL: "" # URL to yum repo for downloading docker. Yum repo configured on the host will be used when left empty.
   DOCKER_LIVE_RESTORE_ENABLED: "true"
   DOCKER_PRIVATE_REGISTRY: ""
   DOCKER_ROOT: /var/lib
   DOCKER_UBUNTU_REPO_URL: "" # URL to apt repo for downloading docker. Apt repo configured on the host will be used when left empty.
   DOCKERHUB_ID: "" 
   DOCKERHUB_PASSWORD: ""
   ETCD_DATA_DIR: /var/opt/pf9/kube/etcd/data # location where etcd will store data
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
   EXTERNAL_DNS_NAME: <NODE IP ADDRESS> 
   FELIX_IPV6SUPPORT: "false"
   FLANNEL_IFACE_LABEL: ""
   FLANNEL_PUBLIC_IFACE_LABEL: ""
   GCR_PRIVATE_REGISTRY: ""
   HOSTID: be0324eb-f74b-4eeb-8437-19ad9a3307f4 # Unique ID for each node
   IPV6_ENABLED: "false"
   K8S_API_PORT: "443"
   K8S_PRIVATE_REGISTRY: ""
   KEYSTONE_DOMAIN: kubernetes-keystone.platform9.horse
   KUBE_PROXY_MODE: ipvs
   KUBE_SERVICE_STATE: "true"
   KUBELET_CLOUD_CONFIG: ""
   MASTER_IP: <NODE IP ADDRESS>
   MTU_SIZE: "1440"
   PF9_NETWORK_PLUGIN: calico
   PRIVILEGED: "true"
   QUAY_PRIVATE_REGISTRY: ""
   REGISTRY_MIRRORS: "" # comma separated list of docker registry mirrors
   RESERVED_CPUS: ""
   ROLE: master
   RUNTIME: containerd # container runtime. Valid values are "docker" and "containerd"
   RUNTIME_CONFIG: ""
   SCHEDULER_FLAGS: ""
   SERVICES_CIDR: 10.21.0.0/22
   TOPOLOGY_MANAGER_POLICY: none
   USE_HOSTNAME: "false"
   STANDALONE: "true"
   ```

4. Create /etc/pf9/nodelet/config_sunpike.yaml on all worker nodes with following contents -
   ```
   # Contents of /etc/pf9/nodelet/config_sunpike.yaml
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
   CLUSTER_ID: cbe813ef-8a68-4af5-bc7d-7242e3ec4c5d # unique ID for each cluster. Must be modified when reusing a node from another cluster managed by nodelet.
   CNI_BRIDGE: cni0
   CONTAINERS_CIDR: 10.20.0.0/22 # container subnet
   CONTROLLER_MANAGER_FLAGS: "" # comma separated list of arguments to be provided to controller manager
   CPU_MANAGER_POLICY: none
   DEBUG: "true"
   DOCKER_CENTOS_REPO_URL: "" # URL to yum repo for downloading docker. Yum repo configured on the host will be used when left empty.
   DOCKER_LIVE_RESTORE_ENABLED: "true"
   DOCKER_PRIVATE_REGISTRY: ""
   DOCKER_ROOT: /var/lib
   DOCKER_UBUNTU_REPO_URL: "" # URL to apt repo for downloading docker. Apt repo configured on the host will be used when left empty.
   DOCKERHUB_ID: "" 
   DOCKERHUB_PASSWORD: ""
   EXTERNAL_DNS_NAME: <MASTER NODE IP ADDRESS> 
   FELIX_IPV6SUPPORT: "false"
   FLANNEL_IFACE_LABEL: ""
   FLANNEL_PUBLIC_IFACE_LABEL: ""
   GCR_PRIVATE_REGISTRY: ""
   HOSTID: be0324eb-f74b-4eeb-8437-19ad9a3307f4 # Unique ID for each node
   IPV6_ENABLED: "false"
   K8S_API_PORT: "443"
   K8S_PRIVATE_REGISTRY: ""
   KEYSTONE_DOMAIN: kubernetes-keystone.platform9.horse
   KUBE_PROXY_MODE: ipvs
   KUBE_SERVICE_STATE: "true"
   KUBELET_CLOUD_CONFIG: ""
   MASTER_IP: <MASTER NODE IP ADDRESS>
   MTU_SIZE: "1440"
   PF9_NETWORK_PLUGIN: calico
   PRIVILEGED: "true"
   QUAY_PRIVATE_REGISTRY: ""
   REGISTRY_MIRRORS: "" # comma separated list of docker registry mirrors
   RESERVED_CPUS: ""
   ROLE: worker
   RUNTIME: containerd # container runtime. Valid values are "docker" and "containerd"
   RUNTIME_CONFIG: ""
   SCHEDULER_FLAGS: ""
   SERVICES_CIDR: 10.21.0.0/22
   TOPOLOGY_MANAGER_POLICY: none
   USE_HOSTNAME: "false"
   STANDALONE: "true"
   ```
   Replace the master node IP address in this config file. 

5. Install the rpm or deb according to your OS on all the hosts. Currently nodelet only supports CentOS 7.8, CentOS 7.9, Ubuntu 18 and Ubuntu 20. Support for other OS and creating a OS independent nodelet binary is in-progress.
   ```
   yum install <RPM>
   OR
   apt install <DEB>
   ```

6. Start the nodelet service on all the hosts
   ```
   systemctl daemon-reload
   systemctl start pf9-nodeletd
   ```
   
