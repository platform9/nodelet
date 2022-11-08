package constants

import (
	"fmt"
)

const (
	// ConfigDir : Directory for nodelet config files
	ConfigDir = "/etc/pf9/nodelet/"
	// DefaultConfigFileName is the default filename to use when writing config to ConfigDir.
	DefaultConfigFileName = "config.yaml"
	// ExtensionOutputFile : File that will be used by hostagent extension
	ExtensionOutputFile = "/var/opt/pf9/kube_status"
	// ErrorState : String depicting host in error state
	ErrorState = "failed"
	// OkState : String depicting host in ok state
	OkState = "ok"
	// ConvergingState : String depicting host in converging state
	ConvergingState = "converging"
	// RetryingState : String depicting host in retrying state
	RetryingState = "retrying"
	// NumRetriesForErrorState : Number of times service start will be retried before host is put in error state
	NumRetriesForErrorState = 10
	// DefaultLoopInterval : Number of seconds to sleep between to iterations of the main nodelet loop
	DefaultLoopInterval = 60 // seconds
	// FailedStatusCheckReapInterval : Number of seconds for which failed status check will be persisted
	FailedStatusCheckReapInterval int64 = 600 // seconds
	// DefaultPhaseBaseDir : directory in which master_scripts and worker_scripts directories are present
	DefaultPhaseBaseDir = "/opt/pf9/pf9-kube/"
	// CgroupName : Name of cgroup to use for running status commands
	CgroupName = "pf9-kube-status"
	// CgroupQuotaParam : Cgroup param to set to limit CPU quotas
	CgroupQuotaParam = "cpu.cfs_quota_us=%v"
	// NotStartedState Phase state when no operation has been performed
	NotStartedState = "not-started"
	// RunningState Phase state when start operation was successful
	RunningState = "running"
	// StoppedState Phase state when stop operation was successful
	StoppedState = "stopped"
	// FailedState Phase state when start operation has failed
	FailedState = ErrorState
	// ExecutingState Phase state when an operation is being performed on a phase
	ExecutingState = "executing"
	// StartOp denotes a cluster bring up operation
	StartOp = "start"
	// StatusOp denotes a status check in progress
	StatusOp = "status"
	// StopOp denotes a cluster tear down
	StopOp = "stop"
	// RoleMaster will turn the host into a Kubernetes master node.
	RoleMaster = "master"
	// RoleWorker will turn the host into a Kubernetes worker node.
	RoleWorker = "worker"
	// RoleNone will not turn the host into a Kubernetes node.
	RoleNone = "none"
	// ServiceStateTrue is the desired state that brings up the host into the cluster.
	ServiceStateTrue = "true"
	// ServiceStateFalse is the desired state that brings down the host from the cluster.
	ServiceStateFalse = "false"
	// ServiceStateIgnore is a state in which nodelet will not do anything.
	ServiceStateIgnore = "ignore"
	// GeneratedFileHeader is the default header for generated files.
	GeneratedFileHeader = "Generated by Nodelet — DO NOT EDIT."
	// DefaultSunpikeKubeEnvPath contains the default path to use as the
	// kube.env config file filled by a Sunpike Host.
	DefaultSunpikeKubeEnvPath = "/etc/pf9/kube_sunpike.env"
	// DefaultResmgrKubeEnvPath contains the default path to use as the
	// kube.env config file filled by resmgr/hostagent.
	DefaultResmgrKubeEnvPath = "/etc/pf9/kube_resmgr.env"
	// DefaultSunpikeConfigPath contains the default path to use as the
	// Nodelet config.yaml file filled by a Sunpike Host.
	DefaultSunpikeConfigPath = "/etc/pf9/nodelet/config_sunpike.yaml"
	// TrueString represents true as a string in nodeletd
	TrueString = "true"
	// LoopBackIpString represents loopback IP string also known as localhost
	LoopBackIpString = "127.0.0.1"
	// LocalHost represents localhost as a string
	LocalHostString = "localhost"
	// LocalCloudProvider represents cloud provider type as local
	LocalCloudProvider = "local"
	// RuntimeContainerd represents containerd service
	RuntimeContainerd = "containerd"

	CgroupSystemd = "SystemdCgroup"

	IsActiveOp = "is-active"

	ActiveState = "active"
)

var (
	// BaseCommand : Base command to be used for invoking different phase scripts
	BaseCommand = []string{"sudo", "/opt/pf9/pf9-kube/setup_env_and_run_script.sh"}
	// BaseCgroupCommand : Base command to be used for invoking different phase scripts
	BaseCgroupCommand = []string{"cgexec", "-g", "cpu:pf9-kube-status", "sudo", "/opt/pf9/pf9-kube/setup_env_and_run_script.sh"}
	// CgroupCreateCmd : Command for creating cgroup
	CgroupCreateCmd = []string{"sudo", "/usr/bin/cgcreate", "-a", "pf9:pf9group", "-t", "pf9:pf9group", "-g", "cpu:pf9-kube-status"}
	// CgroupPeriodCmd : Command for setting cpu.cfs_period_us property to 1s
	CgroupPeriodCmd = []string{"sudo", "/usr/bin/cgset", "-r", "cpu.cfs_period_us=1000000", CgroupName}
	// CgroupQuotaCmd : Base command for setting cpu.cfs_quota_us property
	CgroupQuotaCmd = []string{"sudo", "/usr/bin/cgset", "-r"}
	// ValidCgroupOps is a set of strings containing all the operations that can be performed inside the cgroup
	// very basic replacement to sets with a constant time and simplified lookup.
	ValidCgroupOps = map[string]struct{}{"status": {}}

	// Newly added constants from env variables
	ConfigDstDir             = "/etc/pf9/kube.d"
	AdminCerts               = ConfigDstDir + "/certs/admin"
	KubeConfig               = ConfigDstDir + "/kubeconfigs/admin.yaml"
	KubectlCmd               = fmt.Sprintf("bin/kubectl -v=8 --kubeconfig=%s --context=default-context", KubeConfig)
	KubeStackStartFileMarker = "var/opt/pf9/is_node_booting_up"
	// UserImagesDir is the default directory for tar/zip archives of user images
	UserImagesDir = "/var/opt/pf9/images"
	// ChecksumDir is the directory where checksum file is present
	ChecksumDir = fmt.Sprintf("%s/checksum", UserImagesDir)
	// ChecksumFile contains sha256 hash for tar archives of user images
	ChecksumFile = fmt.Sprintf("%s/sha256sums.txt", ChecksumDir)
	// ContainerdSocket is default address for containerd socket
	ContainerdSocket = "/run/containerd/containerd.sock"
	// DefaultSnapShotter is default snapshotter for containerd
	DefaultSnapShotter = "overlayfs"
	// Containerd binary path
	ContainerdBinPath = "/usr/local/bin/containerd"
	// Nerdctl directory path
	NerdctlDir = "/var/lib/nerdctl"
	// K8sNamespace is namespace for kubernetes
	K8sNamespace = "k8s.io"
	// MobyNamespace is namespace for docker
	MobyNamespace = "moby"
	// K8sRegistry represents registry for official images for kubernetes
	K8sRegistry = "k8s.gcr.io"

	// Kubelet related variables from defaults.env
	KubeletDataDir          = "/var/lib/kubelet"
	KubeletKubeconfig       = "/etc/pf9/kube.d/kubeconfigs/kubelet.yaml"
	KubeletLogDirPath       = "/var/log/pf9/kubelet/"
	CNIConfigDir            = "/etc/cni/net.d"
	CNIBinDir               = "/opt/cni/bin"
	KubeletConfigDir        = "/var/opt/pf9/kube/kubelet-config"
	KubeletDynamicConfigDir = KubeletConfigDir + "/dynamic-config"
	KubeletBootstrapConfig  = KubeletConfigDir + "/bootstrap-config.yaml"
	// KubeEnvPath contains the default path to look for the kube.env config file.
	KubeEnvPath = "/etc/pf9/kube.env"
	// AWSMetadataIp # 169.254.169.254 belongs to the 169.254/16 range of IPv4 Link-Local addresses (https://tools.ietf.org/html/rfc3927).
	//# This IP address in particular is significant because Amazon Web Services uses this IP address
	//# for instance metadata (https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-instance-metadata.html)
	AWSMetadataIp          = "169.254.169.254"
	AWSInstanceIdLoc       = "/var/lib/cloud/data/instance-id"
	AWSAvailabilityZoneURL = "http://" + AWSMetadataIp + "/latest/meta-data/placement/availability-zone"
	OpenstackMetadataIp    = "169.254.169.254"
	Runtime                = "containerd"
	ContainerdCgroup       = "systemd"
	UseHostname            = "false"
	DockerLogMaxFile       = "10"
	ContainerLogMaxFiles   = DockerLogMaxFile
	// ContainerLogMaxSize
	// Why not use DOCKER_LOG_MAX_SIZE variable?
	// The formatting for docker config is 10m while kubelet expects 10Mi. To avoid implement string manipulation in bash just hardcoding
	// the same default as docker config for now.
	ContainerLogMaxSize           = "10Mi"
	EnableCAS                     = false
	AllowSwap                     = false
	KubeletBin                    = "/opt/pf9/pf9-kube/bin/kubelet"
	Pf9KubeletSystemdUnitTemplate = "/opt/pf9/pf9-kube/pf9-kubelet.service.template"
	SystemdRuntimeUnitDir         = "/run/systemd/system"
	Pf9User                       = "pf9"
	Pf9Group                      = "pf9group"
	DnsDomain                     = "cluster.local"
	CPUManagerPolicy              = "none"
	TopologyManagerPolicy         = "none"
	ReservedCPUs                  = ""
	KubeletEnvPath                = "/etc/pf9/kubelet.env"
	KubeletStaticPodPath          = "/etc/pf9/kube.d/master.yaml"
	KubeletClientCaFile           = "/etc/pf9/kube.d/certs/kubelet/server/ca.crt"
	KubeletTlsCertFile            = "/etc/pf9/kube.d/certs/kubelet/server/request.crt"
	KubeletTlsPrivateKeyFile      = "/etc/pf9/kube.d/certs/kubelet/server/request.key"
	KubeletTlsCipherSuites        = "[TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256]"
	KubeletCloudConfig            = ""
	ServicesCIDR                  = "10.21.0.0/22"

	ConfigSrcDir = "/opt/pf9/pf9-kube/conf"
	// CoreDNSTemplate is template file for coredns
	CoreDNSTemplate = fmt.Sprintf("%s/networkapps/coredns.yaml", ConfigSrcDir)
	// CoreDNSFile is applied coredns file
	CoreDNSFile      = fmt.Sprintf("%s/networkapps/coredns-applied.yaml", ConfigSrcDir)
	CoreDNSHostsFile = "/etc/hosts"

	CloudConfigFile = "/etc/pf9/kube.d/cloud-config"

	EtcContainerdDir = "/etc/containerd"

	ContainerdConfigFile = fmt.Sprintf("%s/config.toml", EtcContainerdDir)

	// Phase orders of all the phases
	NoRolePhaseOrder                   = 10
	GenCertsPhaseOrder                 = 20
	KubeconfigPhaseOrder               = 30
	ConfigureRuntimePhaseOrder         = 40
	StartRuntimePhaseOrder             = 45
	LoadImagePhaseOrder                = 48
	ConfigureEtcdPhaseOrder            = 50
	StartEtcdPhaseOrder                = 55
	ConfigureNetworkPhaseOrder         = 60
	ConfigureCNIPhaseOrder             = 65
	AuthWebHookPhaseOrder              = 70
	MiscPhaseOrder                     = 75
	ConfigureKubeletPhaseOrder         = 80
	KubeProxyPhaseOrder                = 90
	WaitForK8sSvcPhaseOrder            = 100
	LabelTaintNodePhaseOrder           = 110
	DynamicKubeletConfigPhaseOrder     = 120
	UncordonNodePhaseOrder             = 130
	DeployAppCatalogPhaseOrder         = 160
	ConfigureStartKeepalivedPhaseOrder = 180
	PF9SentryPhaseOrder                = 205
	PF9CoreDNSPhaseOrder               = 206
	DrainPodsPhaseOrder                = 210

	// PhaseBaseDir is the base directory in which all bash-based phase scripts are located
	PhaseBaseDir = "/opt/pf9/pf9-kube/phases"
)
