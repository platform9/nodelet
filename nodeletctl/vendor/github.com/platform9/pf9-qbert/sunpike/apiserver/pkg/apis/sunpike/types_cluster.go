package sunpike

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ClusterPhase is a label for the condition of a cluster at the current time.
type ClusterPhase string

// These are the valid statuses of pods.
const (
	// ClusterPhasePending means the cluster has been accepted by the system,
	// but is still being deployed or configured.
	ClusterPhasePending ClusterPhase = "Pending"
	// ClusterPhaseRunning means the cluster has been deployed succesfully and
	// is ready to be interacted with.
	ClusterPhaseRunning ClusterPhase = "Running"
	// ClusterPhaseTerminating means that the cluster has been scheduled for
	// deletion, but still has resources awaiting clean up.
	ClusterPhaseTerminating ClusterPhase = "Terminating"
	// ClusterPhaseFailed means that the cluster is in an error state and is
	// likely not operational. Manual intervention might be needed to remediate
	// the situation.
	ClusterPhaseFailed ClusterPhase = "Failing"
	// ClusterPhaseUnknown means that for some reason the state of the cluster
	// could not be determined.
	ClusterPhaseUnknown ClusterPhase = ""
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Cluster is a representation of the configuration and status of a Kubernetes cluster.
type Cluster struct {
	metav1.TypeMeta `json:",inline" protobuf:"bytes,4,opt,name=typeMeta"`

	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Specification of the desired behavior of the Cluster.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status
	Spec ClusterSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`

	// Most recently observed status of the Cluster.
	// This data may not be up to date.
	// Populated by the system.
	// Read-only.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status
	Status ClusterStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterList is a list of Cluster objects.
type ClusterList struct {
	metav1.TypeMeta `json:",inline" protobuf:"bytes,3,opt,name=typeMeta"`

	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Items contains a list of Clusters.
	Items []Cluster `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// ClusterSpec contains the specification of the desired configuration of the Cluster.
type ClusterSpec struct {
	Debug bool `json:"debug,omitempty" protobuf:"bool,1,opt,name=debug" kube.env:"DEBUG"`

	CloudProviderID string `json:"cloudProviderID,omitempty" protobuf:"bytes,2,opt,name=cloudProviderID"`

	// ProjectID specifies the ID of the project that the Cluster.
	ProjectID string `json:"projectID,omitempty" protobuf:"bytes,3,opt,name=projectID" kube.env:"CLUSTER_PROJECT_ID"`

	// KubeVersion is the target version of the control plane.
	// +optional
	KubeVersion string `json:"kubeVersion,omitempty" protobuf:"bytes,4,opt,name=kubeVersion"`

	// Cluster network configuration.
	//
	// [qbert] clusters.containersCidr + clusters.servicesCidr
	// +optional
	ClusterNetwork ClusterNetwork `json:"clusterNetwork,omitempty" protobuf:"bytes,5,opt,name=clusterNetwork"`

	// KubeProxy contains settings for the kube-proxy service, which runs on
	// each node and manages forwarding of traffic addressed to the virtual IP
	// addresses (VIPs) of the cluster’s Kubernetes Service objects to the
	// appropriate backend pods.
	//
	// More info: https://kubernetes.io/docs/reference/command-line-tools-reference/kube-proxy/
	KubeProxy KubeProxyOpts `json:"kubeproxy,omitempty" protobuf:"bytes,6,opt,name=kubeproxy"`

	// KubeletOpts contain Kubelet-specific configuration.
	//
	// See more: https://kubernetes.io/docs/reference/command-line-tools-reference/kubelet/
	Kubelet KubeletOpts `json:"kubelet,omitempty" protobuf:"bytes,7,opt,name=kubelet"`

	// KubeletOpts contain Kubelet-specific configuration.
	//
	// See more: https://kubernetes.io/docs/reference/command-line-tools-reference/kubelet/
	ContainerRuntime ContainerRuntime `json:"containerRuntime,omitempty" protobuf:"bytes,8,opt,name=containerRuntime"`

	// LoadBalancer contains all specification related to ingress and the LoadBalancer service type.
	LoadBalancer LoadBalancer `json:"loadBalancer,omitempty" protobuf:"bytes,9,opt,name=loadBalancer"`

	// Based on the naming in kube-apiserver, where it calls etcd the 'storage-backend'.
	// See: https://kubernetes.io/docs/reference/command-line-tools-reference/kube-apiserver/
	StorageBackend StorageBackend `json:"storageBackend,omitempty" protobuf:"bytes,10,opt,name=storageBackend"`

	// Auth contains all configuration related to cluster authentication and authorization.
	Auth Auth `json:"auth,omitempty" protobuf:"bytes,11,opt,name=auth"`

	// HA contians configuration related to highly-available API server support
	HA HA `json:"ha,omitempty" protobuf:"bytes,12,opt,name=ha"`

	// Scheduler contains the settings for the kube-scheduler. The most
	// relevant parameters of the kube-scheduler are explicitly defined, all other
	// flags should be defined using the ExtraArgs field.
	//
	// More info: https://kubernetes.io/docs/reference/command-line-tools-reference/kube-scheduler
	Scheduler KubeSchedulerOpts `json:"scheduler,omitempty" protobuf:"bytes,13,name=scheduler"`

	// ControllerManager contains the settings for the kube-controller-manager. The most
	// relevant parameters of the kube-controller-manager are explicitly defined, all other
	// flags should be defined using the ExtraArgs field.
	//
	// More info: https://kubernetes.io/docs/reference/command-line-tools-reference/kube-controller-manager
	ControllerManager KubeControllerManagerOpts `json:"controllerManager,omitempty" protobuf:"bytes,14,name=controllerManager"`

	// Apiserver contains the settings for the kube-apiserver. The most
	// relevant parameters of the kube-apiserver are explicitly defined, all other
	// flags should be defined using the ExtraArgs field.
	//
	// More info: https://kubernetes.io/docs/reference/command-line-tools-reference/kube-apiserver
	Apiserver KubeApiserverOpts `json:"apiserver,omitempty" protobuf:"bytes,15,name=apiserver"`

	// ControlPlaneEndpoint represents a predefined endpoint to be used to
	// communicate with the control plane.
	//
	// [qbert] clusters.masterIp/externalDNSName + clusters.k8sApiPort
	// +optional
	ControlPlaneEndpoint APIEndpoint `json:"controlPlaneEndpoint,omitempty" protobuf:"bytes,23,opt,name=controlPlaneEndpoint"`

	// CNI contains the CNI configuration, which includes general options,
	// as well as, CNI-specific settings. The current supported CNI plugins in
	// PMK are Calico, and Flannel, one of which should be non-empty based on
	// the NetworkPlugin field (calico or flannel, respectively).
	//
	// More info: https://kubernetes.io/docs/concepts/extend-kubernetes/compute-storage-net/network-plugins/
	CNI CNIOpts `json:"cni,omitempty" protobuf:"bytes,17,opt,name=cni"`

	// Addons is an aggregation of all supported addons, which include Luigi,
	// Cluster AutoScaler (CAS), App Catalog, and KubeVirt.
	Addons AddonsOpts `json:"addons,omitempty" protobuf:"bytes,18,opt,name=addons"`

	// AllowWorkloadsOnMasters signals whether regular workloads are allowed to
	// be run on master nodes.
	AllowWorkloadsOnMasters bool `json:"allowWorkloadsOnMasters,omitempty" protobuf:"bool,19,opt,name=allowWorkloadsOnMasters" kube.env:"ALLOW_WORKLOADS_ON_MASTER"`

	// VaultToken is the token used to generate the certificates with for the
	// cluster.
	VaultToken string `json:"vaultToken,omitempty" protobuf:"bytes,20,opt,name=vaultToken" kube.env:"VAULT_TOKEN"`

	// PF9 contains all miscellanuous configuration mostly related to specific
	// Platform9 services.
	PF9 PF9 `json:"pf9,omitempty" protobuf:"bytes,21,opt,name=pf9"`

	// DisplayName provide human readable name to the cluster.
	// +optional
	DisplayName string `json:"displayName" protobuf:"bytes,22,opt,name=displayName"`

	// AWS contains all AWS-specific configuration for the cluster.
	AWS *AWSCluster `json:"aws,omitempty" protobuf:"bytes,24,opt,name=aws"`

	// External defines that cluster is externally managed and PF9 is used as monitoring pane.
	// +optional
	External bool `json:"external" protobuf:"bytes,25,opt,name=external"`

	// EKS contains all EKS specific configuration for the cluster.
	// +optional
	EKS *EKSCluster `json:"eks,omitempty" protobuf:"bytes,26,opt,name=eks"`

	// DockerPrivateRegistry is the location of the docker private registry (if any) that hosts the PF9 container images
	DockerPrivateRegistry string `json:"dockerPrivateRegistry,omitempty" protobuf:"bytes,27,opt,name=dockerPrivateRegistry" kube.env:"DOCKER_PRIVATE_REGISTRY"`

	// QuayPrivateRegistry is the location of the quay private registry (if any) that hosts the PF9 container images
	QuayPrivateRegistry string `json:"quayPrivateRegistry,omitempty" protobuf:"bytes,28,opt,name=quayPrivateRegistry" kube.env:"QUAY_PRIVATE_REGISTRY"`

	// GCRPrivateRegistry is the location of the gcr private registry (if any) that hosts the PF9 container images
	GCRPrivateRegistry string `json:"gcrPrivateRegistry,omitempty" protobuf:"bytes,29,opt,name=gcrPrivateRegistry" kube.env:"GCR_PRIVATE_REGISTRY"`

	// K8SPrivateRegistry is the location of the k8s private registry (if any) that hosts the PF9 container images
	K8SPrivateRegistry string `json:"k8sPrivateRegistry,omitempty" protobuf:"bytes,30,opt,name=k8sPrivateRegistry" kube.env:"K8S_PRIVATE_REGISTRY"`

	// UseHostname specifies the option for registering the bare OS node using
	// hostname (instead of the IP) in the PF9 managed k8s cluster. This option is only applicable to IPv4 hosts.
	// This option is ignored when deploying clusters on IPv6 enabled hosts.
	UseHostname bool `json:"useHostname,omitempty" protobuf:"bool,31,opt,name=useHostname" kube.env:"USE_HOSTNAME"`

	// AKS contains all AKS specific configuration for the cluster.
	// +optional
	AKS *AKSCluster `json:"aks,omitempty" protobuf:"bytes,32,opt,name=aks"`

	// GKE contains all GKE specific configuration for the cluster.
	// +optional
	GKE *GKECluster `json:"gke,omitempty" protobuf:"bytes,33,opt,name=gke"`
}

// EKSCluster defines spec for the k8s cluster created
// and managed as part of AWS.
// More https://docs.aws.amazon.com/eks/latest/APIReference/eks-api.pdf#Welcome
type EKSCluster struct {
	// Region explains the region in which the EKSCluster is present.
	Region string `json:"region" protobuf:"bytes,1,opt,name=region"`

	// KubernetesVersion informs us about the kubernetes version on the cluster.
	KubernetesVersion string `json:"kubernetesVersion" protobuf:"bytes,2,opt,name=kubernetesVersion"`

	// EKSVersion is the internal eks version for a given k8s version.
	EKSVersion string `json:"eksVersion" protobuf:"bytes,3,opt,name=eksVersion"`

	// CreatedAt informs the time at which the cluster got created.
	// +optional
	CreatedAt metav1.Time `json:"createdAt,omitempty" protobuf:"bytes,4,opt,name=createdAt"`

	// Status tells us about the cluster status.
	// +optional
	Status string `json:"status,omitempty" protobuf:"bytes,5,opt,name=status"`

	// CA tells us the certificate authority data for the cluster
	// +optional
	CA string `json:"ca,omitempty" protobuf:"bytes,6,name=ca"`

	// The arn of the amazon IAM role that provides permissions to make API calls to AWS Resources.
	// +optional
	IAMRole string `json:"iamRole,omitempty" protobuf:"bytes,7,name=iamRole"`

	// The K8s Networking config for cluster.
	// +optional
	Network *EKSNetwork `json:"network,omitempty" protobuf:"bytes,8,name=network"`

	// Object representing logging configuration for resources in cluster.
	// +optional
	Logging *EKSLogging `json:"logging,omitempty" protobuf:"bytes,9,name=logging"`

	// Metadata to be applied to cluster for categorization and organisation.
	// +optional
	Tags map[string]string `json:"tags,omitempty" protobuf:"bytes,10,name=tags"`

	// Cluster managed node groups.
	// +optional
	NodeGroups []EKSNodeGroup `json:"nodegroups,omitempty" protobuf:"bytes,11,name=nodegroups"`
}

//EKSLogging provides information about the logging
//enabled in eks cluster.
type EKSLogging struct {
	// Logging information for resources in the cluster.
	// +optional
	EKSClusterLogging []EKSClusterLogging `json:"clusterLogging,omitempty" protobuf:"bytes,1,name=clusterLogging"`
}

//EKSClusterLogging lays out the structure encapsulating
//logging in eks clusters.
type EKSClusterLogging struct {
	// +optional
	Types []string `json:"types,omitempty" protobuf:"bytes,1,name=types"`
	// +optional
	Enabled bool `json:"enabled,omitempty" protobuf:"bytes,2,name=enabled"`
}

// EKSNetwork provides networking aspect of the
// EKSCluster.
type EKSNetwork struct {
	// +optional
	ContainerCIDR []string `json:"containerCidr,omitempty" protobuf:"bytes,1,name=containerCidr"`

	// +optional
	ServicesCIDR string `json:"servicesCidr,omitempty" protobuf:"bytes,2,name=servicesCidr"`

	// +optional
	VPC *AWSVPC `json:"vpc,omitempty" protobuf:"bytes,3,name=vpc"`
}

// AWSVPC provides information about the AWSVPC
// More info: https://docs.aws.amazon.com/vpc/latest/userguide/what-is-amazon-vpc.html
type AWSVPC struct {
	VPCID string `json:"vpcId" protobuf:"bytes,1,name=vpcId"`
	// +optional
	SecurityGroup []string `json:"securityGroup,omitempty" protobuf:"bytes,2,name=securityGroup"`

	// +optional
	PublicAccess bool `json:"publicAccess" protobuf:"bytes,3,name=publicAccess"`

	// +optional
	PrivateAccess bool `json:"privateAccess" protobuf:"bytes,4,name=privateAccess"`

	// +optional
	ClusterSecurityGroupID string `json:"clusterSecurityGroupId,omitempty" protobuf:"bytes,5,name=clusterSecurityGroupId"`

	// +optional
	Subnets []string `json:"subnets,omitempty" protobuf:"bytes,6,name=subnets"`
}

// EKSNodeGroup provides information about the instances used
// in EKS Cluster.
type EKSNodeGroup struct {
	// Name describes the name associated with the nodegroup.
	Name string `json:"name" protobuf:"bytes,1,name=name"`

	// ARN associated with the nodegroup.
	ARN string `json:"arn" protobuf:"bytes,2,name=arn"`

	// k8s version of the nodegroup.
	KubernetesVersion string `json:"kubernetesVersion" protobuf:"bytes,3,name=kubernetesVersion"`

	// Time at which the node group was created.
	// +optional
	CreatedAt metav1.Time `json:"createdAt,omitempty" protobuf:"bytes,4,name=createdAt"`

	// Last time at which the node group was updated.
	// +optional
	UpdatedAt metav1.Time `json:"updatedAt,omitempty" protobuf:"bytes,5,name=updatedAt"`

	// Status of the nodegroup.
	// +optional
	Status string `json:"status,omitempty" protobuf:"bytes,6,name=status"`

	// The capacity type of the nodegroup.
	// +optional
	CapacityType string `json:"capacityType,omitempty" protobuf:"bytes,7,name=capacityType"`

	// The types of instances in the nodegroup.
	// +optional
	InstanceTypes []string `json:"instanceTypes,omitempty" protobuf:"bytes,8,name=instanceTypes"`

	// The subnets for the autoscaling group that was associated with the nodegroup.
	// +optional
	Subnets []string `json:"subnets,omitempty" protobuf:"bytes,9,name=subnets"`

	// The type of the ami that was supplied in the configuration.
	// +optional
	AMI string `json:"ami,omitempty" protobuf:"bytes,10,name=ami"`

	// User specified tags on the nodegroups.
	// +optional
	Tags map[string]string `json:"tags,omitempty" protobuf:"bytes,11,name=tags"`

	// User specified labels on the nodegroups.
	// +optional
	Labels map[string]string `json:"labels,omitempty" protobuf:"bytes,12,name=labels"`

	// Identifier of the sshkey that provides access to the ec2 instances in nodegroups.
	// +optional
	SSHKey string `json:"sshKey,omitempty" protobuf:"bytes,13,name=sshKey"`

	// Scaling configuration associated with the nodegroup.
	// +optional
	ScalingConfig *AWSScalingConfig `json:"scalingConfig,omitempty" protobuf:"bytes,14,name=scalingConfig"`

	// The root device disk size for instances in nodegroups.
	// +optional
	DiskSizeInGB int32 `json:"diskSizeInGiB,omitempty" protobuf:"varint,15,name=diskSizeInGiB"`

	// The ARN of the IAM role associated with nodegroups.
	// +optional
	IAMRole string `json:"iamRole,omitempty" protobuf:"bytes,16,name=iamRole"`

	// The instances as part of the nodegroup.
	// +optional
	Instances []EC2Instance `json:"instances,omitempty" protobuf:"bytes,17,name=instances"`
}

// EC2Instance represents information about EC2 instances in AWS.
// More info:https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/Instances.html
type EC2Instance struct {
	// InstanceID represents identifier of instance.
	// +optional
	InstanceID string `json:"instanceId,omitempty" protobuf:"bytes,1,name=instanceId"`
	// The AZ in which the instance was created.
	// +optional
	AvailabilityZone string `json:"availabilityZone,omitempty" protobuf:"bytes,2,name=availabilityZone"`
	// The type of the instance.
	// +optional
	InstanceType string `json:"instanceType,omitempty" protobuf:"bytes,3,name=instanceType"`
	// The ntwork properties associated with the instance.
	// +optional
	Network *EC2InstanceNetwork `json:"network,omitempty" protobuf:"bytes,4,name=network"`
}

// EC2InstanceNetwork represents the network information for
// the EC2 instance.
type EC2InstanceNetwork struct {
	// +optional
	PrivateDNSName string `json:"privateDnsName,omitempty" protobuf:"bytes,1,name=privateDnsName"`

	// +optional
	PublicDNSName string `json:"publicDnsName,omitempty" protobuf:"bytes,2,name=publicDnsName"`

	// +optional
	PrivateIPAddress string `json:"privateIpAddress,omitempty" protobuf:"bytes,3,name=privateIpAddress"`

	// +optional
	PublicIPAddress string `json:"publicIpAddress,omitempty" protobuf:"bytes,4,name=publicIpAddress"`

	// +optional
	Subnet string `json:"subnet,omitempty" protobuf:"bytes,5,name=subnet"`

	// +optional
	VPCID string `json:"vpcId,omitempty" protobuf:"bytes,6,opt,name=vpcId"`
}

// AWSScalingConfig provides information regarding
// scaling of the nodegroup
// MoreInfo: https://docs.aws.amazon.com/autoscaling/ec2/userguide/asg-capacity-limits.html
type AWSScalingConfig struct {
	// +optional
	MinSize int32 `json:"minSize" protobuf:"varint,1,name=minSize"`

	// +optional
	MaxSize int32 `json:"maxSize" protobuf:"varint,2,name=maxsize"`

	// +optional
	DesiredSize int32 `json:"desiredSize" protobuf:"varint,3,name=desiredSize"`
}

// AKSCluster represents a managed AKS cluster
// More info: https://docs.microsoft.com/en-us/rest/api/aks/managedclusters/get#managedcluster
type AKSCluster struct {
	// Location explains the region in which the AKSCluster is present.
	Location string `json:"location" protobuf:"bytes,1,opt,name=location"`

	// KubernetesVersion informs us about the kubernetes version on the cluster.
	KubernetesVersion string `json:"kubernetesVersion" protobuf:"bytes,2,opt,name=kubernetesVersion"`

	// Resource type
	Type string `json:"type" protobuf:"bytes,3,opt,name=type"`

	// The current deployment or provisioning state, which only appears in the response.
	ProvisioningState string `json:"provisioningState" protobuf:"bytes,4,opt,name=provisioningState"`

	// Describes the Power State of the cluster
	PowerState string `json:"powerState" protobuf:"bytes,5,opt,name=powerState"`

	// Whether to enable Kubernetes Role-Based Access Control
	// +optional
	EnableRBAC bool `json:"enableRBAC,omitempty" protobuf:"bytes,6,opt,name=enableRBAC"`

	// The max number of agent pools for the managed cluster
	// +optional
	MaxAgentPools int32 `json:"maxAgentPools,omitempty" protobuf:"bytes,7,opt,name=maxAgentPools"`

	// Name of the resource group containing agent pool nodes
	// +optional
	NodeResourceGroup string `json:"nodeResourceGroup,omitempty" protobuf:"bytes,8,opt,name=nodeResourceGroup"`

	// Cluster network details
	// +optional
	Network *AKSNetwork `json:"network,omitempty" protobuf:"bytes,9,opt,name=network"`

	// Array of agent pools associated with the cluster
	// +optional
	AgentPools []AKSAgentPool `json:"agentPools,omitempty" protobuf:"bytes,10,opt,name=agentPools"`

	// Array of Azure VM instances that make up the AKS cluster nodes
	// +optional
	Instances []AKSInstance `json:"instances,omitempty" protobuf:"bytes,11,opt,name=instances"`

	// ServicePrincipalClientID is the client id of the service principal associated with the cluster
	// +optional
	ServicePrincipalClientID string `json:"servicePrincipalClientID,omitempty" protobuf:"bytes,12,opt,name=servicePrincipalClientID"`

	// EnablePrivateCluster denotes whether a cluster is private, i.e. not accessible over the public network
	// +optional
	EnablePrivateCluster *bool `json:"enablePrivateCluster,omitempty" protobuf:"bytes,13,opt,name=enablePrivateCluster"`

	// DNSPrefix is the prefix specified while creating the cluster
	// +optional
	DNSPrefix string `json:"dnsPrefix,omitempty" protobuf:"bytes,14,opt,name=dnsPrefix"`

	// Tags is a map of all the resource tags associated with the cluster
	// +optional
	Tags map[string]string `json:"tags,omitempty" protobuf:"bytes,15,opt,name=tags"`

	// FQDN is the cluster FQDN that can be used to connect with the K8s API server
	// +optional
	FQDN string `json:"fqdn,omitempty" protobuf:"bytes,16,opt,name=fqdn"`
}

type AKSNetwork struct {
	// Network plugin used for building Kubernetes network
	// +optional
	Plugin string `json:"plugin,omitempty" protobuf:"bytes,1,opt,name=plugin"`

	// Network policy used for building Kubernetes network
	// +optional
	Policy string `json:"policy,omitempty" protobuf:"bytes,2,opt,name=policy"`

	// A CIDR notation IP range from which to assign service cluster IPs. It must not overlap with any Subnet IP ranges
	// +optional
	ServiceCIDR string `json:"serviceCIDR,omitempty" protobuf:"bytes,3,opt,name=serviceCIDR"`

	// A CIDR notation IP range assigned to the Docker bridge network
	// +optional
	ContainerCIDR string `json:"containerCIDR,omitempty" protobuf:"bytes,4,opt,name=containerCIDR"`

	// An IP address assigned to the Kubernetes DNS service. It must be within the Kubernetes service
	// address range specified in serviceCidr
	// +optional
	DNSServiceIP string `json:"dnsServiceIP,omitempty" protobuf:"bytes,5,opt,name=dnsServiceIP"`

	// The outbound (egress) routing method
	// +optional
	OutboundType string `json:"outboundType,omitempty" protobuf:"bytes,6,opt,name=outboundType"`

	// LoadBalancerSKU is the load balance SKU for the cluster
	// +optional
	LoadBalancerSKU string `json:"loadBalancerSKU,omitempty" protobuf:"bytes,7,opt,name=loadBalancerSKU"`

	// LoadBalancerProfile is the profile of the managed cluster load balancer
	// +optional
	LoadBalancerProfile *AKSLoadBalancerProfile `json:"loadBalancerProfile,omitempty" protobuf:"bytes,8,opt,name=loadBalancerProfile"`
}

type AKSLoadBalancerProfile struct {
	// AllocatedOutboundPorts is the desired number of allocated SNAT ports per VM. Allowed values must be in the range of 0 to 64000 (inclusive). The default value is 0 which results in Azure dynamically allocating ports.
	// +optional
	AllocatedOutboundPorts int32 `json:"AllocatedOutboundPorts,omitempty" protobuf:"bytes,1,opt,name=AllocatedOutboundPorts"`

	// ManagedOutboundIPs are the desired managed outbound IPs for the cluster load balancer
	// +optional
	ManagedOutboundIPs int32 `json:"managedOutboundIPs,omitempty" protobuf:"bytes,2,opt,name=managedOutboundIPs"`

	// EffectiveOutboundIPs are the effective outbound IP resources of the cluster load balancer
	// +optional
	EffectiveOutboundIPs []string `json:"effectiveOutboundIPs,omitempty" protobuf:"bytes,3,opt,name=effectiveOutboundIPs"`

	// OutboundIPs are the desired outbound IP resources for the cluster load balancer
	// +optional
	OutboundIPs []string `json:"outboundIPs,omitempty" protobuf:"bytes,4,opt,name=outboundIPs"`
	// +optional

	// OutboundIPPrefixes are the desired outbound IP Prefix resources for the cluster load balancer
	// +optional
	OutboundIPPrefixes []string `json:"outboundIPPrefixes,omitempty" protobuf:"bytes,5,opt,name=outboundIPPrefixes"`
}

type AKSAgentPool struct {
	// Unique name of the agent pool profile in the context of the subscription and resource group
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`

	// Number of agents (VMs) to host docker containers. Allowed values must be in the range of 0
	// to 100 (inclusive) for user pools and in the range of 1 to 100 (inclusive) for system pools.
	// The default value is 1
	Count int32 `json:"count" protobuf:"bytes,2,opt,name=count"`

	// Size of agent VMs
	// +optional
	VMSize string `json:"vmSize,omitempty" protobuf:"bytes,3,opt,name=vmSize"`

	// OS Disk Size in GB to be used to specify the disk size for every machine in this
	// master/agent pool. If you specify 0, it will apply the default osDisk size according
	// to the vmSize specified
	// +optional
	OSDiskSizeGB int32 `json:"osDiskSizeGB,omitempty" protobuf:"bytes,4,opt,name=osDiskSizeGB"`

	// OS disk type to be used for machines in a given agent pool. Allowed values are
	// 'Ephemeral' and 'Managed'. If unspecified, defaults to 'Ephemeral' when the VM
	// supports ephemeral OS and has a cache disk larger than the requested OSDiskSizeGB.
	// Otherwise, defaults to 'Managed'. May not be changed after creation
	// +optional
	OSDiskType string `json:"osDiskType,omitempty" protobuf:"bytes,5,opt,name=osDiskType"`

	// Maximum number of pods that can run on a node
	// +optional
	MaxPods int32 `json:"maxPods,omitempty" protobuf:"bytes,6,opt,name=maxPods"`

	// AgentPoolType represents types of an agent pool
	// +optional
	Type string `json:"type,omitempty" protobuf:"bytes,7,opt,name=type"`

	// Availability zones for nodes. Must use VirtualMachineScaleSets AgentPoolType
	// +optional
	AvailabilityZone []string `json:"availabilityZones,omitempty" protobuf:"bytes,8,opt,name=availabilityZones"`

	// The current deployment or provisioning state, which only appears in the response
	ProvisioningState string `json:"provisioningState" protobuf:"bytes,9,opt,name=provisioningState"`

	// Describes whether the Agent Pool is Running or Stopped
	PowerState string `json:"powerState" protobuf:"bytes,10,opt,name=powerState"`

	// Version of orchestrator specified when creating the managed cluster
	KubernetesVersion string `json:"kubernetesVersion" protobuf:"bytes,11,opt,name=kubernetesVersion"`

	// Agent pool node labels to be persisted across all nodes in agent pool
	// +optional
	Labels map[string]string `json:"labels,omitempty" protobuf:"bytes,12,opt,name=labels"`

	// Represents mode of an agent pool
	// +optional
	Mode string `json:"mode,omitempty" protobuf:"bytes,13,opt,name=mode"`

	// OsType to be used to specify os type. Choose from Linux and Windows. Default to Linux
	// +optional
	OSType string `json:"osType,omitempty" protobuf:"bytes,14,opt,name=osType"`

	// NodeImageVersion is the version of node image
	// +optional
	NodeImageVersion string `json:"nodeImageVersion,omitempty" protobuf:"bytes,15,opt,name=nodeImageVersion"`

	// VnetSubnetID of the agent pool
	// +optional
	VnetSubnetID string `json:"vnetSubnetID,omitempty" protobuf:"bytes,16,opt,name=vnetSubnetID"`
}

type AKSInstance struct {
	// Name of the instance
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`

	// Location of the instance
	Location string `json:"location,omitempty" protobuf:"bytes,2,opt,name=location"`

	// ID or index of the instance
	InstanceId string `json:"instanceId,omitempty" protobuf:"bytes,3,opt,name=instanceId"`

	// This is the uniquie id for the instance
	VMID string `json:"vmId,omitempty" protobuf:"bytes,4,opt,name=vmId"`

	// Array of zones
	Zones []string `json:"zones,omitempty" protobuf:"bytes,5,opt,name=zones"`

	// VM SKU
	SKU *AKSInstanceSKU `json:"sku,omitempty" protobuf:"bytes,6,opt,name=sku"`

	// The virtual machine scale set of which the instance is a part of
	VirtualMachineScaleSetName string `json:"virtualMachineScaleSetName,omitempty" protobuf:"bytes,7,opt,name=virtualMachineScaleSetName"`

	// The agent pool of which the instance is a part of
	AgentPoolName string `json:"agentPoolName,omitempty" protobuf:"bytes,8,opt,name=agentPoolName"`

	// Tags associated with the instance
	// +optional
	Tags map[string]string `json:"tags,omitempty" protobuf:"bytes,9,opt,name=tags"`

	// NetworkInterfaces attached to the instance
	// +optional
	NetworkInterfaces []string `json:"networkInterfaces,omitempty" protobuf:"bytes,10,opt,name=networkInterfaces"`

	// OSProfile describes an instance OS profile like computer name, username, etc.
	// +optional
	OSProfile *AKSInstanceOSProfile `json:"osProfile,omitempty" protobuf:"bytes,11,opt,name=osProfile"`
}

type AKSInstanceOSProfile struct {
	// ComputerName is the host OS name of the virtual machine This name cannot be updated after
	// the VM is created. Max-length (Windows): 15 characters. Max-length (Linux): 64 characters.
	// +optional
	ComputerName string `json:"computerName,omitempty" protobuf:"bytes,1,opt,name=computerName"`

	// AdminUsername is the name of the administrator account. This property cannot be updated
	// after the VM is created.
	// +optional
	AdminUsername string `json:"adminUsername,omitempty" protobuf:"bytes,2,opt,name=adminUsername"`

	// LinuxConfiguration is the Linux operating system settings on the virtual machine
	// +optional
	LinuxConfiguration *AKSInstanceLinuxConfiguration `json:"linuxConfiguration,omitempty" protobuf:"bytes,3,opt,name=linuxConfiguration"`

	// WindowsConfiguration is the Windows operating system settings on the virtual machine
	// +optional
	WindowsConfiguration *AKSInstanceWindowsConfiguration `json:"windowsConfiguration,omitempty" protobuf:"bytes,4,opt,name=windowsConfiguration"`
}

type AKSInstanceLinuxConfiguration struct {
	// SSHKeys specify the ssh key configurations for a Linux OS
	// +optional
	SSHKeys []AKSLinuxInstanceSSHKey `json:"sshKeys,omitempty" protobuf:"bytes,1,opt,name=sshKeys"`

	// DisablePasswordAuthentication specifies whether password authentication should be disabled
	// +optional
	DisablePasswordAuthentication bool `json:"disablePasswordAuthentication,omitempty" protobuf:"bytes,2,opt,name=disablePasswordAuthentication"`

	// ProvisionVMAgent indicates whether virtual machine agent should be provisioned on the virtual machine
	// +optional
	ProvisionVMAgent bool `json:"provisionVMAgent,omitempty" protobuf:"bytes,3,opt,name=provisionVMAgent"`
}

type AKSLinuxInstanceSSHKey struct {
	// KeyData is the SSH public key certificate used to authenticate with the VM through ssh.
	// The key needs to be at least 2048-bit and in ssh-rsa format.
	// +optional
	KeyData string `json:"keyData,omitempty" protobuf:"bytes,1,opt,name=keyData"`

	// Path is the full path on the created VM where ssh public key is stored. If the file already exists,
	// the specified key is appended to the file. Example: /home/user/.ssh/authorized_keys
	// +optional
	Path string `json:"path,omitempty" protobuf:"bytes,2,opt,name=path"`
}

type AKSInstanceWindowsConfiguration struct {
	// WinRMListeners is the list of Windows Remote Management listeners
	// +optional
	WinRMListeners []AKSWindowsInstanceWinRMListener `json:"winRMListeners,omitempty" protobuf:"bytes,1,opt,name=winRMListeners"`

	// EnableAutomaticUpdates indicates whether Automatic Updates is enabled for the
	// Windows virtual machine. Default value is true
	// +optional
	EnableAutomaticUpdates bool `json:"enableAutomaticUpdates,omitempty" protobuf:"bytes,2,opt,name=enableAutomaticUpdates"`

	// ProvisionVMAgent indicates whether virtual machine agent should be provisioned on the virtual machine.
	// +optional
	ProvisionVMAgent bool `json:"provisionVMAgent,omitempty" protobuf:"bytes,3,opt,name=provisionVMAgent"`

	// TimeZone specifies the time zone of the virtual machine. e.g. "Pacific Standard Time"
	// +optional
	TimeZone string `json:"timeZone,omitempty" protobuf:"bytes,4,opt,name=timeZone"`
}

type AKSWindowsInstanceWinRMListener struct {
	// +optional
	CertificateURL string `json:"certificateURL,omitempty" protobuf:"bytes,1,opt,name=certificateURL"`
	// +optional
	Protocol string `json:"protocol,omitempty" protobuf:"bytes,2,opt,name=protocol"`
}

type AKSInstanceSKU struct {
	// Name of the SKU
	// +optional
	Name string `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`

	// Tier of the SKU i.e. Standard, etc.
	// +optional
	Tier string `json:"tier,omitempty" protobuf:"bytes,2,opt,name=tier"`
}

// ClusterStatus represents information about the status of a Cluster. Status may
// trail the actual state of a system, especially if the Hosts that make up the
// Cluster are not able to contact Sunpike.
type ClusterStatus struct {

	// Phase represents the current phase of cluster actuation.
	// E.g. Pending, Running, Terminating, Failed etc.
	//
	// [qbert] clusters.status
	// +optional
	Phase ClusterPhase `json:"phase,omitempty" protobuf:"bytes,1,opt,name=phase"`

	// Message is a human-readable string that summarizes why the Cluster in this phase.
	// +optional
	Message string `json:"message,omitempty" protobuf:"bytes,2,opt,name=message"`

	// Masters contains the number of master nodes currently part of this cluster.
	//
	// Although this can also be retrieved by listing the Hosts and finding those
	// belonging to this cluster, this field provides an alternative way for easier
	// API consumption.
	//
	// [qbert] clusters.numMasters
	// +optional
	Masters int32 `json:"masters,omitempty" protobuf:"varint,3,opt,name=masters"`

	// Workers contains the number of worker nodes currently part of this cluster.
	//
	// Although this can also be retrieved by listing the Hosts and finding those
	// belonging to this cluster, this field provides an alternative way for easier
	// API consumption.
	//
	// [qbert] clusters.numWorkers
	// +optional
	Workers int32 `json:"workers,omitempty" protobuf:"varint,4,opt,name=workers"`

	// ControlPlaneEndpoint represents the endpoint used to communicate with the control plane.
	// +optional
	ControlPlaneEndpoint string `json:"controlPlaneEndpoint" protobuf:"bytes,5,opt,name=controlPlaneEndpoint"`

	// Type describes the controller-observed cloud type of this Cluster. This is
	// based on what subfields are set in the CloudProviderSpec.
	//
	// Although this can also be retrieved by looking up the CloudProvider
	// associated with this cluster, this field provides an alternative way for
	// easier API consumption.
	//
	// +optional
	Type CloudProviderType `json:"type,omitempty" protobuf:"bytes,6,opt,name=type"`

	// Conditions defines current service state of the Cluster.
	// +optional
	Conditions Conditions `json:"conditions,omitempty" protobuf:"bytes,7,opt,name=conditions"`

	// ObservedGeneration is the latest generation observed by the controller.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty" protobuf:"varint,8,opt,name=observedGeneration"`
}

// APIEndpoint represents a reachable Kubernetes API endpoint.
type APIEndpoint struct {
	// Host is the hostname on which the kube-apiserver should be serving.
	//
	// If the Host is left empty, a hostname will be generated.
	Host string `json:"host,omitempty" protobuf:"bytes,1,opt,name=host"`

	// Port specifies on which the kube-apiserver should be serving.
	Port int32 `json:"port,omitempty" protobuf:"varint,2,opt,name=port"`

	// UsePF9Domain indicates whether a the domain should be generated using a
	// platform9-managed domain. If it is set to false, a domain will be
	// generated specific to the cloud provider. For instance, in case of an AWS
	// cluster the domain name would be that of the ELB.
	//
	// If Host is not empty, this field will be ignored.
	// future(erwin): turn into enum?
	UsePF9Domain bool `json:"usePF9Domain,omitempty" protobuf:"varint,3,opt,name=usePF9Domain"`
}

// IsZero returns true if both host and port are zero values.
func (v APIEndpoint) IsZero() bool {
	return v.Host == "" && v.Port == 0
}

// IsValid returns true if both host and port are non-zero values.
func (v APIEndpoint) IsValid() bool {
	return v.Host != "" && v.Port != 0
}

// ClusterNetwork specifies the different networking parameters for a cluster.
type ClusterNetwork struct {
	// The network ranges from which service VIPs are allocated.
	// +optional
	ServicesCIDR string `json:"services,omitempty" protobuf:"bytes,1,opt,name=services"`

	// The network ranges from which Pod networks are allocated.
	// +optional
	PodsCIDR string `json:"pods,omitempty" protobuf:"bytes,2,opt,name=pods"`

	// Domain name for services.
	// Would be equal to ServiceFqdn in cluser_properties
	// +optional
	ServiceDomain string `json:"serviceDomain,omitempty" protobuf:"bytes,3,opt,name=serviceDomain"`
}

type LoadBalancer struct {
	// MetalLB specifies the options for the MetalLB addon which provides
	// support for load-balancer services.
	//
	// More info: https://metallb.universe.tf
	MetalLB MetalLBOpts `json:"metallb,omitempty" protobuf:"bytes,1,opt,name=metallb"`
}

type ContainerRuntime struct {
	// DockerOpts are options for the Docker runtime on the Host.
	Docker DockerOpts `json:"docker,omitempty" protobuf:"bytes,1,opt,name=docker"`

	// Runtime specifies the container runtime to use
	Runtime string `json:"runtime,omitempty" protobuf:"bytes,2,opt,name=runtime" kube.env:"RUNTIME"`
}

type StorageBackend struct {
	// Etcd contain configuration for the etcd cluster as a storage backend for
	// the Cluster. This is separated from the apiserver settings, because we
	// plan to separate out etcd from the master nodes.
	//
	// More info: https://etcd.io/docs/latest/
	Etcd EtcdOpts `json:"etcd,omitempty" protobuf:"bytes,1,opt,name=etcd"`
}

type Auth struct {
	// Keystone is the container for all settings related to authentication with
	// the OpenStack Keystone identity service.
	//
	// More info: https://docs.openstack.org/keystone/latest/
	Keystone KeystoneOpts `json:"keystone,omitempty" protobuf:"bytes,1,opt,name=keystone"`
}

type HA struct {
	// For HA setup
	Keepalived KeepalivedOpts `json:"keepalived,omitempty" protobuf:"bytes,1,opt,name=keepalived"`
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

// AddonsOpts is an aggregation of all supported addons.
type AddonsOpts struct {
	// AppCatalog specifies if and how the App Catalog addon should be
	// installed in the cluster.
	AppCatalog AppCatalogOpts `json:"appCatalog,omitempty" protobuf:"bytes,1,opt,name=appCatalog"`

	// CAS specifies if and how the Cluster AutoScaler addon should be
	// deployed in the cluster.
	CAS ClusterAutoScalerOpts `json:"cas,omitempty" protobuf:"bytes,2,opt,name=cas"`

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
}

// ProfileAgentOpts contains configuration related to platform9 profile engine agent
type ProfileAgentOpts struct {
	// Enabled signals that profile agent should be installed on the cluster, if set.
	Enabled bool `json:"enabled,omitempty" protobuf:"bool,1,opt,name=enabled" kube.env:"ENABLE_PROFILE_AGENT"`
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
}

// PF9 contains miscellaneous configuration, mostly related to PF9 services.
type PF9 struct {
	// Masterless, if set, instructs the cluster to run a proxy to a remote
	// apiserver, rather than running the apiserver in-cluster.
	Masterless bool `json:"masterless,omitempty" protobuf:"bool,1,opt,name=masterless" kube.env:"MASTERLESS_ENABLED"`

	// BouncerSlowReqWebhook is a field to enable code instrumentation to be
	// able to detect keystone slowness in some environments. It is unclear
	// whether this is still used.
	//
	// More info: https://platform9.atlassian.net/browse/INF-764
	BouncerSlowReqWebhook string `json:"bouncerSlowReqWebhook,omitempty" protobuf:"bytes,2,opt,name=bouncerSlowReqWebhook" kube.env:"BOUNCER_SLOW_REQUEST_WEBHOOK"`
}

// AWSCluster contains all AWS-specific configuration for the cluster
type AWSCluster struct {
	// SSHKeyName is the name of the ssh key to attach to the Hosts.
	// +optional
	SSHKeyName string `json:"sshKeyName,omitempty" protobuf:"bytes,1,opt,name=sshKeyName"`

	// Region contains the AWS Region the cluster lives in.
	Region string `json:"region,omitempty" protobuf:"bytes,2,opt,name=region"`

	// AZs specifies the Availablity Zones that the Cluster lives in.
	AZs []string `json:"azs,omitempty" protobuf:"bytes,3,opt,name=azs"`

	// AMI is the reference to the AMI from which to create the Hosts.
	// +optional
	AMI string `json:"ami,omitempty" protobuf:"bytes,4,opt,name=ami"`

	// MasterFlavor is the type of instance to use to create master Hosts.
	// Example: m4.xlarge
	MasterFlavor string `json:"masterFlavor,omitempty" protobuf:"bytes,5,opt,name=masterFlavor"`

	// WorkerFlavor is the type of instance to use to create worker Hosts.
	// Example: m4.xlarge
	WorkerFlavor string `json:"workerFlavor,omitempty" protobuf:"bytes,6,opt,name=workerFlavor"`
}

// GKECluster is Google Kubernetes Engine cluster.
// https://cloud.google.com/kubernetes-engine/docs/reference/rest/v1beta1/projects.zones.clusters
type GKECluster struct {
	// The list of Google Compute Engine zones in which the cluster's nodes should be located.
	Locations []string `json:"locations,omitempty" protobuf:"bytes,1,opt,name=locations"`

	// Unique id for the cluster.
	ID string `json:"id" protobuf:"bytes,2,opt,name=id"`

	// Base64-encoded public certificate that is the root of trust for the cluster.
	ClusterCACertificate string `json:"clusterCaCertificate" protobuf:"bytes,3,opt,name=clusterCaCertificate"`

	// The initial Kubernetes version for this cluster.
	InitialClusterVersion string `json:"initialClusterVersion" protobuf:"bytes,4,opt,name=initialClusterVersion"`

	// The Channel specifies which release channel the cluster is subscribed to.
	ReleaseChannel string `json:"releaseChannel" protobuf:"bytes,5,opt,name=releaseChannel"`

	// The DatabaseEncryption denotes the state of etcd encryption.
	DatabaseEncryption string `json:"databaseEncryption" protobuf:"bytes,6,opt,name=databaseEncryption"`

	// Cluster network details
	// +optional
	Network *GKEClusterNetwork `json:"network,omitempty" protobuf:"bytes,7,opt,name=network"`

	// Array of nodepools associated with the cluster
	// +optional
	NodePools []GKENodePool `json:"nodePools,omitempty" protobuf:"bytes,8,opt,name=nodePools"`

	// PrivateCluster indicates whether a cluster access is limited to a private network only
	// +optional
	PrivateCluster *bool `json:"privateCluster,omitempty" protobuf:"bytes,9,opt,name=privateCluster"`
}

// GKEClusterNetwork
type GKEClusterNetwork struct {
	// UseIpAliases indicates whether alias IPs is used for pod IPs in the cluster.
	UseIpAliases bool `json:"useIpAliases,omitempty" protobuf:"bytes,1,opt,name=useIpAliases"`
	// The relative name of the Google Compute Engine network to which the cluster is connected
	Network string `json:"network,omitempty" protobuf:"bytes,2,opt,name=network"`
	// The relative name of the Google Compute Engine subnetwork to which the cluster is connected.
	Subnetwork string `json:"subnetwork,omitempty" protobuf:"bytes,3,opt,name=subnetwork"`
	// The IP address range of the container pods in this cluster, in CIDR notation
	PodIpv4CIDR string `json:"podIpv4CIDR,omitempty" protobuf:"bytes,4,opt,name=podIpv4CIDR"`
	// The IP address range of the Kubernetes services in this cluster, in CIDR notation
	ServicesIpv4CIDR string `json:"servicesIpv4CIDR,omitempty" protobuf:"bytes,5,opt,name=servicesIpv4CIDR"`
	// NetworkPolicyConfig indicates whether NetworkPolicy is enabled for this cluster.
	NetworkPolicyConfig bool `json:"networkPolicyConfig,omitempty" protobuf:"bytes,6,opt,name=networkPolicyConfig"`
}

// GKENodePool is a group of nodes within a cluster
// https://cloud.google.com/kubernetes-engine/docs/reference/rest/v1beta1/projects.zones.clusters.nodePools
type GKENodePool struct {

	// The name of the node pool.
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`
	// Size of the disk attached to each node, specified in GB.
	DiskSizeGb int32 `json:"diskSizeGb,omitempty" protobuf:"bytes,2,opt,name=diskSizeGb"`
	// The name of a Google Compute Engine machine type
	MachineType string `json:"machineType" protobuf:"bytes,3,opt,name=machineType"`
	// The image type to use for this node.
	ImageType string `json:"imageType" protobuf:"bytes,4,opt,name=imageType"`
	// Type of the disk attached to each node (e.g. 'pd-standard', 'pd-ssd' or 'pd-balanced')
	DiskType string `json:"diskType" protobuf:"bytes,5,opt,name=diskType"`
	// The initial node count for the pool
	NodeCount int32 `json:"nodeCount,omitempty" protobuf:"bytes,6,opt,name=nodeCount"`
	// The constraint on the maximum number of pods that can be run simultaneously on a node in the node pool.
	MaxPodsPerNode string `json:"maxPodsPerNode" protobuf:"bytes,7,opt,name=maxPodsPerNode"`
	// The status of the nodes in this pool instance.
	Status string `json:"status" protobuf:"bytes,8,opt,name=status"`
	// The version of the Kubernetes of this node.
	K8sVersion string `json:"k8sVersion" protobuf:"bytes,9,opt,name=k8sVersion"`
	// The list of Google Compute Engine zones in which the NodePool's nodes should be located.
	Locations []string `json:"locations,omitempty" protobuf:"bytes,10,opt,name=locations"`
	// Array of GKEInstances that make up the GKE cluster nodes
	// +optional
	Instances []GKEInstance `json:"instances,omitempty" protobuf:"bytes,11,opt,name=instances"`
}

// GKEInstance is an instance in an instance group that is a part of a GKE cluster node pool
// https://cloud.google.com/workflows/docs/reference/googleapis/compute/v1/Overview?hl=en#ManagedInstance
type GKEInstance struct {
	// Name of the instance
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`
	// Status of the instance
	Status string `json:"status" protobuf:"bytes,2,opt,name=status"`
	// ID of the instance is a GUID
	ID string `json:"id" protobuf:"bytes,3,opt,name=id"`
	// Location of the instance
	Location string `json:"location" protobuf:"bytes,4,opt,name=location"`
	// InstanceTemplate used for creating the instance
	InstanceTemplate string `json:"instanceTemplate" protobuf:"bytes,5,opt,name=instanceTemplate"`
	// InstanceGroupName is the instance group that contains this instance
	InstanceGroupName string `json:"instanceGroupName" protobuf:"bytes,6,opt,name=instanceGroupName"`
}
