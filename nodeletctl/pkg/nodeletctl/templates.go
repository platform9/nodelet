package nodeletctl

const masterNodeletConfigTmpl = `
ALLOW_WORKLOADS_ON_MASTER: {{ .AllowWorkloadsOnMaster }}
API_SERVER_FLAGS: ""
APISERVER_STORAGE_BACKEND: etcd3
APP_CATALOG_ENABLED: "false"
AUTHZ_ENABLED: "true"
BOUNCER_SLOW_REQUEST_WEBHOOK: ""
CALICO_IPIP_MODE: Always
CALICO_IPV4: autodetect
CALICO_IPV4_BLOCK_SIZE: "26"
CALICO_IPV4_DETECTION_METHOD: {{ .CalicoV4Interface }}
CALICO_IPV6: none
CALICO_IPV6_DETECTION_METHOD: {{ .CalicoV6Interface }}
CALICO_IPV6POOL_BLOCK_SIZE: "116"
CALICO_IPV6POOL_CIDR: ""
CALICO_IPV6POOL_NAT_OUTGOING: "false"
CALICO_NAT_OUTGOING: "true"
CALICO_ROUTER_ID: hash
CLOUD_PROVIDER_TYPE: local
CLUSTER_ID: {{ .ClusterId }}
CLUSTER_PROJECT_ID: 373d078433b8422490fdfcd96d406805
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
  ETCD_NAME={{ .HostId }}
  ETCD_STRICT_RECONFIG_CHECK=true
  ETCD_INITIAL_CLUSTER_TOKEN={{ .ClusterId }}
  ETCD_INITIAL_CLUSTER_STATE={{ .EtcdClusterState }}
  ETCD_INITIAL_CLUSTER={{- range $MasterName, $MasterIp := .MasterList }}{{ $MasterName }}=https://{{ $MasterIp }}:2380,{{ end }}
  ETCD_INITIAL_ADVERTISE_PEER_URLS=https://{{ .HostIp }}:2380
  ETCD_LISTEN_PEER_URLS=https://{{ .HostIp }}:2380
  ETCD_ADVERTISE_CLIENT_URLS=https://{{ .HostIp }}:4001
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
EXTERNAL_DNS_NAME: ""
EXTRA_OPTS: ""
FELIX_IPV6SUPPORT: "false"
GCR_PRIVATE_REGISTRY: ""
HOSTID: {{ .HostId }}
IPV6_ENABLED: "false"
K8S_API_PORT: "443"
K8S_PRIVATE_REGISTRY: ""
KEYSTONE_DOMAIN: kubernetes-keystone.platform9.horse
KEYSTONE_ENABLED: "true"
KUBE_PROXY_MODE: ipvs
KUBE_SERVICE_STATE: "true"
KUBELET_CLOUD_CONFIG: ""
MASTER_IP: {{ .MasterIp }}
MASTER_VIP_ENABLED: {{ .MasterVipEnabled }}
MASTER_VIP_IFACE: {{ .MasterVipInterface }}
MASTER_VIP_PRIORITY: ""
MASTER_VIP_VROUTER_ID: {{ .MasterVipVrouterId }}
MASTERLESS_ENABLED: "false"
MAX_NUM_WORKERS: "0"
METALLB_CIDR: ""
METALLB_ENABLED: "false"
MIN_NUM_WORKERS: "0"
MTU_SIZE: {{ .Mtu }}
PF9_NETWORK_PLUGIN: calico
PRIVILEGED: {{ .Privileged }}
QUAY_PRIVATE_REGISTRY: ""
REGISTRY_MIRRORS: https://dockermirror.platform9.io/
RESERVED_CPUS: ""
ROLE: {{ .NodeletRole }}
RUNTIME: {{ .ContainerRuntime.Name }}
CONTAINERD_CGROUP:  {{ .ContainerRuntime.CgroupDriver }}
DOCKER_CGROUP: {{ .ContainerRuntime.CgroupDriver }}
RUNTIME_CONFIG: ""
SCHEDULER_FLAGS: ""
SERVICES_CIDR: 10.21.0.0/22
TOPOLOGY_MANAGER_POLICY: none
USE_HOSTNAME: "false"
STANDALONE: "true"
DOCKER_ROOT: /var/lib/docker
{{if .UserImages -}}
USER_IMAGES_DIR: "/var/opt/pf9/images"
{{ end -}}
{{ if .CoreDNSHostsFile -}}
COREDNS_HOSTS_FILE: "/etc/pf9/hosts"
{{ else -}}
COREDNS_HOSTS_FILE: "/etc/hosts"
{{ end }}
`

const workerNodeletConfigTmpl = `
ALLOW_WORKLOADS_ON_MASTER: {{ .AllowWorkloadsOnMaster }}
API_SERVER_FLAGS: ""
APISERVER_STORAGE_BACKEND: etcd3
APP_CATALOG_ENABLED: "false"
AUTHZ_ENABLED: "true"
BOUNCER_SLOW_REQUEST_WEBHOOK: ""
CALICO_IPIP_MODE: Always
CALICO_IPV4: autodetect
CALICO_IPV4_BLOCK_SIZE: "26"
CALICO_IPV4_DETECTION_METHOD: {{ .CalicoV4Interface }}
CALICO_IPV6: none
CALICO_IPV6_DETECTION_METHOD: {{ .CalicoV6Interface }}
CALICO_IPV6POOL_BLOCK_SIZE: "116"
CALICO_IPV6POOL_CIDR: ""
CALICO_IPV6POOL_NAT_OUTGOING: "false"
CALICO_NAT_OUTGOING: "true"
CALICO_ROUTER_ID: hash
CLOUD_PROVIDER_TYPE: local
CLUSTER_ID: {{ .ClusterId }}
CLUSTER_PROJECT_ID: 373d078433b8422490fdfcd96d406805
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
EXTERNAL_DNS_NAME: ""
EXTRA_OPTS: ""
FELIX_IPV6SUPPORT: "false"
GCR_PRIVATE_REGISTRY: ""
HOSTID: {{ .HostId }}
IPV6_ENABLED: "false"
K8S_API_PORT: "443"
K8S_PRIVATE_REGISTRY: ""
KEYSTONE_DOMAIN: kubernetes-keystone.platform9.horse
KEYSTONE_ENABLED: "true"
KUBE_PROXY_MODE: ipvs
KUBE_SERVICE_STATE: "true"
KUBELET_CLOUD_CONFIG: ""
MASTER_IP: {{ .MasterIp }}
MASTER_VIP_ENABLED: {{ .MasterVipEnabled }}
MASTER_VIP_IFACE: {{ .MasterVipInterface }}
MASTER_VIP_PRIORITY: ""
MASTER_VIP_VROUTER_ID: ""
MASTERLESS_ENABLED: "false"
MAX_NUM_WORKERS: "0"
METALLB_CIDR: ""
METALLB_ENABLED: "false"
MIN_NUM_WORKERS: "0"
MTU_SIZE: {{ .Mtu }}
PF9_NETWORK_PLUGIN: calico
PRIVILEGED: {{ .Privileged }}
QUAY_PRIVATE_REGISTRY: ""
REGISTRY_MIRRORS: https://dockermirror.platform9.io/
RESERVED_CPUS: ""
ROLE: {{ .NodeletRole }}
RUNTIME: {{ .ContainerRuntime.Name }}
CONTAINERD_CGROUP:  {{ .ContainerRuntime.CgroupDriver }}
DOCKER_CGROUP: {{ .ContainerRuntime.CgroupDriver }}
RUNTIME_CONFIG: ""
SCHEDULER_FLAGS: ""
SERVICES_CIDR: 10.21.0.0/22
TOPOLOGY_MANAGER_POLICY: none
USE_HOSTNAME: "false"
STANDALONE: "true"
DOCKER_ROOT: /var/lib/docker
{{if .UserImages -}}
USER_IMAGES_DIR: "/var/opt/pf9/images"
{{ end -}}
{{ if .CoreDNSHostsFile -}}
COREDNS_HOSTS_FILE: "/etc/pf9/hosts"
{{ else -}}
COREDNS_HOSTS_FILE: "/etc/hosts"
{{ end }}
`
const adminKubeconfigTemplate = `
apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: {{ .CACertData }}
    server: https://{{ .MasterIp }}:{{ .K8sApiPort }}
  name: {{ .ClusterId }}
contexts:
- context:
    cluster: {{ .ClusterId }}
    user: admin
  name: default
current-context: default
kind: Config
preferences: {}
users:
- name: admin
  user:
    client-certificate-data: {{ .ClientCertData }}
    client-key-data: {{ .ClientKeyData }}
`
