package sunpike

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

	// Conditions defines current service state of the Cluster.
	// +optional
	Conditions Conditions `json:"conditions,omitempty" protobuf:"bytes,30,opt,name=conditions"`

	// ObservedGeneration is the latest generation observed by the controller.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty" protobuf:"varint,31,opt,name=observedGeneration"`

	// KubeVersion specifies the current version of Kubernetes running
	// on the corresponding Node. This is meant to be a means of bubbling
	// up status from the Node to the Machine.
	// It is entirely optional, but useful for end-user UX if it’s present.
	// +optional
	KubeVersion string `json:"kubeVersion,omitempty" protobuf:"bytes,32,opt,name=kubeVersion"`

	// PrimaryIP contains the primary IP of this Host.
	// +optional
	PrimaryIP string `json:"primaryIP,omitempty" protobuf:"bytes,33,opt,name=primaryIP"`
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

	MasterIPv6 string `json:"masterIPv6,omitempty" protobuf:"bytes,14,opt,name=masterIPv6" kube.env:"MASTER_IPV6"`
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
