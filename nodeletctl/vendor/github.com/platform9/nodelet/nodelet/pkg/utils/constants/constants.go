package constants

import "fmt"

const (
	// ConfigDir : Directory for nodelet config files
	ConfigDir = "/etc/pf9/nodelet/"
	// DefaultConfigFileName is the default filename to use when writing config to ConfigDir.
	DefaultConfigFileName = "config.yaml"
	// KubeEnvPath contains the default path to look for the kube.env config file.
	KubeEnvPath = "/etc/pf9/kube.env"
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
	//LoopBackIpString represents loopback IP string also known as localhost
	LoopBackIpString = "127.0.0.1"
	// LocalHost represents localhost as a string
	LocalHostString = "localhost"
	//LocalCloudProvider represents cloud provider type as local
	LocalCloudProvider = "local"
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
	// ContainerdAddress is default address for containerd socket
	ContainerdAddress = "/run/containerd/containerd.sock"
	// DefaultSnapShotter is default snapshotter for containerd
	DefaultSnapShotter = "overlayfs"
	//K8sNamespace is namespace for kubernetes
	K8sNamespace = "k8s.io"

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