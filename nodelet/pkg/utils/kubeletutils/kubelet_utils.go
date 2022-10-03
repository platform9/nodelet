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
	"strings"
)

type KubeletUtilsInterface interface {
	EnsureKubeletStopped() error
	EnsureKubeletRunning(config.Config) error
	FetchAwsInstanceId() (string, error)
	FetchAwsAz() (string, error)
	TrimSans(string) string
	PrepareKubeletBootstrapConfig(cfg config.Config) error
	EnsureDirReadableByPf9(string) error
	KubeletSetup(string) error
	GenerateKubeletSystemdUnit(string) error
	ConfigureKubeletHttpProxy()
	KubeletStart() error
	KubeletStop() error
	IsKubeletRunning() bool
}

type KubeletImpl struct {
	Cmd      command.CLI
	NetUtils netutils.NetInterface
}

func New() KubeletUtilsInterface {
	return &KubeletImpl{
		Cmd:      command.New(),
		NetUtils: netutils.New(),
	}
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

	pauseImg := cfg.K8sPrivateRegistry + "/pause:3.6"

	// Prepares kubelet bootstrap config and saves it to bootstrap-config.yaml file
	err := k.PrepareKubeletBootstrapConfig(cfg)
	if err != nil {
		zap.S().Errorf("failed to prepare kubelet bootstrap config, %s", err)
		return err
	}

	// need to create this with sudo due to lack of permissions
	_, stdErr, err := k.Cmd.RunCommandWithStdErr(context.Background(), nil, 0, "", "sudo", "/bin/mkdir", "-p", constants.KubeletDataDir)
	if err != nil {
		zap.S().Panicf("failed to create kubelet data directory. %s, %s\n", err, stdErr[0])
		return err
	}
	zap.S().Debugf("kubelet data directory created\n")

	//err = os.MkdirAll(constants.KubeletDataDir, 0660)
	//if err != nil {
	//	zap.S().Panicf("failed to create kubelet data directory directory, %s", err)
	//	return err
	//}
	// remove permission for /var/lib

	kubeletArgs := fmt.Sprintf(" --kubeconfig=" + constants.KubeletKubeconfig +
		" --enable-server" +
		" --network-plugin=cni" +
		" --cni-conf-dir=" + constants.CNIConfigDir +
		" --cni-bin-dir=" + constants.CNIBinDir +
		" --log-dir=" + constants.KubeletLogDirPath +
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

	nodeName, err := k.NetUtils.GetNodeIdentifier(cfg)
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

	nodeIP, err := k.NetUtils.GetNodeIP()
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
	instanceId, err := os.ReadFile(constants.AWSInstanceIdLoc)
	if err != nil {
		zap.S().Panicf("failed to read instance-id. %s", err)
		return "", err
	}
	return string(instanceId), nil
}

func (k *KubeletImpl) FetchAwsAz() (string, error) {
	url := constants.AWSAvailabilityZoneURL
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

	err := os.MkdirAll(constants.KubeletConfigDir, 0770)
	if err != nil {
		zap.S().Panicf("failed to create kubelet config directory directory %s", err)
		return err
	}

	err = k.EnsureDirReadableByPf9(constants.KubeletConfigDir)
	if err != nil {
		zap.S().Errorf("failed to ensure directory readable by pf9, %s", err)
		return err
	}

	dnsIp, err := k.NetUtils.AddrConv(cfg.ServicesCIDR, 10)
	if err != nil {
		zap.S().Errorf("failed to convert address, %s", err)
		return err
	}

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

	// Write the kubeletBootstrapConfig to the config file
	err = os.WriteFile(constants.KubeletBootstrapConfig, []byte(kubeletBootstrapConfig), 0770)
	if err != nil {
		zap.S().Errorf("failed to write kubelet bootstrap config file in %s, %s", constants.KubeletBootstrapConfig, err)
		return err
	}

	return nil
}

func (k *KubeletImpl) EnsureDirReadableByPf9(dir string) error {
	//user, err := osuser.Lookup(constants.Pf9User)
	//if err != nil {
	//	return err
	//}
	//
	//uid, err := strconv.Atoi(user.Uid)
	//if err != nil {
	//	return err
	//}
	//gid, err := strconv.Atoi(user.Gid)
	//if err != nil {
	//	return err
	//}
	//
	//err = os.Chown(dir, uid, gid)
	//if err != nil {
	//	zap.S().Errorf("failed to change file permissions. %s", err)
	//	return err
	//}
	//return nil

	usrgrp := constants.Pf9User + ":" + constants.Pf9Group
	_, stdOut, stdErr, err := k.Cmd.RunCommandWithStdOutStdErr(context.Background(), nil, 0, "", "sudo", "chown", "-R", usrgrp, dir)
	if err != nil {
		zap.S().Panicf("failed ensure directory readable by pf9 user. %s %s\n", stdErr[0], err)
		return err
	}
	zap.S().Debugf("ensured directory readable by pf9 user\n%s", stdOut)
	return nil

}

// os specific stuff for ubuntu and centos but since both were same just combined them

func (k *KubeletImpl) KubeletSetup(kubeletArgs string) error {
	err := k.GenerateKubeletSystemdUnit(kubeletArgs)
	if err != nil {
		zap.S().Errorf("failed to generate kubelet systemd unit, %s", err)
		return err
	}

	_, stdErr, err := k.Cmd.RunCommandWithStdErr(context.Background(), nil, 0, "", "sudo", "systemctl", "daemon-reload")
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
	err = os.WriteFile(constants.KubeletEnvPath, []byte(kubeletEnv), 0770)
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

	// create the file
	_, stdErr, err := k.Cmd.RunCommandWithStdErr(context.Background(), nil, 0, "", "sudo", "touch", constants.SystemdRuntimeUnitDir+"/pf9-kubelet.service")
	if err != nil {
		zap.S().Panicf("failed to create pf9-kubelet.service %s, %s\n", err, stdErr[0])
		return err
	}
	zap.S().Debugf("created pf9-kubelet.service")

	// sudo chown pf9:pf9group /run/systemd/system/pf9-kubelet.service

	// own file
	usrgrp := constants.Pf9User + ":" + constants.Pf9Group
	_, stdOut, stdErr, err := k.Cmd.RunCommandWithStdOutStdErr(context.Background(), nil, 0, "", "sudo", "chown", usrgrp, constants.SystemdRuntimeUnitDir+"/pf9-kubelet.service")
	if err != nil {
		zap.S().Panicf("failed to own pf9 kubelet service file. %s %s\n", stdErr[0], err)
		return err
	}
	zap.S().Debugf("pf9 user ownes pf9 kubelet service file \n%s", stdOut)

	// change permissions
	_, stdOut, stdErr, err = k.Cmd.RunCommandWithStdOutStdErr(context.Background(), nil, 0, "", "sudo", "chmod", "770", constants.SystemdRuntimeUnitDir+"/pf9-kubelet.service")
	if err != nil {
		zap.S().Panicf("failed to change permissions for pf9 kubelet service file. %s %s\n", stdErr[0], err)
		return err
	}
	zap.S().Debugf("changed permissions for pf9 kubelet service file \n%s", stdOut)

	// write file
	err = os.WriteFile(constants.SystemdRuntimeUnitDir+"/pf9-kubelet.service", []byte(pf9KubeletService), 0770)
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
	_, stdErr, err := k.Cmd.RunCommandWithStdErr(context.Background(), nil, 0, "", "sudo", "systemctl", "start", "pf9-kubelet")
	if err != nil {
		zap.S().Panicf("failed to start pf9-kubelet. %s, %s\n", err, stdErr[0])
		return err
	}
	zap.S().Debugf("pf9-kubelet started.")
	return nil
}

func (k *KubeletImpl) KubeletStop() error {
	_, stdErr, err := k.Cmd.RunCommandWithStdErr(context.Background(), nil, 0, "", "sudo", "systemctl", "stop", "pf9-kubelet")
	if err != nil {
		zap.S().Panicf("failed to stop pf9-kubelet. %s, %s\n", err, stdErr[0])
		return err
	}
	zap.S().Debugf("pf9-kubelet stopped\n")
	return nil
}

// fixme
func (k *KubeletImpl) IsKubeletRunning() bool {
	_, stdOut, stdErr, err := k.Cmd.RunCommandWithStdOutStdErr(context.Background(), nil, 0, "", "sudo", "systemctl", "is-active", "pf9-kubelet")
	if stdOut[0] == "inactive" /*err != nil*/ {
		zap.S().Debugf("pf9-kubelet is not active.%s, %s\n", err, stdErr[0])
		return false
	}
	zap.S().Debugf("pf9-kubelet is active. %s\n", stdOut)
	return true
}
