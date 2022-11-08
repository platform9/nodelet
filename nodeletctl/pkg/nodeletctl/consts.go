package nodeletctl

const (
	DefaultClusterName    = "airctl-mgmt"
	ClusterStateDir       = "/etc/nodelet/"
	NodeletConfigDir      = "/etc/pf9/nodelet"
	NodeletConfigFile     = "config_sunpike.yaml"
	NodeletUser           = "pf9"
	NodeletTarSrc         = "/opt/pf9/airctl/nodelet/nodelet.tar.gz"
	NodeletTarDst         = "/tmp/nodelet.tar.gz"
	NodeletPkgsTmpDir     = "/tmp/nodelet-pkgs/"
	OsTypeCentos          = "centos"
	OsTypeUbuntu          = "ubuntu"
	NodeConverged         = "converging"
	NodeHealthy           = "ok"
	CACertExpiryYears     = 3
	CAExpiryLimitDays     = 90
	RootCACRT             = "rootCA.crt"
	RootCAKey             = "rootCA.key"
	AdminKubeconfig       = "admin.kubeconfig"
	RemoteCertsDir        = "/etc/pf9/kube.d/"
	KubeStatusFile        = "/var/opt/pf9/kube_status"
	SyncRetrySeconds      = 30
	ClusterTimeout        = 30
	WorkerLabel           = "node-role.kubernetes.io/worker"
	MasterLabel           = "node-role.kubernetes.io/master"
	UserImagesDir         = "/var/opt/pf9/images"
	CAPath                = "/etc/pki/ca-trust/source/anchors/nodelet-ca.pem"
	DefaultCalicoV4Cidr   = "10.20.0.0/22"
	DefaultCalicoV6Cidr   = "fd00:101::/116"
	DefaultV4ServicesCidr = "10.21.0.0/22"
	DefaultV6ServicesCidr = "fd00:102::/116"
)
