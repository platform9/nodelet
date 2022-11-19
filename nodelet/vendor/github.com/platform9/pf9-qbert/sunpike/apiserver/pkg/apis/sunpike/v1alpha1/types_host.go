package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// HostState is the status of the Host.
type HostState string

const (
	HostStateUnknown    HostState = "Unknown"
	HostStateOk         HostState = "Ok"
	HostStateConverging HostState = "Converging"
	HostStateRetrying   HostState = "Retrying"
	HostStateFailed     HostState = "Failed"
)

var HostStateSet = map[HostState]struct{}{
	HostStateUnknown:    {},
	HostStateOk:         {},
	HostStateConverging: {},
	HostStateRetrying:   {},
	HostStateFailed:     {},
}

// For backward-compatibility
// TODO(erwin) remove this once pf9-kube no longer depends on unversioned pf9-qbert.
type NodeState = HostState

const (
	NodeStateUnknown    = HostStateUnknown
	NodeStateOk         = HostStateOk
	NodeStateConverging = HostStateConverging
	NodeStateRetrying   = HostStateRetrying
	NodeStateFailed     = HostStateFailed
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Host is a representation of the configuration and status of a single machine.
type Host struct {
	metav1.TypeMeta `json:",inline" protobuf:"bytes,4,opt,name=typeMeta"`

	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Specification of the desired behavior of the Host.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status
	Spec HostSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`

	// Most recently observed status of the Host.
	// This data may not be up to date.
	// Populated by the system.
	// Read-only.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status
	Status HostStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// HostList is a list of Host objects.
type HostList struct {
	metav1.TypeMeta `json:",inline" protobuf:"bytes,3,opt,name=typeMeta"`

	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Items contains a list of Hosts.
	Items []Host `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// HostSpec contains the specification of the desired behavior of the Host.
//
// Dev note: This type MUST be kept in sync with server/sunpike.js in qbert.
// Make sure update both data structures when adding/modifying new fields.
type HostSpec struct {
	// The goal of ExtraCfg is to hold any additional args that have not been
	// formalized yet. Based on discussions with Raghvendra, Arun and Abhimanyu
	// regarding multi-version support it might be beneficial to have a
	// free-form dict object and enforce the validation in the API layer based
	// on a schema that is NOT hardcoded in sunpike.
	ExtraCfg map[string]string `json:"extraCfg,omitempty" protobuf:"bytes,1,opt,name=extraCfg"`

	// PF9Cfg contains miscellaneous configuration related to PF9 services.
	PF9Cfg PF9Opts `json:"pf9,omitempty" protobuf:"bytes,2,opt,name=pf9"`

	// ClusterCfg contains the cluster-wide configuration. These settings
	// include the control plane (apiserver, scheduler, and controller-manager),
	// networking (CNI, KubeProxy), and the ingress load-balancing (MetalLB).
	ClusterCfg KubeClusterOpts `json:"clusterCfg,omitempty" protobuf:"bytes,3,opt,name=clusterCfg"`

	// Etcd contain configuration for the etcd cluster as a storage backend for
	// the Cluster. This is separated from the apiserver settings, because we
	// plan to separate out etcd from the master nodes.
	//
	// More info: https://etcd.io/docs/latest/
	Etcd EtcdOpts `json:"etcd,omitempty" protobuf:"bytes,4,opt,name=etcd"`

	// ExtraOpts can be used to pass arbitrary key value pairs to be used in
	// bash scripts. For example, if ExtraOpts is set to "FOO=BAR,JANE=BOB",
	// then the following will be defined and available in the phase scripts:
	//   export EXTRA_OPT_FOO=BAR
	//   export EXTRA_OPT_JANE=BOB
	ExtraOpts string `json:"extraOpts,omitempty" protobuf:"bytes,5,opt,name=extraOpts" kube.env:"EXTRA_OPTS"`

	// ServicesCIDR contains a CIDR notation IP range from which to assign
	// service cluster IPs. This must not overlap with any IP ranges assigned
	// to nodes or pods.
	ServicesCIDR string `json:"servicesCIDR,omitempty" protobuf:"bytes,6,opt,name=servicesCIDR" kube.env:"SERVICES_CIDR"`

	// ContainersCIDR is the range of pods in the cluster. When configured,
	// traffic sent to a Service cluster IP from outside this range will be
	// masqueraded and traffic sent from pods to an external LoadBalancer IP
	// will be directed to the respective cluster IP instead
	ContainersCIDR string `json:"containersCIDR,omitempty" protobuf:"bytes,7,opt,name=containersCIDR" kube.env:"CONTAINERS_CIDR"`

	// AllowWorkloadsOnMaster signals whether regular workloads are allowed to
	// be run on this Host if it is a master Node.
	AllowWorkloadsOnMaster bool `json:"allowWorkloadsOnMaster,omitempty" protobuf:"bool,8,opt,name=allowWorkloadsOnMaster" kube.env:"ALLOW_WORKLOADS_ON_MASTER"`

	// KubeletOpts contain Kubelet-specific configuration.
	//
	// See more: https://kubernetes.io/docs/reference/command-line-tools-reference/kubelet/
	Kubelet KubeletOpts `json:"kubelet,omitempty" protobuf:"bytes,9,opt,name=kubelet"`

	// DockerOpts are options for the Docker runtime on the Host.
	Docker DockerOpts `json:"docker,omitempty" protobuf:"bytes,10,opt,name=docker"`

	// DockerPrivateRegistry is the location of the docker private registry (if any) that hosts the PF9 container images
	DockerPrivateRegistry string `json:"dockerPrivateRegistry,omitempty" protobuf:"bytes,11,opt,name=dockerPrivateRegistry" kube.env:"DOCKER_PRIVATE_REGISTRY"`

	// QuayPrivateRegistry is the location of the quay private registry (if any) that hosts the PF9 container images
	QuayPrivateRegistry string `json:"quayPrivateRegistry,omitempty" protobuf:"bytes,12,opt,name=quayPrivateRegistry" kube.env:"QUAY_PRIVATE_REGISTRY"`

	// GCRPrivateRegistry is the location of the gcr private registry (if any) that hosts the PF9 container images
	GCRPrivateRegistry string `json:"gcrPrivateRegistry,omitempty" protobuf:"bytes,13,opt,name=gcrPrivateRegistry" kube.env:"GCR_PRIVATE_REGISTRY"`

	// K8SPrivateRegistry is the location of the k8s private registry (if any) that hosts the PF9 container images
	K8SPrivateRegistry string `json:"k8sPrivateRegistry,omitempty" protobuf:"bytes,14,opt,name=k8sPrivateRegistry" kube.env:"K8S_PRIVATE_REGISTRY"`

	// ContainerRuntime specifies the container runtime to use
	ContainerRuntime string `json:"containerRuntime,omitempty" protobuf:"bytes,15,opt,name=containerRuntime" kube.env:"RUNTIME"`

    ServicesCIDRv6 string `json:"servicesCIDRv6,omitempty" protobuf:"bytes,16,opt,name=servicesCIDR" kube.env:"SERVICES_CIDR_V6"`
}

// HostStatus represents information about the status of a Host. Status may
// trail the actual state of a system, especially if the Host is not able to
// contact Sunpike.
type HostStatus struct {
	//
	// State
	//

	// HostState constains the overall state of this Host.
	HostState HostState `json:"hostState,omitempty" protobuf:"bytes,1,rep,name=hostState"`

	// ClusterRole is the role that is currently observed to be installed on
	// the Host.
	ClusterRole string `json:"clusterRole,omitempty" protobuf:"bytes,2,opt,name=clusterRole"`

	// ClusterID is the observed ClusterID. This should be equivalent to the
	// ClusterID in the HostSpec, but is also stored in the HostStatus for
	// backward-compatibility.
	ClusterID string `json:"clusterID,omitempty" protobuf:"bytes,3,opt,name=clusterID"`

	// ServiceState is the current desired state of the Host. This should be
	// equivalent to the KubeServiceState in the HostSpec, but is also stored
	// in the HostStatus for backward-compatibility.
	//
	// If true the Host is being added to the
	// cluster as a Node; if set to false the Host is being removed from the
	// cluster as a Node; and, if set to another value (commonly "" or "ignore")
	// the Host is simply being ignored and left in whatever state it is.
	ServiceState bool `json:"serviceState,omitempty" protobuf:"bool,4,opt,name=serviceState"`

	// Hostname contains the hostname of the Host.
	Hostname string `json:"hostname,omitempty" protobuf:"bytes,5,opt,name=hostname"`

	// Nodelet contains information about the current state of the Nodelet
	// process on the Host.
	Nodelet NodeletStatus `json:"nodelet,omitempty" protobuf:"bytes,6,opt,name=nodelet"`

	//
	// Phases
	//

	// Phases provides details about the progress of the scripts (phases)
	// responsible for bringing the Host into or out of the Cluster. The phases
	// are run in the order they are shown here.
	Phases []HostPhase `json:"phases,omitempty" protobuf:"bytes,10,rep,name=phases"`

	// PhaseCompleted specifies the order of the phase until which the chain has
	// been completed successfully so far.
	PhaseCompleted int32 `json:"phaseCompleted,omitempty" protobuf:"varint,11,opt,name=phaseCompleted"`

	// LastFailedPhase will be the order of the phase which failed to start in
	// last attempt. If none have failed, this will be -1.
	LastFailedPhase int32 `json:"lastFailedPhase,omitempty" protobuf:"varint,12,opt,name=lastFailedPhase"`

	// StartAttempts describes the number of times Nodelet has attempted to run
	// the start scripts.
	StartAttempts int32 `json:"startAttempts,omitempty" protobuf:"varint,13,opt,name=startAttempts"`

	// CurrentPhase is the order of the phase which is currently running.
	CurrentPhase int32 `json:"currentPhase,omitempty" protobuf:"varint,14,opt,name=currentPhase"`

	//
	// StatusChecks
	//

	// AllStatusChecks will be the list of integer orders of the phases that
	// should be run as part of a status check.
	AllStatusChecks []int32 `json:"allStatusChecks,omitempty" protobuf:"bytes,20,rep,name=allStatusChecks"`

	// LastFailedCheck will be the order of the phase for which status check failed.
	LastFailedCheck int32 `json:"lastFailedCheck,omitempty" protobuf:"varint,21,opt,name=lastFailedCheck"`

	// LastFailedCheckTime contains the timestamp at which the last failed
	// status check was performed. The field is formatted as UNIX timestamp.
	LastFailedCheckTime int64 `json:"lastFailedCheckTime,omitempty" protobuf:"varint,22,opt,name=lastFailedCheckTime"`

	// CurrentStatusCheck will be the order of the phase for which status
	// check is running.
	//
	// Note(erwin): with the current reduced status updates, this will always be
	// 				empty. We should consider removing this field.
	CurrentStatusCheck int32 `json:"currentStatusCheck,omitempty" protobuf:"varint,23,opt,name=currentStatusCheck"`

	// CurrentStatusCheckTime specifies the timestamp at which the current
	// status check has started. The field is formatted as UNIX timestamp.
	//
	// Note(erwin): with the current reduced status updates, this will always be
	// 				empty. We should consider removing this field.
	CurrentStatusCheckTime int64 `json:"currentStatusCheckTime,omitempty" protobuf:"varint,24,opt,name=currentStatusCheckTime"`

	// Interfaces contains information related to all the interfaces found on the host
	Interfaces []NetworkInterface `json:"interfaces,omitempty" protobuf:"bytes,25,opt,name=interfaces"`

	AddonOperatorVersion string `json:"addonOperatorVersion,omitempty" protobuf:"bytes,26,opt,name=addonOperatorVersion"`
}

type NetworkInterface struct {
	// Name of the network interface e.g. eth0, ens129, etc.
	Name string `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`

	// IPAddrs contains the list of IP addresses associated with this interface
	IPAddrs []string `json:"ipaddrs,omitempty" protobuf:"bytes,2,opt,name=ipaddrs"`

	// MACAddr is the MAC address of this interface
	MACAddr string `json:"macaddr,omitempty" protobuf:"bytes,3,opt,name=macaddr"`

	// IsDefault will indicate if a default route is associated with this interface
	IsDefault bool `json:"isdefault,omitempty" protobuf:"bytes,4,opt,name=isdefault"`
}

// KubeletOpts contain Kubelet-specific configuration.
//
// See more: https://kubernetes.io/docs/reference/command-line-tools-reference/kubelet/
type KubeletOpts struct {
	// CloudCfg contains the configuration data that will be used in the
	// --cloud-config flag of kubelet. Empty string for no configuration data.
	//
	// (DEPRECATED: will be removed in 1.23, in favor of removing cloud
	// providers code from Kubelet.)
	CloudCfg string `json:"cloudCfg,omitempty" protobuf:"bytes,1,opt,name=cloudCfg" kube.env:"KUBELET_CLOUD_CONFIG"`

	// ProviderID contains the provider ID that is specified by CAPI machine and
	// will be populated in the kubelet running on the host.
	ProviderID string `json:"providerID,omitempty" protobuf:"bytes,2,opt,name=providerID" kube.env:"KUBELET_PROVIDER_ID"`

	// ExtraArgs contains arguments that will be passed as-is to kubelet process
	ExtraArgs string `json:"extraArgs,omitempty" protobuf:"bytes,3,opt,name=extraArgs" kube.env:"KUBELET_FLAGS"`

	// NodeLabels contains labels for the host
	NodeLabels map[string]string `json:"nodeLabels,omitempty" protobuf:"bytes,4,opt,name=nodeLabels"`

	// NodeTaints contains taints for the host
	NodeTaints map[string]string `json:"nodeTaints,omitempty" protobuf:"bytes,5,opt,name=nodeTaints"`
}

// DockerOpts are options for the Docker runtime on the Host.
type DockerOpts struct {
	// LiveRestore enables the Docker live restore capability, which ensures
	// that containers remain running if the daemon becomes (temporarily)
	// unavailable.
	//
	// More info: https://github.com/splunk/docker/blob/master/docs/admin/live-restore.md
	LiveRestore bool `json:"liveRestore,omitempty" protobuf:"bool,1,opt,name=liveRestore" kube.env:"DOCKER_LIVE_RESTORE_ENABLED"`

	// RootDir specifies the parent directory of the Docker daemon root
	// directory. By default, the RootDir is set to /var/lib. The reason that
	// you should provide the parent dir, is that in the phase scripts 'docker'
	// is affixed to this root path.
	RootDir string `json:"rootDir,omitempty" protobuf:"bytes,2,opt,name=rootDir" kube.env:"DOCKER_ROOT"`

	// DockerhubID contains the username of the Docker user to use when interacting
	// with the Dockerhub registry. Setting this will enable Docker to access
	// private repositories and mitigate the pull limits. If no ID is provided,
	// the Host will use unauthenticated requests to pull from the registry.
	DockerhubID string `json:"dockerhubID,omitempty" protobuf:"bytes,3,opt,name=dockerhubID" kube.env:"DOCKERHUB_ID"`

	// DockerhubPassword contains the password of the Docker user to use when
	// interacting with the Dockerhub registry. Setting this will enable Docker to
	// access private repositories and mitigate the pull limits. If no password is
	// provided, the Host will use unauthenticated requests to pull from the registry.
	DockerhubPassword string `json:"dockerhubPassword,omitempty" protobuf:"bytes,4,opt,name=dockerhubPassword" kube.env:"DOCKERHUB_PASSWORD"`

	// RegistryMirrors specifies the Dockerhub mirrors that should be tried for
	// pulling images. The mirrors should be formatted as comma-seperated list
	// of URLs.
	//
	// More info: https://docs.docker.com/registry/recipes/mirror/
	// or https://cloud.google.com/container-registry/docs/pulling-cached-images
	RegistryMirrors string `json:"registryMirrors,omitempty" protobuf:"bytes,5,opt,name=registryMirrors" kube.env:"REGISTRY_MIRRORS"`

	// Docker's Centos Package Repo URL. This will be added to the various mirror Platform9 uses.
	// defaults to empty which means default OS supported repo would be used
	// this is useful when customers want to control what docker version is installed
	DockerCentosPackageRepoUrl string `json:"dockerCentosPackageRepoUrl,omitempty" protobuf:"bytes,6,opt,name=dockerCentosPackageRepoUrl" kube.env:"DOCKER_CENTOS_REPO_URL"`

	// Docker's Ubuntu Package Repo URL. This will be added to the various mirror Platform9 uses.
	// defaults to empty which means default OS supported repo would be used
	// this is useful when customers want to control what docker version is installed
	DockerUbuntuPackageRepoUrl string `json:"dockerUbuntuPackageRepoUrl,omitempty" protobuf:"bytes,7,opt,name=dockerUbuntuPackageRepoUrl" kube.env:"DOCKER_UBUNTU_REPO_URL"`
}

// EtcdOpts contain configuration for the etcd cluster as a storage backend for
// the Cluster.
//
// More info: https://etcd.io/docs/latest/op-guide/configuration/
type EtcdOpts struct {
	// DataDir specifies the path on the Host (!) where the etcd data should
	// be stored.
	DataDir string `json:"dataDir,omitempty" protobuf:"bytes,1,opt,name=dataDir" kube.env:"ETCD_DATA_DIR"`

	// DiscoveryURL is used to bootstrap the cluster.
	// Note(erwin): does not seem to be used.
	DiscoveryURL string `json:"discoveryURL,omitempty" protobuf:"bytes,2,opt,name=discoveryURL" kube.env:"ETCD_DISCOVERY_URL"`

	// ElectionTimeout is the time (in milliseconds) for an election to timeout.
	// It is equivalent to the –-election-timeout flag in etcd.
	//
	// More info: https://etcd.io/docs/latest/tuning/#time-parameters
	ElectionTimeout int32 `json:"electionTimeout,omitempty" protobuf:"varint,3,opt,name=electionTimeout" kube.env:"ETCD_ELECTION_TIMEOUT"`

	// Env is a catch-all field to specify any environment variables that will
	// be propagated to etcd. The environment variables in this field are
	// line-separated. For example:
	//      ETCD_NAME=08e5cfc1-0e35-4ddb-8fd5-0ae68383c831
	//      ETCD_STRICT_RECONFIG_CHECK=true
	//      ETCD_INITIAL_CLUSTER_TOKEN=9a3fb982-4a6d-4c93-896a-fd8e77577c63
	//      ETCD_INITIAL_CLUSTER_STATE=new
	//
	// For the possible environment variables see: https://etcd.io/docs/latest/op-guide/configuration/
	Env string `json:"env,omitempty" protobuf:"bytes,4,opt,name=env" kube.env:"ETCD_ENV"`

	// HeartbeatIntervalMs specifies time (in milliseconds) of a heartbeat interval.
	HeartbeatIntervalMs int32 `json:"heartbeatIntervalMs,omitempty" protobuf:"varint,5,opt,name=heartbeatIntervalMs" kube.env:"ETCD_HEARTBEAT_INTERVAL"`

	// Version specifies the version of etcd to run.
	Version string `json:"version,omitempty" protobuf:"bytes,6,opt,name=version" kube.env:"ETCD_VERSION"`
}

// CalicoOpts are options for the Calico CNI plugin.
//
// More info: https://docs.projectcalico.org/reference/node/configuration
type CalicoOpts struct {
	// IPIPMode is the IPIP Mode to use for the IPv4 POOL created at start up.
	// Options: Always, CrossSubnet, Never (“Off” is also accepted as a synonym for “Never”)
	//
	// Corresponds to the CALICO_IPV4POOL_IPIP environment variable in Calico.
	IPIPMode string `json:"IPIPMode,omitempty" protobuf:"bytes,1,opt,name=IPIPMode" kube.env:"CALICO_IPIP_MODE"`

	// IPv4BlkSize is the block size to use for the IPv4 POOL created at
	// startup. Block size for IPv4 should be in the range 20-32 (inclusive).
	//
	// Corresponds to the CALICO_IPV4POOL_BLOCK_SIZE environment variable in Calico.
	IPv4BlkSize int32 `json:"IPv4BlkSize,omitempty" protobuf:"varint,2,opt,name=IPv4BlkSize" kube.env:"CALICO_IPV4_BLOCK_SIZE"`

	// NatOutgoing controls whether the NAT Outgoing for the IPv4 Pool should
	// be created at start up.
	//
	// Corresponds to the CALICO_IPV4POOL_NAT_OUTGOING environment variable in Calico.
	NatOutgoing bool `json:"NatOutgoing,omitempty" protobuf:"bool,3,opt,name=NatOutgoing" kube.env:"CALICO_NAT_OUTGOING"`

	// IPv4Mode is the IPv4 address to assign this host or detection behavior
	// at startup. For the details of the behavior possible with this field,
	// see: https://docs.projectcalico.org/reference/node/configuration#ip-setting
	//
	// Corresponds to the IP environment variable in Calico.
	IPv4Mode string `json:"IPv4Mode,omitempty" protobuf:"bytes,4,opt,name=IPv4Mode" kube.env:"CALICO_IPV4"`

	// IPv4DetectionMethod specifies the method to use to autodetect the IPv4
	// address for this host. This is only used when the IPv4 address is being
	// autodetected. For details of the valid methods, see:
	// https://docs.projectcalico.org/reference/node/configuration#ip-autodetection-methods
	//
	// Corresponds to the IP_AUTODETECTION_METHOD environment variable in Calico.
	IPv4DetectionMethod string `json:"IPv4DetectionMethod,omitempty" protobuf:"bytes,5,opt,name=IPv4DetectionMethod" kube.env:"CALICO_IPV4_DETECTION_METHOD"`

	// IPv6Mode is the IPv6 address to assign this host or detection behavior
	// at startup. For the details of the behavior possible with this field,
	// see: https://docs.projectcalico.org/reference/node/configuration#ip-setting
	//
	// Corresponds to the IP6 environment variable in Calico.
	IPv6Mode string `json:"IPv6Mode,omitempty" protobuf:"bytes,6,opt,name=IPv6Mode" kube.env:"CALICO_IPV6"`

	// IPv6BlkSize is the block size to use for the IPv6 POOL created at
	// startup. Block size for IPv6 should be in the range 116-128 (inclusive).
	//
	// Corresponds to the CALICO_IPV6POOL_BLOCK_SIZE environment variable in Calico.
	IPv6BlkSize int32 `json:"IPv6BlkSize,omitempty" protobuf:"varint,7,opt,name=IPv6BlkSize" kube.env:"CALICO_IPV6POOL_BLOCK_SIZE"`

	// IPv6PoolCIDR specifies the IPv6 Pool to create if none exists at start-up.
	//
	// Corresponds to the CALICO_IPV6POOL_CIDR environment variable in Calico.
	IPv6PoolCIDR string `json:"IPv6PoolCIDR,omitempty" protobuf:"bytes,8,opt,name=IPv6PoolCIDR" kube.env:"CALICO_IPV6POOL_CIDR"`

	// IPv6PoolNAT controls whether NAT Outgoing for the IPv6 Pool should be
	// created at start up.
	//
	// Corresponds to the CALICO_IPV6POOL_NAT_OUTGOING environment variable in Calico.
	IPv6PoolNAT bool `json:"IPv6PoolNAT,omitempty" protobuf:"bool,9,opt,name=IPv6PoolNAT" kube.env:"CALICO_IPV6POOL_NAT_OUTGOING"`

	// IPv6DetectionMethod specifies the method to use to autodetect the IPv4
	// address for this host. This is only used when the IPv6 address is being
	// autodetected. For details of the valid methods, see:
	// https://docs.projectcalico.org/reference/node/configuration#ip-autodetection-methods
	//
	// Corresponds to the IP6_AUTODETECTION_METHOD environment variable in Calico.
	IPv6DetectionMethod string `json:"IPv6DetectionMethod,omitempty" protobuf:"bytes,10,opt,name=IPv6DetectionMethod" kube.env:"CALICO_IPV6_DETECTION_METHOD"`

	// RouterID sets the router id to use for BGP if no IPv4 address is set on
	// the node. For an IPv6-only system, this may be set to hash. It then uses
	// the hash of the nodename to create a 4 byte router id.
	//
	// Corresponds to the CALICO_ROUTER_ID environment variable in Calico.
	RouterID string `json:"routerID,omitempty" protobuf:"bytes,11,opt,name=routerID" kube.env:"CALICO_ROUTER_ID"`

	// FelixIPv6Support enables Calico networking and security for IPv6 traffic
	// as well as for IPv4.
	//
	// More info: https://docs.projectcalico.org/reference/felix/configuration
	FelixIPv6Support bool `json:"felixIPv6Support,omitempty" protobuf:"bool,12,opt,name=felixIPv6Support" kube.env:"FELIX_IPV6SUPPORT"`

	//Corresponds to the CALICO_NODE_CPU_LIMIT environment variable in Calico.
	NodeCpuLimit string `json:"nodeCpuLimit,omitempty" protobuf:"string,13,opt,name=nodeCpuLimit" kube.env:"CALICO_NODE_CPU_LIMIT"`
	//Corresponds to the CALICO_NODE_MEMORY_LIMIT environment variable in Calico.
	NodeMemoryLimit string `json:"nodeMemoryLimit,omitempty" protobuf:"string,14,opt,name=nodeMemoryLimit" kube.env:"CALICO_NODE_MEMORY_LIMIT"`
	//Corresponds to the CALICO_TYPHA_CPU_LIMIT environment variable in Calico.
	TyphaCpuLimit string `json:"typhaCpuLimit,omitempty" protobuf:"string,15,opt,name=typhaCpuLimit" kube.env:"CALICO_TYPHA_CPU_LIMIT"`
	//Corresponds to the CALICO_TYPHA_MEMORY_LIMIT environment variable in Calico.
	TyphaMemoryLimit string `json:"typhaMemoryLimit,omitempty" protobuf:"string,16,opt,name=typhaMemoryLimit" kube.env:"CALICO_TYPHA_MEMORY_LIMIT"`
	//Corresponds to the CALICO_CONTROLLER_CPU_LIMIT environment variable in Calico.
	ControllerCpuLimit string `json:"controllerCpuLimit,omitempty" protobuf:"string,17,opt,name=controllerCpuLimit" kube.env:"CALICO_CONTROLLER_CPU_LIMIT"`
	//Corresponds to the CALICO_CONTROLLER_MEMORY_LIMIT environment variable in Calico.
	ControllerMemoryLimit string `json:"controllerMemoryLimit,omitempty" protobuf:"string,18,opt,name=controllerMemoryLimit" kube.env:"CALICO_CONTROLLER_MEMORY_LIMIT"`
}

// FlannelOpts are options for the Flannel CNI plugin.
//
// See more: https://github.com/coreos/flannel/blob/master/Documentation/configuration.md
type FlannelOpts struct {
	// InterfaceLabel to use (IP or name) for inter-host communication. Defaults
	// to the interface for the default route on the machine.
	InterfaceLabel string `json:"interfaceLabel,omitempty" protobuf:"bytes,1,opt,name=interfaceLabel" kube.env:"FLANNEL_IFACE_LABEL"`

	// PublicInterfaceLabel specifies the IP accessible by other nodes for
	// inter-host communication. Defaults to the IP of the interface being used
	// for communication.
	PublicInterfaceLabel string `json:"publicInterfaceLabel,omitempty" protobuf:"bytes,2,opt,name=publicInterfaceLabel" kube.env:"FLANNEL_PUBLIC_IFACE_LABEL"`
}

// AWSOpts are options for the AWS VPC-CNI plugin.
//
// See more: https://github.com/aws/amazon-vpc-cni-k8s/blob/af55286bb5429a06841d2940597410dcc4e74d7e/README.md
type AWSOpts struct {
	// Specifies whether an external NAT gateway should be used to provide SNAT of
	// secondary ENI IP addresses.
	//
	// Corresponds to the AWS_VPC_CNI_EXTERNALSNAT environment variable in aws.
	ExternalSNAT bool `json:"externalSNAT,omitempty" protobuf:"bool,1,opt,name=externalSNAT" kube.env:"AWS_VPC_CNI_EXTERNALSNAT"`
}

// CNIOpts contains the CNI configuration, which includes general options,
// as well as, CNI-specific settings. Current supported CNI plugins are Calico,
// and Flannel, one of which should be non-empty based on the NetworkPlugin
// field (calico or flannel, respectively).
type CNIOpts struct {
	// Bridge specifies the CNI bridge to use.
	// Note(erwin): this seems to be used only for flannel.
	Bridge string `json:"bridge,omitempty" protobuf:"bytes,1,opt,name=bridge" kube.env:"CNI_BRIDGE"`

	// MTUSize configures the MTU to use for workload interfaces and the tunnels.
	// Note(erwin): this seems to be used only for calico.
	MTUSize int32 `json:"MTUSize,omitempty" protobuf:"varint,2,opt,name=MTUSize" kube.env:"MTU_SIZE"`

	// IPv6 indicates whether this cluster should support IPv6.
	// Note(erwin): this seems to be used only for calico.
	IPv6 bool `json:"IPv6,omitempty" protobuf:"bool,3,opt,name=IPv6" kube.env:"IPV6_ENABLED"`

	// NetworkPlugin specifies the CNI plugin to use for this cluster. The
	// options are: calico or flannel. Based on this setting either the Calico
	// field should be filled, or the Flannel field should be filled.
	NetworkPlugin string `json:"networkPlugin,omitempty" protobuf:"bytes,4,opt,name=networkPlugin" kube.env:"PF9_NETWORK_PLUGIN"`

	// Calico contains options specific to the Calico CNI plugin.
	Calico CalicoOpts `json:"calico,omitempty" protobuf:"bytes,5,opt,name=calico"`

	// Flannel contains options specific to the Flannel CNI plugin.
	//
	// More info: https://github.com/coreos/flannel/blob/master/Documentation/configuration.md
	Flannel FlannelOpts `json:"flannel,omitempty" protobuf:"bytes,6,opt,name=flannel"`

	// AWSOpts are options for the AWS VPC-CNI plugin.
	//
	// See more: https://github.com/aws/amazon-vpc-cni-k8s/blob/af55286bb5429a06841d2940597410dcc4e74d7e/README.md
	AWS AWSOpts `json:"aws,omitempty" protobuf:"bytes,7,opt,name=aws"`

    IPv4 bool `json:"IPv4,omitempty" protobuf:"bool,8,opt,name=IPv4" kube.env:"IPV4_ENABLED"`
}

// AddonsOpts is an aggregation of all supported addons.
type AddonsOpts struct {
	// AppCatalog specifies if and how the App Catalog addon should be
	// installed in the cluster.
	AppCatalog AppCatalogOpts `json:"appCatalog,omitempty" protobuf:"bytes,1,opt,name=appCatalog"`

	// CAS specifies if and how the Cluster AutoScaler addon should be
	// deployed in the cluster.
	CAS ClusterAutoScalerOpts `json:"CAS,omitempty" protobuf:"bytes,2,opt,name=CAS"`

	// Luigi contains the settings for the Luigi addons, if enabled.
	Luigi LuigiOpts `json:"luigi,omitempty" protobuf:"bytes,3,opt,name=luigi"`

	// Kubevirt specifies if and how the KubeVirt addon should be
	// installed in the cluster.
	Kubevirt KubeVirtOpts `json:"kubevirt,omitempty" protobuf:"bytes,4,opt,name=kubevirt"`

	// CPUManager defines the options used to manage the CPU and topology
	// manager feature.
	CPUManager CPUManagerOpts `json:"cpuManager,omitempty" protobuf:"bytes,5,opt,name=cpuManager"`

	// ProfileAgent defines the options used to manage and configure the platform9 profile engine agent on the clsuter
	ProfileAgent ProfileAgentOpts `json:"profileAgent,omitempty" protobuf:"bytes,6,opt,name=profileAgent"`

	// AddonOperator defines the options used to manage and configure platform9 addon-operator on the cluster
	AddonOperator AddonOperatorOpts `json:"addonOperator,omitempty" protobuf:"bytes,7,opt,name=addonOperator"`
}

// ProfileAgentOpts contains configuration related to platform9 profile engine agent
type ProfileAgentOpts struct {
	// Enabled signals that profile agent should be installed on the cluster, if set.
	Enabled bool `json:"enabled,omitempty" protobuf:"bool,1,opt,name=enabled" kube.env:"ENABLE_PROFILE_AGENT"`
}

// AddonOperatorOpts contains configuration related to platform9 addon operator
type AddonOperatorOpts struct {
	// taf of addon operator image tag if configured should be used for addon operator configuration on the cluster.
	ImageTag string `json:"imageTag,omitempty" protobuf:"string,1,opt,name=imageTag" kube.env:"ADDON_OPERATOR_IMAGE_TAG"`
}

// AppCatalogOpts contain configuration for the App Catalog addon.
type AppCatalogOpts struct {
	// Enabled signals that AppCatalog should be installed on the cluster, if set.
	Enabled bool `json:"enabled,omitempty" protobuf:"bool,1,opt,name=enabled" kube.env:"APP_CATALOG_ENABLED"`
}

// ClusterAutoScalerOpts contain configuration for the Cluster AutoScaler (CAS)
// addon. This addon is only supported when using the azure or aws cloud
// provider.
type ClusterAutoScalerOpts struct {
	// Enabled signals that CAS should be installed on the cluster, if set.
	Enabled bool `json:"enabled,omitempty" protobuf:"bool,1,opt,name=enabled" kube.env:"ENABLE_CAS"`

	// MinWorkers specifies the minimum number of the workers for the autoscaler
	// to maintain.
	MinWorkers int32 `json:"minWorkers,omitempty" protobuf:"varint,2,opt,name=minWorkers" kube.env:"MIN_NUM_WORKERS"`

	// MaxWorkers specifies the maximum number of the workers for the autoscaler
	// to maintain.
	MaxWorkers int32 `json:"maxWorkers,omitempty" protobuf:"varint,3,opt,name=maxWorkers" kube.env:"MAX_NUM_WORKERS"`
}

// LuigiOpts contain configuration for the Luigi addon.
type LuigiOpts struct {
	// Enabled signals that Luigi should be installed on the cluster, if set.
	Enabled bool `json:"enabled,omitempty" protobuf:"bool,1,opt,name=enabled" kube.env:"DEPLOY_LUIGI_OPERATOR"`
}

// KubeVirtOpts contain configuration for the KubeVirt addon.
type KubeVirtOpts struct {
	// Enabled signals that KubeVirt should be installed on the cluster, if set.
	Enabled bool `json:"enabled,omitempty" protobuf:"bool,1,opt,name=enabled" kube.env:"DEPLOY_KUBEVIRT"`
}

// CPUManagerOpts defines the options used to manage the CPU and topology manager feature
type CPUManagerOpts struct {
	CPUManagerPolicy      string `json:"cpuManagerPolicy,omitempty" protobuf:"bytes,1,opt,name=cpuManagerPolicy" kube.env:"CPU_MANAGER_POLICY"`
	TopologyManagerPolicy string `json:"topologyManagerPolicy,omitempty" protobuf:"bytes,2,opt,name=topologyManagerPolicy" kube.env:"TOPOLOGY_MANAGER_POLICY"`
	ReservedCPUs          string `json:"reservedCPUs,omitempty" protobuf:"bytes,3,opt,name=reservedCPUs" kube.env:"RESERVED_CPUS"`
}

// MetalLBOpts are the options for the MetalLB addon which provides support for
// load-balancer services.
//
// More info: https://metallb.universe.tf
type MetalLBOpts struct {
	// CIDR contains the address range to give MetalLB control over. These will
	// be assigned to services with the type LoadBalancer.
	//
	// Examples: 192.168.1.240-192.168.1.250 or 10.21.0.0/22
	CIDR string `json:"CIDR,omitempty" protobuf:"bytes,1,opt,name=CIDR" kube.env:"METALLB_CIDR"`

	// Enabled signals that the addon should be installed on the cluster, if set.
	Enabled bool `json:"enabled,omitempty" protobuf:"bool,2,opt,name=enabled" kube.env:"METALLB_ENABLED"`
}

// KubeProxyOpts contain settings for the kube-proxy service, which runs on each
// node and manages forwarding of traffic addressed to the virtual IP addresses
// (VIPs) of the cluster’s Kubernetes Service objects to the appropriate
// backend pods.
//
// More info: https://kubernetes.io/docs/reference/command-line-tools-reference/kube-proxy/
type KubeProxyOpts struct {
	// Which proxy mode to use: 'userspace' (older) or 'iptables' (faster) or
	// 'ipvs' or 'kernelspace' (windows). If blank, use the best-available
	// proxy (currently iptables). If the iptables proxy is selected,
	// regardless of how, but the system's kernel or iptables versions are
	// insufficient, this always falls back to the userspace proxy.
	Mode string `json:"mode,omitempty" protobuf:"bytes,1,opt,name=mode" kube.env:"KUBE_PROXY_MODE"`
}

// KeystoneOpts is the container for all settings related to authentication with
// the OpenStack Keystone identity service.
// Note(erwin): this does not seem to be used by pf9-kube, so this could be left out.
//
// More info: https://docs.openstack.org/keystone/latest/
type KeystoneOpts struct {
	// Enabled signals whether Keystone should be used for authentication.
	Enabled bool `json:"enabled,omitempty" protobuf:"bool,1,opt,name=enabled" kube.env:"KEYSTONE_ENABLED"`

	// Domain contains the DNS name of the Keystone service.
	Domain string `json:"domain,omitempty" protobuf:"bytes,2,opt,name=domain" kube.env:"KEYSTONE_DOMAIN"`

	// AuthURL should contain the base URL to the Keystone service.
	AuthURL string `json:"authURL,omitempty" protobuf:"bytes,3,opt,name=authURL" kube.env:"OS_AUTH_URL"`

	Password          string `json:"password,omitempty" protobuf:"bytes,4,opt,name=password" kube.env:"OS_PASSWORD"`
	Username          string `json:"username,omitempty" protobuf:"bytes,5,opt,name=username" kube.env:"OS_USERNAME"`
	ProjectDomainName string `json:"projectDomainName,omitempty" protobuf:"bytes,6,opt,name=projectDomainName" kube.env:"OS_PROJECT_DOMAIN_NAME"`
	ProjectName       string `json:"projectName,omitempty" protobuf:"bytes,7,opt,name=projectName" kube.env:"OS_PROJECT_NAME"`
	Region            string `json:"region,omitempty" protobuf:"bytes,8,opt,name=region" kube.env:"OS_REGION"`
	UserDomainName    string `json:"userDomainName,omitempty" protobuf:"bytes,9,opt,name=userDomainName" kube.env:"OS_USER_DOMAIN_NAME"`
}

// KeepalivedOpts contains the settings to configure keepalived, which is used
// to handle failovers of a virtual IP in a multi master deployment on bare OS
// cluster.
//
// More info: https://www.keepalived.org/manpage.html
type KeepalivedOpts struct {
	// Enabled signals whether keepalived should be configured on the cluster.
	Enabled bool `json:"enabled,omitempty" protobuf:"bool,1,opt,name=enabled" kube.env:"MASTER_VIP_ENABLED"`

	MasterVIPInterface string `json:"masterVIPInterface,omitempty" protobuf:"bytes,2,opt,name=masterVIPInterface" kube.env:"MASTER_VIP_IFACE"`

	// MasterVIPPriority is for electing MASTER, highest priority wins.
	// Note(erwin): seems to be unused.
	MasterVIPPriority string `json:"masterVIPPriority,omitempty" protobuf:"bytes,3,opt,name=masterVIPPriority" kube.env:"MASTER_VIP_PRIORITY"`

	// MasterVIPRouterID is an arbitrary unique number from 1 to 255 used to
	// differentiate multiple instances of vrrpd running on the same NIC
	// (and hence same socket).
	MasterVIPRouterID string `json:"masterVIPRouterID,omitempty" protobuf:"bytes,4,opt,name=masterVIPRouterID" kube.env:"MASTER_VIP_VROUTER_ID"`
}

// KubeApiserverOpts contains the settings for the kube-apiserver. The most
// relevant parameters of the kube-apiserver are explicitly defined, all other
// flags should be defined using the ExtraArgs field.
//
// More info: https://kubernetes.io/docs/reference/command-line-tools-reference/kube-apiserver
type KubeApiserverOpts struct {
	// StorageBackend defines the storage backend for persistence.
	// Options: 'etcd3'
	// This is equivalent to the --storage-backend flag for kube-apiserver.
	StorageBackend string `json:"storageBackend,omitempty" protobuf:"bytes,1,opt,name=storageBackend" kube.env:"APISERVER_STORAGE_BACKEND"`

	// Privileged allows this cluster to run privileged containers. This is
	// required for Calico CNI and CSI.
	// This is equivalent to the --allow-privileged flag for kube-apiserver.
	//
	// More info: https://docs.docker.com/engine/reference/run/#runtime-privilege-and-linux-capabilities
	Privileged bool `json:"privileged,omitempty" protobuf:"bool,2,opt,name=privileged" kube.env:"PRIVILEGED"`

	// Port specifies the HTTPS port on which the kube-apiserver should be
	// served.
	// This is equivalent to the --secure-port flag for kube-apiserver.
	Port int32 `json:"port,omitempty" protobuf:"varint,3,opt,name=port" kube.env:"K8S_API_PORT"`

	// Authz indicates if authorization should be enabled on the kube-apiserver.
	// This option has been deprecated since authorization has been enabled by
	// default since Kubernetes 1.10. This field is no longer used by scripts.
	Authz bool `json:"authz,omitempty" protobuf:"bool,4,opt,name=authz" kube.env:"AUTHZ_ENABLED"`

	// RuntimeConfig is comma-separated list of key=value pairs.
	//
	// Equivalent to the --runtime-config of kube-apiserver.
	RuntimeConfig string `json:"runtimeConfig,omitempty" protobuf:"bytes,5,opt,name=runtimeConfig" kube.env:"RUNTIME_CONFIG"`

	// ExtraArgs is a catch-all for all flags of kube-apiserver that are not
	// present as an explicit field. These flags should be seperated with a ",".
	// For example: --skip-log-headers,--tls-min-version=VersionTLS11
	ExtraArgs string `json:"extraArgs,omitempty" protobuf:"bytes,6,opt,name=extraArgs" kube.env:"API_SERVER_FLAGS"`
}

// KubeSchedulerOpts contains the settings for the kube-scheduler. The most
// relevant parameters of the kube-scheduler are explicitly defined, all other
// flags should be defined using the ExtraArgs field.
//
// More info: https://kubernetes.io/docs/reference/command-line-tools-reference/kube-scheduler
type KubeSchedulerOpts struct {
	// ExtraArgs is a catch-all for all flags of kube-scheduler that are not
	// present as an explicit field. These flags should be separated with a ",".
	// For example: --skip-log-headers,--tls-min-version=VersionTLS11
	ExtraArgs string `json:"extraArgs,omitempty" protobuf:"bytes,1,opt,name=extraArgs" kube.env:"SCHEDULER_FLAGS"`
}

// KubeControllerManagerOpts contains the settings for the kube-controller-manager. The most
// relevant parameters of the kube-controller-manager are explicitly defined, all other
// flags should be defined using the ExtraArgs field.
//
// More info: https://kubernetes.io/docs/reference/command-line-tools-reference/kube-controller-manager
type KubeControllerManagerOpts struct {
	// ExtraArgs is a catch-all for all flags of kube-controller-manager that are not
	// present as an explicit field. These flags should be separated with a ",".
	// For example: --skip-log-headers,--tls-min-version=VersionTLS11
	ExtraArgs string `json:"extraArgs,omitempty" protobuf:"bytes,1,opt,name=extraArgs" kube.env:"CONTROLLER_MANAGER_FLAGS"`
}

// KubeClusterOpts contains the cluster-wide configuration. These settings
// include the control plane (apiserver, scheduler, and controller-manager),
// networking (CNI, KubeProxy), and the ingress load-balancing (MetalLB).
type KubeClusterOpts struct {
	// Scheduler contains the settings for the kube-scheduler. The most
	// relevant parameters of the kube-scheduler are explicitly defined, all other
	// flags should be defined using the ExtraArgs field.
	//
	// More info: https://kubernetes.io/docs/reference/command-line-tools-reference/kube-scheduler
	Scheduler KubeSchedulerOpts `json:"scheduler,omitempty" protobuf:"bytes,1,name=scheduler"`

	// ControllerManager contains the settings for the kube-controller-manager. The most
	// relevant parameters of the kube-controller-manager are explicitly defined, all other
	// flags should be defined using the ExtraArgs field.
	//
	// More info: https://kubernetes.io/docs/reference/command-line-tools-reference/kube-controller-manager
	ControllerManager KubeControllerManagerOpts `json:"controllerManager,omitempty" protobuf:"bytes,2,name=controllerManager"`

	// Apiserver contains the settings for the kube-apiserver. The most
	// relevant parameters of the kube-apiserver are explicitly defined, all other
	// flags should be defined using the ExtraArgs field.
	//
	// More info: https://kubernetes.io/docs/reference/command-line-tools-reference/kube-apiserver
	Apiserver KubeApiserverOpts `json:"apiserver,omitempty" protobuf:"bytes,3,name=apiserver"`

	// CNI contains the CNI configuration, which includes general options,
	// as well as, CNI-specific settings. The current supported CNI plugins in
	// PMK are Calico, and Flannel, one of which should be non-empty based on
	// the NetworkPlugin field (calico or flannel, respectively).
	//
	// More info: https://kubernetes.io/docs/concepts/extend-kubernetes/compute-storage-net/network-plugins/
	CNI CNIOpts `json:"cni,omitempty" protobuf:"bytes,4,opt,name=cni"`

	// Addons is an aggregation of all supported addons, which include Luigi,
	// Cluster AutoScaler (CAS), App Catalog, and KubeVirt.
	Addons AddonsOpts `json:"addons,omitempty" protobuf:"bytes,5,opt,name=addons"`

	// KubeProxy contains settings for the kube-proxy service, which runs on
	// each node and manages forwarding of traffic addressed to the virtual IP
	// addresses (VIPs) of the cluster’s Kubernetes Service objects to the
	// appropriate backend pods.
	//
	// More info: https://kubernetes.io/docs/reference/command-line-tools-reference/kube-proxy/
	KubeProxy KubeProxyOpts `json:"kubeproxy,omitempty" protobuf:"bytes,6,opt,name=kubeproxy"`

	// MetalLB specifies the options for the MetalLB addon which provides
	// support for load-balancer services.
	//
	// More info: https://metallb.universe.tf
	MetalLB MetalLBOpts `json:"metallb,omitempty" protobuf:"bytes,7,opt,name=metallb"`

	// UseHostname specifies the option for registering the bare OS node using
	// hostname (instead of the IP) in the PF9 managed k8s cluster. This option is only applicable to IPv4 hosts.
	// This option is ignored when deploying clusters on IPv6 enabled hosts.
	UseHostname bool `json:"useHostname,omitempty" protobuf:"bool,8,opt,name=useHostname" kube.env:"USE_HOSTNAME"`

	// Version of kubernetes to deploy on a node
	KubernetesVersion string `json:"kubernetesVersion,omitempty" protobuf:"bytes,9,opt,name=kubernetesVersion"`
}

// CatapultMonitoringOpts contains the settings to configure catapult monitoring
type CatapultMonitoringOpts struct {
	// Enabled signals whether catapult monitoring should be configured on the cluster.
	Enabled bool `json:"enabled,omitempty" protobuf:"bool,1,opt,name=enabled" kube.env:"CATAPULT_ENABLED"`
}

// PF9Opts contains miscellaneous configuration, mostly related to PF9 services.
type PF9Opts struct {
	// VaultToken is the token used to generate the certificates with for the
	// cluster.
	VaultToken string `json:"vaultToken,omitempty" protobuf:"bytes,1,opt,name=vaultToken" kube.env:"VAULT_TOKEN"`

	// Masterless, if set, instructs the cluster to run a proxy to a remote
	// apiserver, rather than running the apiserver in-cluster.
	Masterless bool `json:"masterless,omitempty" protobuf:"bool,2,opt,name=masterless" kube.env:"MASTERLESS_ENABLED"`

	// BouncerSlowReqWebhook is a field to enable code instrumentation to be
	// able to detect keystone slowness in some environments. It is unclear
	// whether this is still used.
	//
	// More info: https://platform9.atlassian.net/browse/INF-764
	BouncerSlowReqWebhook string `json:"bouncerSlowReqWebhook,omitempty" protobuf:"bytes,3,opt,name=bouncerSlowReqWebhook" kube.env:"BOUNCER_SLOW_REQUEST_WEBHOOK"`

	// CloudProviderType contains the cloud provider that should be used to
	// provision and bootstrap this Host with. For example, based on the value
	// of this field, the manifests used to bootstrap the control plane are
	// chosen.
	//
	// Options: aws|azure|local|openstack
	CloudProviderType string `json:"cloudProviderType,omitempty" protobuf:"bytes,4,opt,name=cloudProviderType" kube.env:"CLOUD_PROVIDER_TYPE"`

	// ClusterID specifies the ID of the Cluster that this Host belongs to.
	ClusterID string `json:"clusterID,omitempty" protobuf:"bytes,5,opt,name=clusterID" kube.env:"CLUSTER_ID"`

	// ClusterProjectID specifies the ID of the project that the Cluster and
	// thereby this Host belong too.
	ClusterProjectID string `json:"clusterProjectID,omitempty" protobuf:"bytes,6,opt,name=clusterProjectID" kube.env:"CLUSTER_PROJECT_ID"`

	// Debug will increase the verbosity of logging if set.
	Debug bool `json:"debug,omitempty" protobuf:"bool,7,opt,name=debug" kube.env:"DEBUG"`

	// KubeServiceState is the desired state of this Host in relation to
	// the target cluster. If set to "true" the Host should be added to the
	// cluster as a Node; if set to "false" the Host should be removed from the
	// cluster as a Node; and, if set to another value (commonly "" or "ignore")
	// the Host should simply be ignored and left in whatever state it is.
	KubeServiceState string `json:"kubeServiceState,omitempty" protobuf:"bytes,8,opt,name=kubeServiceState" kube.env:"KUBE_SERVICE_STATE"`

	// ExternalDNSName specifies the externally-accessible DNS name for the
	// control plane of the cluster.
	ExternalDNSName string `json:"externalDNSName,omitempty" protobuf:"bytes,9,opt,name=externalDNSName" kube.env:"EXTERNAL_DNS_NAME"`

	// ClusterRole specifies the role that this Host should take within the
	// cluster. Options:
	// - master:	turn the host into a Kubernetes master node.
	// - worker:	turn the host into a Kubernetes worker node.
	// - none:		do not turn the host into a Kubernetes node.
	ClusterRole string `json:"clusterRole,omitempty" protobuf:"bytes,10,opt,name=clusterRole" kube.env:"ROLE"`

	Keepalived KeepalivedOpts `json:"keepalived,omitempty" protobuf:"bytes,11,opt,name=keepalived"`

	// Keystone is the container for all settings related to authentication with
	// the OpenStack Keystone identity service.
	//
	// More info: https://docs.openstack.org/keystone/latest/
	Keystone KeystoneOpts `json:"keystone,omitempty" protobuf:"bytes,12,opt,name=keystone"`

	// MasterIP either contains a FDQN (in case of a public cloud), or the
	// primary IP of the master node (in case of a single master), or the
	// primary IP of the last added master node (in case of multi-master
	// without keepalived) or the Virtual IP (if keepalived is enabled).
	MasterIP string `json:"masterIP,omitempty" protobuf:"bytes,13,opt,name=masterIP" kube.env:"MASTER_IP"`

	// ClusterName specifies the name of the cluster
	ClusterName string `json:"clusterName,omitempty" protobuf:"bytes,14,opt,name=clusterName" kube.env:"CLUSTER_NAME"`

	// CatapultMonitoring contains catapult monitoring configs
	CatapultMonitoring CatapultMonitoringOpts `json:"catapultMonitoring,omitempty" protobuf:"bytes,15,opt,name=catapultMonitoring"`

	// isAirgapped specifies whether cluster is running in airgapped or SaaS env
	IsAirgapped bool `json:"isAirgapped,omitempty" protobuf:"bool,16,opt,name=isAirgapped" kube.env:"IS_AIRGAPPED"`

    MasterIPv6 string `json:"masterIPv6,omitempty" protobuf:"bytes,17,opt,name=masterIPv6" kube.env:"MASTER_IPV6"`
}

// NodeletStatus contains information about the Nodelet process
// itself (rather than about the Host).
type NodeletStatus struct {
	// Version is the version of Nodelet that is running on the Host.
	Version string `json:"version,omitempty" protobuf:"bytes,1,opt,name=version"`
}

// HostPhase contains the status of a single phase script.
type HostPhase struct {
	// Name provides a human-readable name for this phase. These names can
	// contain special characters and spaces.
	//
	// Example: "Configure and start auth web hook / pf9-bouncer"
	Name string `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`

	// Order is a unique number that is used to determine the order of the
	// phases. The phases are ordered ascendingly based on this field.
	Order int32 `json:"order,omitempty" protobuf:"varint,2,opt,name=order"`

	// StartedAt indicates when this phase started running.
	StartedAt metav1.Time `json:"startedAt,omitempty" protobuf:"bytes,3,opt,name=startedAt"`

	// Operation describes the type of operation that the phase is currently
	// performing. Options: start|stop|status.
	Operation string `json:"operation,omitempty" protobuf:"bytes,4,opt,name=operation"`

	// Status describes the current state of the phase.
	//
	// Options:
	// - not-started: state when no operation has been performed.
	// - running:     state when start operation was successful.
	// - stopped:     state when stop operation was successful.
	// - failed:      state when start operation has failed.
	// - executing:   state when an operation is being performed on a phase.
	Status string `json:"status,omitempty" protobuf:"bytes,5,opt,name=status"`

	// Message is a human readable description indicating details about why the
	// phase is in this status.
	//
	// When the status is an error, the Message contains the error message.
	Message string `json:"message,omitempty" protobuf:"bytes,6,opt,name=message"`
}
