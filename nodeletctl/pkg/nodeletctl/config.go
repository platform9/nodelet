package nodeletctl

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"go.uber.org/zap"
)

type NodeletConfig struct {
	AllowWorkloadsOnMaster bool
	CalicoV4Interface      string
	CalicoV6Interface      string
	ClusterId              string
	ContainerRuntime       ContainerRuntimeConfig
	EtcdClusterState       string
	HostId                 string
	HostIp                 string
	K8sApiPort             string
	MasterList             *map[string]string
	MasterIp               string
	MasterVipEnabled       bool
	MasterVipInterface     string
	MasterVipVrouterId     int
	Mtu                    string
	Privileged             string
	NodeletRole            string
	UserImages             []string
	CoreDNSHostsFile       string
	IPv6Enabled            bool
	UseHostname            bool
	CalicoIP4              string
	CalicoIP6              string
	CalicoV4BlockSize      int
	CalicoV6BlockSize      int
	CalicoV6ContainersCidr string
	CalicoV4ContainersCidr string
	CalicoV4NATOutgoing    bool
	CalicoV6NATOutgoing    bool
	CalicoV4IpIpMode       string
	ContainersCidr         string
	ServicesCidr           string
}

func setNodeletClusterCfg(cfg *BootstrapConfig, nodelet *NodeletConfig) {
	nodelet.AllowWorkloadsOnMaster = cfg.AllowWorkloadsOnMaster
	nodelet.ClusterId = cfg.ClusterId
	nodelet.ContainerRuntime = cfg.ContainerRuntime
	nodelet.K8sApiPort = cfg.K8sApiPort
	nodelet.MasterIp = cfg.MasterIp
	nodelet.MasterVipEnabled = cfg.MasterVipEnabled
	nodelet.MasterVipInterface = cfg.MasterVipInterface
	nodelet.MasterVipVrouterId = cfg.MasterVipVrouterId
	nodelet.Mtu = cfg.MTU
	nodelet.Privileged = cfg.Privileged
	nodelet.UserImages = cfg.UserImages
	nodelet.CoreDNSHostsFile = cfg.DNS.HostsFile
	nodelet.IPv6Enabled = cfg.IPv6Enabled

	//Set default Calico opts first
	nodelet.CalicoV4Interface = cfg.Calico.V4Interface
	nodelet.CalicoV4BlockSize = cfg.Calico.V4BlockSize
	nodelet.CalicoV4IpIpMode = cfg.Calico.V4IpIpMode
	nodelet.CalicoV4NATOutgoing = cfg.Calico.V4NATOutgoing
	nodelet.ContainersCidr = cfg.Calico.V4ContainersCidr
	nodelet.CalicoV6Interface = cfg.Calico.V6Interface
	nodelet.CalicoV6BlockSize = cfg.Calico.V6BlockSize
	nodelet.CalicoV6NATOutgoing = cfg.Calico.V6NATOutgoing
	nodelet.CalicoV6ContainersCidr = cfg.Calico.V6ContainersCidr

	if cfg.IPv6Enabled {
		// Always use hostname as node identifier for IPv6
		nodelet.UseHostname = true
		// Disable IPv4 as dualstack not yet supported
		nodelet.CalicoIP4 = "none"
		nodelet.CalicoIP6 = "autodetect"

		// Need to set this field for v6, as it is used to set kube-proxy arg
		nodelet.ContainersCidr = cfg.Calico.V6ContainersCidr
		if cfg.ServicesCidr == "" {
			nodelet.ServicesCidr = DefaultV6ServicesCidr
			cfg.ServicesCidr = DefaultV6ServicesCidr
		} else {
			nodelet.ServicesCidr = cfg.ServicesCidr
		}
	} else {
		// IPv4 only
		nodelet.UseHostname = cfg.UseHostname
		nodelet.CalicoIP4 = "autodetect"
		nodelet.CalicoIP6 = "none"
		if cfg.ServicesCidr == "" {
			nodelet.ServicesCidr = DefaultV4ServicesCidr
			cfg.ServicesCidr = DefaultV4ServicesCidr
		} else {
			nodelet.ServicesCidr = cfg.ServicesCidr
		}
	}
}

func GenNodeletConfigLocal(host *NodeletConfig, templateName string) (string, error) {
	nodeStateDir := filepath.Join(ClusterStateDir, host.ClusterId, host.HostId)
	if _, err := os.Stat(nodeStateDir); os.IsNotExist(err) {
		zap.S().Infof("Creating node state dir: %s\n", nodeStateDir)
		if err := os.MkdirAll(nodeStateDir, 0777); err != nil {
			return "", fmt.Errorf("Failed to create node state dir for host %s: %s", host.HostId, err)
		}
	}

	nodeletCfgFile := filepath.Join(nodeStateDir, NodeletConfigFile)

	t := template.Must(template.New(host.HostId).Parse(templateName))

	fd, err := os.Create(nodeletCfgFile)
	if err != nil {
		return "", fmt.Errorf("Failed to Create nodelet config File: %s err: %s", nodeletCfgFile, err)
	}
	defer fd.Close()

	err = t.Execute(fd, host)
	if err != nil {
		return "", fmt.Errorf("template.Execute failed for file: %s err: %s\n", nodeletCfgFile, err)
	}

	return nodeletCfgFile, nil
}
