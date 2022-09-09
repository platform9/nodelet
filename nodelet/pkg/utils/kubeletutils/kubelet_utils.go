package kubeletutils

import (
	"context"
	"fmt"
	"github.com/platform9/nodelet/nodelet/pkg/utils/command"
	"github.com/platform9/nodelet/nodelet/pkg/utils/config"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	"github.com/platform9/nodelet/nodelet/pkg/utils/netutils"
	"go.uber.org/zap"
	"io"
	"net/http"
	"os"
	osuser "os/user"
	"strconv"
	"strings"
)

type KubeletUtilsInterface interface {
	EnsureKubeletRunning(config.Config) error
	EnsureKubeletStopped() error
	FetchAwsInstanceId() (string, error)
	FetchAwsAz() (string, error)
	TrimSans(str string) string
	PrepareKubeletBootstrapConfig(cfg config.Config) error
	EnsureDirReadableByPf9(dir string) error
	KubeletSetup(kubeletArgs string) error
	GenerateKubeletSystemdUnit(kubeletArgs string) error
	ConfigureKubeletHttpProxy()
	KubeletStart() error
	KubeletStop() error
	IsKubeletRunning() bool
}

type KubeletImpl struct{}

func New() KubeletUtilsInterface {
	return &KubeletImpl{}
}

func (k *KubeletImpl) EnsureKubeletStopped() error {
	if k.IsKubeletRunning() {
		err := k.KubeletStop()
		if err != nil {
			zap.S().Panicf("failed to ensure kubelet has stopped %s", err)
			return err
		}
	}
	return nil
}

func (k *KubeletImpl) EnsureKubeletRunning(cfg config.Config) error {
	if k.IsKubeletRunning() {
		return nil
	}

	kubeconfig := "/etc/pf9/kube.d/kubeconfigs/kubelet.yaml"
	logDirPath := "/var/log/pf9/kubelet/"
	pauseImg := cfg.K8sPrivateRegistry + "/pause:3.2"

	err := k.PrepareKubeletBootstrapConfig(cfg)
	if err != nil {
		zap.S().Errorf("failed to prepare kubelet bootstrap config")
		return err
	}

	err = os.MkdirAll(constants.KubeletDataDir, 0660)
	if err != nil {
		zap.S().Panicf("failed to create kubelet data directory directory")
		return err
	}

	kubeletArgs := fmt.Sprintf(" --kubeconfig=" + kubeconfig +
		" --enable-server" +
		" --network-plugin=cni" +
		" --cni-conf-dir=" + constants.CNIConfigDir +
		" --cni-bin-dir=" + constants.CNIBinDir +
		" --log-dir=" + logDirPath +
		" --logtostderr=false" +
		" --config=" + constants.KubeletBootstrapConfig +
		" --register-schedulable=false" +
		" --pod-infra-container-image=" + pauseImg +
		" --dynamic-config-dir=" + constants.KubeletDynamicConfigDir +
		" --cgroup-driver=" + constants.ContainerdCgroup)

	if cfg.Runtime == "containerd" {
		containerLogMaxFiles := cfg.ContainerLogMaxFiles
		//Why not use DOCKER_LOG_MAX_SIZE variable?
		//The formatting for docker config is 10m while kubelet expects 10Mi. To avoid implement string manipulation in bash just hardcoding
		//the same default as docker config for now.
		containerLogMaxSize := cfg.ContainerLogMaxSize

		kubeletArgs += fmt.Sprintf(" --container-runtime=remote" +
			" --runtime-request-timeout=15m" +
			" --container-runtime-endpoint=unix://" + constants.ContainerdSocket +
			" --container-log-max-files=" + containerLogMaxFiles +
			" --container-log-max-size=" + containerLogMaxSize)
	}

	netutls := netutils.New()
	nodeName, err := netutls.GetNodeIdentifier(cfg)
	if err != nil {
		zap.S().Errorf("failed to fetch node name %s", err)
		return err
	}

	// if CLOUD_PROVIDER_TYPE is not local i.e. AWS, Azure, etc. or if it is local but USE_HOSTNAME is not true then use the node_endpoint (IP address).
	if cfg.CloudProviderType != "local" || (cfg.CloudProviderType == "local" && cfg.UseHostname != "true") {
		// if --hostname-override is not specified hostname of the node is used by default
		// in case --hostname-override is specified along with cloud provider then cloud provider determines
		// the hostname
		// https://kubernetes.io/docs/reference/command-line-tools-reference/kubelet/
		kubeletArgs += " --hostname-override=" + nodeName
	}

	nodeIP, err := netutls.GetNodeIP()
	if err != nil {
		zap.S().Panicf("failed to fetch NodeIP %s", err)
		return err
	}

	if cfg.CloudProviderType == "local" && cfg.UseHostname != "true" {
		kubeletArgs += " --node-ip=" + nodeIP
	}

	if cfg.CloudProviderType == "aws" {
		kubeletArgs += " --cloud-provider=aws"
		if cfg.EnableCAS == true {
			instanceId, err := k.FetchAwsInstanceId()
			if err != nil {
				zap.S().Panicf("failed to fetch AWS instance-id %s", err)
				return err
			}
			availabilityZone, err := k.FetchAwsAz()
			if err != nil {
				zap.S().Panicf("failed to fetch AWS availability zone %s", err)
				return err
			}
			trimmedInstanceId := k.TrimSans(instanceId)
			trimmedAvailabilityZone := k.TrimSans(availabilityZone)
			kubeletArgs += " --provider-id=aws:///" + trimmedAvailabilityZone + "/" + trimmedInstanceId
		}
	}
	if cfg.CloudProviderType == "openstack" {
		kubeletArgs += " --cloudProvider=openstack"
		kubeletArgs += " --cloud-config=" + constants.CloudConfigFile
	}
	if cfg.CloudProviderType == "azure" {
		kubeletArgs += " --cloudProvider=azure"
		kubeletArgs += " --cloudConfig=" + constants.CloudConfigFile
	}

	if cfg.Debug == "true" {
		kubeletArgs += " --v=8"
	} else {
		kubeletArgs += " --v=2"
	}

	err = k.KubeletSetup(kubeletArgs)
	if err != nil {
		zap.S().Panicf("failed to setup kubelet %s", err)
		return err
	}
	zap.S().Infof("Starting kubelet...")
	err = k.KubeletStart()
	if err != nil {
		zap.S().Panicf("failed to start kubelet %s", err)
		return err
	}
	return nil
}

func (k *KubeletImpl) FetchAwsInstanceId() (string, error) {
	instanceId, err := os.ReadFile("/var/lib/cloud/data/instance-id")
	if err != nil {
		zap.S().Panicf("failed to read instance-id. %s", err)
		return "", err
	}
	return string(instanceId), nil
}

// TODO do a cross check
func (k *KubeletImpl) FetchAwsAz() (string, error) {
	url := "http://" + constants.AWSMetadataIp + "/latest/meta-data/placement/availability-zone"
	resp, err := http.Get(url)
	if err != nil {
		zap.S().Panicf("failed to fetch AWS availability zone with %s\n", err)
		return "", err
	}

	az, err := io.ReadAll(resp.Body)
	if err != nil {
		zap.S().Panicf("failed to read response body while fetching AWS availability zone with %s", err)
		return "", err
	}
	//Convert the body to type string
	return string(az), nil
}

func (k *KubeletImpl) TrimSans(str string) string {
	// Remove all new-lines from the string
	str = strings.ReplaceAll(str, "\n", "")
	// Remove all spaces from the string
	str = strings.ReplaceAll(str, " ", "")
	return str
}

func (k *KubeletImpl) PrepareKubeletBootstrapConfig(cfg config.Config) error {

	err := os.MkdirAll(constants.KubeletConfigDir, 0660)
	if err != nil {
		zap.S().Panicf("failed to create kubelet config directory directory")
		return err
	}

	err = k.EnsureDirReadableByPf9(constants.KubeletConfigDir)
	if err != nil {
		zap.S().Errorf("failed to ensure directory readable by pf9")
		return err
	}

	dnsIp := "bin/addr_conv -cidr " + cfg.ServicesCIDR + " -pos 10"

	kubeletBootstrapConfig := "apiVersion: kubelet.config.k8s.io/v1beta1\n" +
		"kind: KubeletConfiguration\n" +
		"address: 0.0.0.0\n" +
		"authentication:\n" +
		"  anonymous:\n" +
		"    enabled: false\n" +
		"  webhook:\n" +
		"    enabled: true\n" +
		"  x509:\n" +
		"    clientCAFile:" + constants.KubeletClientCaFile + "\n" +
		"authorization:\n" +
		"  mode: AlwaysAllow\n" +
		"clusterDNS:\n" +
		"- " + dnsIp + "\n" +
		"clusterDomain: " + constants.DnsDomain + "\n" +
		// Fixme what to put in defaults for this or should I delete it when empty
		"cpuManagerPolicy:" + cfg.CPUManagerPolicy + "\n" +
		"topologyManagerPolicy:" + cfg.TopologyManagerPolicy + "\n" +
		"reservedSystemCPUs:" + cfg.ReservedCPUs + "\n" +
		"featureGates:\n" +
		"  DynamicKubeletConfig: true\n" +
		"maxPods: 200\n" +
		"readOnlyPort: 0\n" +
		"tlsCertFile: " + constants.KubeletTlsCertFile + "\n" +
		"tlsPrivateKeyFile: " + constants.KubeletTlsPrivateKeyFile + "\n" +
		"tlsCipherSuites: " + constants.KubeletTlsCipherSuites + "\n"

	if cfg.ContainerdCgroup == "systemd" {
		kubeletBootstrapConfig += "cgroupDriver: systemd\n"
	} else {
		kubeletBootstrapConfig += "cgroupDriver: cgroupfs\n"
	}

	// Reason why worker doesn't need staticPodPath # Apiserver, controller-manager, and scheduler don't run on workers,
	// so don't need staticPodPath (it spams pf9-kubelet journalctl logs)
	if cfg.ClusterRole == "master" {
		kubeletBootstrapConfig += "staticPodPath: " + constants.KubeletStaticPodPath + "\n"
	}

	if cfg.AllowSwap {
		kubeletBootstrapConfig += "failSwapOn: false\n"
	}

	return nil
}

func (k *KubeletImpl) EnsureDirReadableByPf9(dir string) error {
	user, err := osuser.Lookup(constants.Pf9User)
	if err != nil {
		return err
	}

	uid, err := strconv.Atoi(user.Uid)
	if err != nil {
		return err
	}
	gid, err := strconv.Atoi(user.Gid)
	if err != nil {
		return err
	}

	err = os.Chown(dir, uid, gid)
	if err != nil {
		zap.S().Errorf("failed to change file permissions. %s", err)
		return err
	}
	return nil
}

// os specific stuff for ubuntu and centos but since both were same just combined them

func (k *KubeletImpl) KubeletSetup(kubeletArgs string) error {
	err := k.GenerateKubeletSystemdUnit(kubeletArgs)
	if err != nil {
		zap.S().Errorf("failed to generate kubelet systemd unit")
		return err
	}

	cmd := command.New()
	_, stdErr, err := cmd.RunCommandWithStdErr(context.Background(), nil, 0, "", "/bin/sudo", "/usr/bin/systemctl", "daemon-reload")
	if err != nil {
		zap.S().Panicf("failed to reload daemon. %s %s\n", stdErr[0], err)
		return err
	}
	zap.S().Debugf("daemon reloaded\n")
	return nil

	/* These lines were there in the script but commented so just keeping a copy
	# Master component containers won't stay running unless selinux is in permissive mode
	# set selinux mode to Permissive when already it is in Enforcing mode. Ignore if the mode is Disabled by default
	# ret=`getenforce`
	# if [ "${ret}" == "Enforcing" ]; then
	#    sed -i s/SELINUX=enforcing/SELINUX=permissive/g /etc/selinux/config
	#    # set selinux mode to Permissive(0)
	#    setenforce Permissive
	# fi
	*/

}

func (k *KubeletImpl) GenerateKubeletSystemdUnit(kubeletArgs string) error {
	zap.S().Infof("Generating runtime systemd unit for kubelet")

	kubeEnv, err := os.ReadFile(constants.KubeEnvPath)
	if err != nil {
		zap.S().Panicf("failed to read file /etc/pf9/kube.env. %s", err)
		return err
	}
	// Remove export from each field in file
	kubeletEnv := strings.ReplaceAll(string(kubeEnv), "export ", "")
	err = os.WriteFile(constants.KubeletEnvPath, []byte(kubeletEnv), 0660)
	if err != nil {
		zap.S().Panicf("failed to write to file /etc/pf9/kubelet.env. %s", err)
		return err
	}

	// Proxy Implementation not done
	//if pf9KubeHttpProxyConfigured == true {
	//	zap.S().Infof("kubelet configuration: http proxy configuration detected; appending proxy env vars to /etc/pf9/kubelet.env")
	//	configureKubeletHttpProxy()
	//} else {
	//	zap.S().Infof("kubelet configuration: http proxy configuration not detected")
	//}

	pf9KubeletSystemdUnitTemplate, err := os.ReadFile(constants.Pf9KubeletSystemdUnitTemplate)
	if err != nil {
		zap.S().Panicf("failed to read file Pf9KubeletSystemdUnitTemplate. %s", err)
		return err
	}
	pf9KubeletService := strings.ReplaceAll(string(pf9KubeletSystemdUnitTemplate), "__KUBELET_BIN__", constants.KubeletBin)
	pf9KubeletService = strings.ReplaceAll(pf9KubeletService, "__KUBELET_ARGS__", kubeletArgs)
	err = os.WriteFile(constants.SystemdRuntimeUnitDir+"/pf9-kubelet.service", []byte(pf9KubeletService), 0660)
	if err != nil {
		zap.S().Panicf("failed to write to file pf9-kubelet.service. %s", err)
		return err
	}
	return nil
}

func (k *KubeletImpl) ConfigureKubeletHttpProxy() {
	zap.S().Debugf("configureKubeletHttpProxy() Function not implemented")
	return
}

func (k *KubeletImpl) KubeletStart() error {
	cmd := command.New()
	_, stdErr, err := cmd.RunCommandWithStdErr(context.Background(), nil, 0, "", "/bin/sudo", "/usr/bin/systemctl", "start", "pf9-kubelet")
	if err != nil {
		zap.S().Panicf("failed to start pf9-kubelet. %s, %s\n", err, stdErr[0])
		return err
	}
	zap.S().Debugf("pf9-kubelet started.")
	return nil
}

func (k *KubeletImpl) KubeletStop() error {
	cmd := command.New()
	_, stdErr, err := cmd.RunCommandWithStdErr(context.Background(), nil, 0, "", "/bin/sudo", "/usr/bin/systemctl", "stop", "pf9-kubelet")
	if err != nil {
		zap.S().Panicf("failed to stop pf9-kubelet. %s, %s\n", err, stdErr[0])
		return err
	}
	zap.S().Debugf("pf9-kubelet stopped:\n%s\n")
	return nil
}

func (k *KubeletImpl) IsKubeletRunning() bool {
	cmd := command.New()
	_, stdOut, stdErr, err := cmd.RunCommandWithStdOutStdErr(context.Background(), nil, 0, "", "/bin/sudo", "/usr/bin/systemctl", "is-active", "pf9-kubelet")
	if err != nil {
		zap.S().Debugf("pf9-kubelet is not active.%s, %s\n", err, stdErr[0])
		return false
	}
	zap.S().Debugf("pf9-kubelet is active. %s\n", stdOut)
	return true
}
