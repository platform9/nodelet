package nodeletctl

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"text/template"
	"time"

	"github.com/ghodss/yaml"
	"go.uber.org/zap"
)

type BootstrapConfig struct {
	SSHUser                string                 `json:"sshUser,omitempty"`
	SSHPrivateKeyFile      string                 `json:"sshPrivateKeyFile,omitempty"`
	CertsDir               string                 `json:"certsDir,omitempty"`
	KubeConfig             string                 `json:"kubeconfig,omitempty"`
	Pf9KubePkg             string                 `json:"nodeletPkg,omitempty"`
	ClusterId              string                 `json:"clusterName,omitempty"`
	AllowWorkloadsOnMaster bool                   `json:"allowWorkloadsOnMaster,omitempty"`
	K8sApiPort             string                 `json:"k8sApiPort,omitempty"`
	MasterIp               string                 `json:"masterIp,omitempty"`
	MasterVipEnabled       bool                   `json:"masterVipEnabled,omitempty"`
	MasterVipInterface     string                 `json:"masterVipInterface,omitempty"`
	MasterVipVrouterId     int                    `json:"masterVipVrouterId,omitempty"`
	CalicoV4Interface      string                 `json:"calicoV4Interface,omitempty"`
	CalicoV6Interface      string                 `json:"calicoV6Interface,omitempty"`
	MTU                    string                 `json:"mtu,omitempty"`
	Privileged             string                 `json:"privileged,omitempty"`
	ContainerRuntime       ContainerRuntimeConfig `json:"containerRuntime,omitempty"`
	UserImages             []string               `json:"userImages,omitempty"`
	DNS                    CoreDNSConfig          `json:"dns,omitempty"`
	MasterNodes            []HostConfig           `json:"masterNodes"`
	WorkerNodes            []HostConfig           `json:"workerNodes"`
}

type CoreDNSConfig struct {
	HostsFile   string   `json:"hostsFile,omitempty"`
	InlineHosts []string `json:"corednsHosts,omitempty"`
}

type ContainerRuntimeConfig struct {
	Name         string `json:"name,omitempty"`
	CgroupDriver string `json:"cgroupDriver,omitempty"`
}

type HostConfig struct {
	NodeName            string  `json:"nodeName"`
	NodeIP              *string `json:"nodeIP,omitempty"`
	V4InterfaceOverride *string `json:"calicoV4Interface,omitempty"`
	V6InterfaceOverride *string `json:"calicoV6Interface,omitempty"`
}

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
}

type ClusterStatus struct {
	statusMap map[string]*NodeStatus
}

type NodeStatus struct {
	nodeHealth string
	errMsg     error
	deployer   *NodeletDeployer
}

var globalClusterStatus *ClusterStatus

func CreateCluster(cfgPath string) error {
	clusterCfg, err := ParseBootstrapConfig(cfgPath)
	if err != nil {
		zap.S().Infof("Failed to Parse Cluster Config: %s", err)
		return fmt.Errorf("Failed to Parse Cluster Config: %s", err)
	}

	clusterStateDir := filepath.Join(ClusterStateDir, clusterCfg.ClusterId)
	if _, err := os.Stat(clusterStateDir); err == nil {
		zap.S().Warnf("Found pre-existing cluster state directory %s, re-using certs. Consider removing or scale or upgrade operations", clusterStateDir)
		fmt.Printf("Found pre-existing cluster state directory %s, re-using certs. Consider removing or scale or upgrade operations", clusterStateDir)
	}

	masters, err := GetCurrentMasters(clusterCfg)
	if err == nil {
		if len(masters) > 0 {
			return fmt.Errorf("Found an already active cluster with masters: %v\nUse nodeletctl delete first", masters)
		}
	}

	if err := DeployCluster(clusterCfg); err != nil {
		zap.S().Infof("Cluster failed: %s\n", err)
		return fmt.Errorf("Cluster failed: %s", err)
	}

	if err := clusterCfg.saveClusterConfig(); err != nil {
		zap.S().Errorf("Failed to save cluster config: %s", err)
		return err
	}

	return nil
}

func UpgradeCluster(cfgPath string) error {

	clusterCfg, err := ParseBootstrapConfig(cfgPath)
	if err != nil {
		zap.S().Infof("Failed to Parse Cluster Config: %s", err)
		return fmt.Errorf("Failed to Parse Cluster Config: %s", err)
	}

	// TODO: Need to validate if nodelet version in the bootstrap is the supported version, possibly by
	// checking a manifest file in the rpm/deb package

	if err != nil {
		return fmt.Errorf("failed to DetermineKubePkgName: %s", err)
	}

	globalClusterStatus = new(ClusterStatus)
	globalClusterStatus.statusMap = make(map[string]*NodeStatus)

	var masterList = make(map[string]string)
	for _, host := range clusterCfg.MasterNodes {
		if host.NodeIP != nil {
			masterList[host.NodeName] = *host.NodeIP
		} else {
			masterList[host.NodeName] = host.NodeName
		}
	}

	masters, err := GetCurrentMasters(clusterCfg)
	if err != nil {
		return fmt.Errorf("Failed to get active K8s masters: %s", err)
	}

	workers, err := GetCurrentWorkers(clusterCfg)
	if err != nil {
		return fmt.Errorf("Failed to get active K8s workers: %s", err)
	}

	if len(clusterCfg.MasterNodes) != len(masters) {
		return fmt.Errorf("Masters in the cluster and Bootstrap config doesnt match")
	}
	newMasters, oldMasters, _ := getDiffNodes(clusterCfg.MasterNodes, masters)
	if len(newMasters) != 0 || len(oldMasters) != 0 {
		return fmt.Errorf("Masters in the cluster and Bootstrap config doesnt match")
	}

	if len(clusterCfg.WorkerNodes) != len(workers) {
		return fmt.Errorf("Workers in the cluster and Bootstrap config doesnt match")
	}
	newWorkers, oldWorkers, _ := getDiffNodes(clusterCfg.WorkerNodes, workers)
	if len(newWorkers) != 0 || len(oldWorkers) != 0 {
		return fmt.Errorf("Workers in the cluster and Bootstrap config doesnt match")
	}

	// Upgrade each master node sequentially
	for _, host := range clusterCfg.MasterNodes {
		zap.S().Infof("Upgrading master node: %s", host.NodeName)
		// Build nodeletconfig and deployer
		nodeletCfg := new(NodeletConfig)
		setNodeletClusterCfg(clusterCfg, nodeletCfg)
		nodeletCfg.HostId = host.NodeName
		if host.NodeIP != nil {
			nodeletCfg.HostIp = *host.NodeIP
		} else {
			nodeletCfg.HostIp = host.NodeName
		}
		nodeletCfg.NodeletRole = "master"
		nodeletCfg.MasterList = &masterList
		nodeletCfg.EtcdClusterState = "new"

		nodeletSrcFile, err := GenNodeletConfigLocal(nodeletCfg, masterNodeletConfigTmpl)
		if err != nil {
			zap.S().Infof("Failed to generate config: %s", err)
			return fmt.Errorf("Failed to generate config: %s", err)
		}
		zap.S().Debugf("master nodeletsrc file %s", nodeletSrcFile)

		deployer, err := GetNodeletDeployer(clusterCfg, globalClusterStatus, nodeletCfg, nodeletCfg.HostIp, nodeletSrcFile)
		if err != nil {
			zap.S().Errorf("failed to get nodelet deployer: %v", err)
			return fmt.Errorf("failed to get nodelet deployer: %v", err)
		}
		globalClusterStatus.statusMap[host.NodeName] = &NodeStatus{
			deployer: deployer,
		}

		if err := deployer.UpgradeMaster(); err != nil {
			SetClusterNodeStatus(deployer.clusterStatus, deployer.nodeletCfg.HostId, "failed", err)
		}
	}

	if err := UpgradeWorkers(clusterCfg, globalClusterStatus); err != nil {
		zap.S().Infof("Failed to upgrade workers: %s", err)
		return fmt.Errorf("Failed to upgrade workers: %s", err)
	}

	SyncNodes(clusterCfg, nil)

	zap.S().Infof("Cluster Upgraded successfully")
	return nil
}

func UpgradeWorkers(clusterCfg *BootstrapConfig, clusterStatus *ClusterStatus) error {
	// Upgrade each worker node parallely
	var wg sync.WaitGroup
	for _, host := range clusterCfg.WorkerNodes {
		zap.S().Infof("Upgrading worker: %s", host.NodeName)
		nodeletCfg := new(NodeletConfig)
		setNodeletClusterCfg(clusterCfg, nodeletCfg)
		nodeletCfg.HostId = host.NodeName
		nodeletCfg.NodeletRole = "worker"
		nodeletSrcFile, err := GenNodeletConfigLocal(nodeletCfg, workerNodeletConfigTmpl)
		if err != nil {
			zap.S().Infof("Failed to generate config: %s", err)
			return fmt.Errorf("Failed to generate config: %s", err)
		}
		deployer, err := GetNodeletDeployer(clusterCfg, clusterStatus, nodeletCfg, host.NodeName, nodeletSrcFile)
		if err != nil {
			zap.S().Errorf("failed to get nodelet deployer: %s", err)
			return fmt.Errorf("failed to get nodelet deployer: %s", err)
		}
		clusterStatus.statusMap[host.NodeName] = &NodeStatus{
			deployer: deployer,
		}
		wg.Add(1)
		go deployer.UpgradeWorker(&wg)
	}
	wg.Wait()
	return nil
}
func InitBootstrapConfig() *BootstrapConfig {
	bootstrapCfg := &BootstrapConfig{
		AllowWorkloadsOnMaster: false,
		CalicoV4Interface:      "first-found",
		CalicoV6Interface:      "first-found",
		ClusterId:              DefaultClusterName,
		ContainerRuntime:       ContainerRuntimeConfig{"containerd", "systemd"},
		SSHUser:                "root",
		SSHPrivateKeyFile:      "/root/.ssh/id_rsa",
		Pf9KubePkg:             NodeletTarSrc,
		Privileged:             "true",
		K8sApiPort:             "443",
		MasterVipEnabled:       false,
		MTU:                    "1440",
	}
	return bootstrapCfg
}

func ParseBootstrapConfig(cfgPath string) (*BootstrapConfig, error) {
	zap.S().Infof("ParseBootstrapConfig cfgPath: %s", cfgPath)
	cfgFile, err := ioutil.ReadFile(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("Error opening bootstrap config file: %s", cfgFile)
	}

	bootstrapConfig := InitBootstrapConfig()
	err = yaml.Unmarshal(cfgFile, bootstrapConfig)
	if err != nil {
		return nil, fmt.Errorf("error decoding bootstrap config: %v", err)
	}

	if err := isClusterCfgValid(bootstrapConfig); err != nil {
		return nil, fmt.Errorf("Invalid cluster config: %s", err)
	}

	if len(bootstrapConfig.DNS.InlineHosts) > 0 {
		bootstrapConfig.DNS.HostsFile, err = WriteHostsFileForEntries(bootstrapConfig.ClusterId, bootstrapConfig.DNS.InlineHosts)
		if err != nil {
			return nil, fmt.Errorf("Failed to generate custom hosts file: %s", err)
		}
	}

	return bootstrapConfig, nil
}

func DeployCluster(clusterCfg *BootstrapConfig) error {
	zap.S().Infof("Deploying cluster %s", clusterCfg.ClusterId)
	if clusterCfg.CertsDir == "" {
		certsDir, err := GenCALocal(clusterCfg.ClusterId)
		if err != nil {
			return fmt.Errorf("Cert generation failed: %s\n", err)
		}
		clusterCfg.CertsDir = certsDir
	}

	err := trustCA(clusterCfg.CertsDir)
	if err != nil {
		zap.S().Errorf("error adding nodelet Root CA as trusted certs: %s\n", err)
		return fmt.Errorf("error adding nodelet Root CA as trusted certs: %s\n", err)
	}

	if err := GenKubeconfig(clusterCfg); err != nil {
		zap.S().Infof("Failed to generate kubeconfig: %s\n", err)
		return err
	}

	globalClusterStatus = new(ClusterStatus)
	globalClusterStatus.statusMap = make(map[string]*NodeStatus)
	var masterList = make(map[string]string)
	for _, host := range clusterCfg.MasterNodes {
		if host.NodeIP != nil {
			masterList[host.NodeName] = *host.NodeIP
		} else {
			masterList[host.NodeName] = host.NodeName
		}
	}

	for numMaster, host := range clusterCfg.MasterNodes {
		zap.S().Infof("Deploying master node %s", host.NodeName)
		nodeletCfg := new(NodeletConfig)
		setNodeletClusterCfg(clusterCfg, nodeletCfg)
		nodeletCfg.HostId = host.NodeName
		if host.NodeIP != nil {
			nodeletCfg.HostIp = *host.NodeIP
		} else {
			nodeletCfg.HostIp = host.NodeName
		}
		nodeletCfg.NodeletRole = "master"
		nodeletCfg.MasterList = &masterList
		nodeletCfg.EtcdClusterState = "new"

		nodeletSrcFile, err := GenNodeletConfigLocal(nodeletCfg, masterNodeletConfigTmpl)
		if err != nil {
			zap.S().Infof("Failed to generate config: %s", err)
			return fmt.Errorf("Failed to generate config: %s", err)
		}
		zap.S().Debugf("master nodeletsrc file %s", nodeletSrcFile)
		deployer, err := GetNodeletDeployer(clusterCfg, globalClusterStatus, nodeletCfg, nodeletCfg.HostIp, nodeletSrcFile)
		if err != nil {
			zap.S().Errorf("failed to get nodelet deployer: %v", err)
			return fmt.Errorf("failed to get nodelet deployer: %v", err)
		}
		globalClusterStatus.statusMap[host.NodeName] = &NodeStatus{
			deployer: deployer,
		}

		converged, err := deployer.SpawnMaster(numMaster)
		zap.S().Infof("Master status: %s\n", converged)
		if err != nil {
			zap.S().Infof("err = %s\n", err)
		}
	}

	if err := DeployWorkers(clusterCfg, globalClusterStatus, &clusterCfg.WorkerNodes); err != nil {
		return fmt.Errorf("ScaleCluster failed to deploy new workers: %s", err)
	}

	SyncNodes(clusterCfg, nil)
	zap.S().Infof("Cluster deployed successfully")
	return nil
}

func SetClusterNodeStatus(status *ClusterStatus, nodeName, health string, err error) {
	status.statusMap[nodeName].nodeHealth = health
	status.statusMap[nodeName].errMsg = err
}

func SyncAndRetry(clusterCfg *BootstrapConfig, nodeletStatus *ClusterStatus, nodesToSync *[]HostConfig, done chan bool) {
	var synced bool = true
	// TODO: For now just display status's... retry DeployNodelet?
	for _, node := range *nodesToSync {
		nodeHealth := nodeletStatus.statusMap[node.NodeName].nodeHealth
		zap.S().Infof("Node %s in state: %s\n", node.NodeName, nodeHealth)
		if nodeHealth != NodeHealthy {
			if nodeletStatus.statusMap[node.NodeName].errMsg != nil {
				zap.S().Infof("Error: %s\n\n", nodeletStatus.statusMap[node.NodeName].errMsg)
			}
			nd := nodeletStatus.statusMap[node.NodeName].deployer
			newStatus, newErr := nd.RefreshNodeletStatus()
			SetClusterNodeStatus(nodeletStatus, node.NodeName, newStatus, newErr)
			synced = false
		}
	}

	if !synced {
		zap.S().Infof("Nodes are not synced, will re-check...\n")
		// TODO: After how long do we give up?
		return
	}
	close(done)
}

func setNodeletClusterCfg(cfg *BootstrapConfig, nodelet *NodeletConfig) {
	nodelet.AllowWorkloadsOnMaster = cfg.AllowWorkloadsOnMaster
	nodelet.CalicoV4Interface = cfg.CalicoV4Interface
	nodelet.CalicoV6Interface = cfg.CalicoV6Interface
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

func DeleteCluster(cfgPath string) error {
	clusterCfg, err := ParseBootstrapConfig(cfgPath)
	if err != nil {
		zap.S().Infof("Failed to Parse Cluster Config: %s", err)
		return fmt.Errorf("Failed to Parse Cluster Config: %s", err)
	}

	allNodes := append(clusterCfg.WorkerNodes, clusterCfg.MasterNodes...)
	deleteFailed := false

	for _, host := range allNodes {
		deployer, err := GetNodeletDeployer(clusterCfg, nil, nil, host.NodeName, "")
		if err != nil {
			zap.S().Errorf("failed to get nodelet deployer: %v", err)
			return fmt.Errorf("failed to get nodelet deployer: %v", err)
		}
		zap.S().Infof("Cleaning up node %s", host.NodeName)
		err = deployer.DeleteNodelet()
		if err != nil {
			zap.S().Infof("Failed to delete node %s: %s\n", host.NodeName, err)
			deleteFailed = true
		}
	}

	if deleteFailed {
		return fmt.Errorf("Cluster delete failed. Please check logs for further fetails")
	}
	return nil
}

func GetClusterNodeletStatus(clusterCfg *BootstrapConfig, nodes *[]HostConfig) (*ClusterStatus, error) {

	nodeletStatus := new(ClusterStatus)
	nodeletStatus.statusMap = make(map[string]*NodeStatus)

	for _, host := range *nodes {
		nodeletCfg := new(NodeletConfig)
		setNodeletClusterCfg(clusterCfg, nodeletCfg)
		nodeletCfg.HostId = host.NodeName
		if host.NodeIP != nil {
			nodeletCfg.HostIp = *host.NodeIP
		} else {
			nodeletCfg.HostIp = host.NodeName
		}

		deployer, err := GetNodeletDeployer(clusterCfg, nodeletStatus, nodeletCfg, nodeletCfg.HostIp, "")
		if err != nil {
			zap.S().Errorf("failed to get nodelet deployer: %v", err)
			return nil, fmt.Errorf("failed to get nodelet deployer: %v", err)
		}
		nodeletStatus.statusMap[host.NodeName] = &NodeStatus{
			deployer: deployer,
		}
		zap.S().Infof("Fetching status for node %s", host.NodeName)
		nodeHealth, err := deployer.RefreshNodeletStatus()
		if err != nil {
			zap.S().Infof("Node %s not yet authorized", host.NodeName)
			continue
		}

		SetClusterNodeStatus(nodeletStatus, host.NodeName, nodeHealth, nil)
	}

	return nodeletStatus, nil
}

func GetCurrentWorkers(clusterCfg *BootstrapConfig) ([]string, error) {
	client, err := GetClient(clusterCfg)
	if err != nil {
		return nil, err
	}

	return client.GetMatchingNodes(WorkerLabel)
}

func GetCurrentMasters(clusterCfg *BootstrapConfig) ([]string, error) {
	client, err := GetClient(clusterCfg)
	if err != nil {
		return nil, err
	}

	return client.GetMatchingNodes(MasterLabel)
}

func ScaleCluster(cfgPath string) error {
	clusterCfg, err := ParseBootstrapConfig(cfgPath)
	if err != nil {
		zap.S().Infof("Failed to Parse Cluster Config: %s", err)
		return fmt.Errorf("Failed to Parse Cluster Config: %s", err)
	}

	if clusterCfg.CertsDir == "" && !CertsExist(clusterCfg.ClusterId) {
		return fmt.Errorf("Could not find existing certs for cluster %s", clusterCfg.ClusterId)
	} else if clusterCfg.CertsDir == "" {
		clusterCfg.CertsDir = filepath.Join(ClusterStateDir, clusterCfg.ClusterId, "certs")
	}

	nodeletStatus := new(ClusterStatus)
	nodeletStatus.statusMap = make(map[string]*NodeStatus)

	masters, err := GetCurrentMasters(clusterCfg)
	if err != nil {
		return fmt.Errorf("Failed to get active K8s masters: %s", err)
	}

	workers, err := GetCurrentWorkers(clusterCfg)
	if err != nil {
		return fmt.Errorf("Failed to get active K8s workers: %s", err)
	}

	newMasters, oldMasters, currMasters := getDiffNodes(clusterCfg.MasterNodes, masters)
	newWorkers, oldWorkers, _ := getDiffNodes(clusterCfg.WorkerNodes, workers)

	if err := AddMasters(clusterCfg, nodeletStatus, &currMasters, &newMasters); err != nil {
		return fmt.Errorf("ScaleCluster failed to deploy new masters: %s", err)
	}

	if err := RemoveMasters(clusterCfg, nodeletStatus, &currMasters, &oldMasters); err != nil {
		return fmt.Errorf("ScaleCluster failed to remove old masters: %s", err)
	}

	if err := DeployWorkers(clusterCfg, nodeletStatus, &newWorkers); err != nil {
		return fmt.Errorf("ScaleCluster failed to deploy new workers: %s", err)
	}

	// This blocks until all nodes are converged/ok
	if err := SyncNodes(clusterCfg, nil); err != nil {
		return err
	}

	if err := DeleteWorkers(clusterCfg, oldWorkers); err != nil {
		return fmt.Errorf("ScaleCluster failed to cleanup old nodes: %s", err)
	}

	return nil
}

func getDiffNodes(desired []HostConfig, active []string) ([]HostConfig, []HostConfig, []HostConfig) {
	activeNodes := make(map[string]struct{})
	desiredNodes := make(map[string]HostConfig)
	newNodes := []HostConfig{}
	oldNodes := []HostConfig{}
	currNodes := []HostConfig{}

	for _, masterName := range active {
		activeNodes[masterName] = struct{}{}
	}
	for _, hostConfig := range desired {
		desiredNodes[hostConfig.NodeName] = hostConfig
	}

	for nodeName := range activeNodes {
		if _, ok := desiredNodes[nodeName]; !ok {
			node := HostConfig{NodeName: nodeName}
			oldNodes = append(oldNodes, node)
		}
	}
	for nodeName, hostConfig := range desiredNodes {
		if _, ok := activeNodes[nodeName]; !ok {
			newNodes = append(newNodes, hostConfig)
		} else if ok {
			currNodes = append(currNodes, hostConfig)
		}
	}

	zap.S().Infof("newNodes: %#v\n", newNodes)
	zap.S().Infof("oldNodes: %#v\n", oldNodes)
	zap.S().Infof("currNodes: %#v\n", currNodes)

	return newNodes, oldNodes, currNodes
}

func AddMasters(clusterCfg *BootstrapConfig, clusterStatus *ClusterStatus, currMasters, newMasters *[]HostConfig) error {
	zap.S().Infof("Adding %d masters", len(*newMasters))
	var masterList = make(map[string]string)
	for _, host := range *currMasters {
		if host.NodeIP != nil {
			masterList[host.NodeName] = *host.NodeIP
		} else {
			masterList[host.NodeName] = host.NodeName
		}
	}

	for numMaster, host := range *newMasters {
		zap.S().Infof("Adding master %s", host.NodeName)
		nodeletCfg := new(NodeletConfig)
		setNodeletClusterCfg(clusterCfg, nodeletCfg)
		nodeletCfg.HostId = host.NodeName
		if host.NodeIP != nil {
			nodeletCfg.HostIp = *host.NodeIP
		} else {
			nodeletCfg.HostIp = host.NodeName
		}
		nodeletCfg.NodeletRole = "master"
		nodeletCfg.EtcdClusterState = "existing"
		masterList[host.NodeName] = nodeletCfg.HostIp
		nodeletCfg.MasterList = &masterList
		nodeletSrcFile, err := GenNodeletConfigLocal(nodeletCfg, masterNodeletConfigTmpl)
		if err != nil {
			zap.S().Infof("Failed to generate config: %s", err)
			return fmt.Errorf("Failed to generate config: %s", err)
		}
		zap.S().Debugf("nodelet src file %s", nodeletSrcFile)

		deployer, err := GetNodeletDeployer(clusterCfg, clusterStatus, nodeletCfg, nodeletCfg.HostIp, nodeletSrcFile)
		if err != nil {
			zap.S().Errorf("failed to get nodelet deployer: %v", err)
			return fmt.Errorf("failed to get nodelet deployer: %v", err)
		}
		clusterStatus.statusMap[host.NodeName] = &NodeStatus{
			deployer: deployer,
		}
		zap.S().Infof("Added master %s to etcd ", host.NodeName)
		if err := AddNodeToEtcd(clusterCfg, currMasters, nodeletCfg.HostIp); err != nil {
			return fmt.Errorf("Failed to add nodes %+v as etcd members: %s", newMasters, err)
		}
		zap.S().Infof("Spawning master %s", host.NodeName)
		_, _ = deployer.SpawnMaster(numMaster)
		if err := SyncNodes(clusterCfg, &[]HostConfig{host}); err != nil {
			return fmt.Errorf("Failed to sync new master: %+v", host)
		}

		*currMasters = append(*currMasters, host)

	}
	return nil
}

func RemoveMasters(clusterCfg *BootstrapConfig, clusterStatus *ClusterStatus, currMasters, oldMasters *[]HostConfig) error {
	for _, host := range *oldMasters {
		nodeletCfg := new(NodeletConfig)
		setNodeletClusterCfg(clusterCfg, nodeletCfg)
		nodeletCfg.HostId = host.NodeName
		if host.NodeIP != nil {
			nodeletCfg.HostIp = *host.NodeIP
		} else {
			nodeletCfg.HostIp = host.NodeName
		}
		if err := RemoveNodeFromEtcd(clusterCfg, currMasters, nodeletCfg.HostIp); err != nil {
			return fmt.Errorf("Failed to remove nodes %+v from etcd members: %s", host, err)
		}

		deployer, err := GetNodeletDeployer(clusterCfg, nil, nil, host.NodeName, "")
		if err != nil {
			zap.S().Errorf("failed to get nodelet deployer: %v", err)
			return fmt.Errorf("failed to get nodelet deployer: %v", err)
		}
		zap.S().Infof("Removing node %s from cluster %s", host.NodeName, clusterCfg.ClusterId)
		err = deployer.DeleteNodelet()
		if err != nil {
			return fmt.Errorf("Failed to delete node %s: %s\n", host.NodeName, err)
		}

		// Remove the deleted master from slice of current active masters
		// Needed for next iteration of loop to keep etcd masters up to date
		for i, master := range *currMasters {
			if host.NodeName == master.NodeName {
				(*currMasters)[i] = (*currMasters)[len(*currMasters)-1]
				*currMasters = (*currMasters)[:len(*currMasters)-1]
				break
			}
		}
		// Resync and check new current masters to make sure OK
		SyncNodes(clusterCfg, currMasters)
	}
	return nil
}

func DeployWorkers(clusterCfg *BootstrapConfig, clusterStatus *ClusterStatus, workers *[]HostConfig) error {
	var wg sync.WaitGroup

	for _, host := range *workers {
		nodeletCfg := new(NodeletConfig)
		setNodeletClusterCfg(clusterCfg, nodeletCfg)
		nodeletCfg.HostId = host.NodeName
		nodeletCfg.NodeletRole = "worker"
		nodeletSrcFile, err := GenNodeletConfigLocal(nodeletCfg, workerNodeletConfigTmpl)
		if err != nil {
			zap.S().Infof("Failed to generate config: %s", err)
			return fmt.Errorf("Failed to generate config: %s", err)
		}
		deployer, err := GetNodeletDeployer(clusterCfg, clusterStatus, nodeletCfg, host.NodeName, nodeletSrcFile)
		if err != nil {
			zap.S().Errorf("failed to get nodelet deployer: %s", err)
			return fmt.Errorf("failed to get nodelet deployer: %s", err)
		}
		clusterStatus.statusMap[host.NodeName] = &NodeStatus{
			deployer: deployer,
		}
		wg.Add(1)
		zap.S().Infof("Adding worker %s to cluster %s", host.NodeName, clusterCfg.ClusterId)
		go deployer.SpawnWorker(&wg)
	}

	wg.Wait()
	return nil
}

func DeleteWorkers(clusterCfg *BootstrapConfig, oldNodes []HostConfig) error {
	for _, host := range oldNodes {
		deployer, err := GetNodeletDeployer(clusterCfg, nil, nil, host.NodeName, "")
		if err != nil {
			zap.S().Errorf("failed to get nodelet deployer: %v", err)
			return fmt.Errorf("failed to get nodelet deployer: %v", err)
		}
		zap.S().Infof("Removing node %s from cluster %s", host.NodeName, clusterCfg.ClusterId)
		err = deployer.DeleteNodelet()
		if err != nil {
			return fmt.Errorf("Failed to delete node %s: %s\n", host.NodeName, err)
		}
	}
	return nil
}

func RegenClusterCerts(cfgPath string) error {
	clusterCfg, err := ParseBootstrapConfig(cfgPath)
	if err != nil {
		zap.S().Infof("Failed to Parse Cluster Config: %s", err)
		return fmt.Errorf("Failed to Parse Cluster Config: %s", err)
	}

	err = RegenCA(clusterCfg)
	if err != nil {
		zap.S().Errorf("Failed to regenerate new CA: %s", err)
		return fmt.Errorf("Failed to regenerate new CA: %s", err)
	}

	err = trustCA(clusterCfg.CertsDir)
	if err != nil {
		zap.S().Errorf("error adding nodelet Root CA as trusted certs: %s\n", err)
		return fmt.Errorf("error adding nodelet Root CA as trusted certs: %s\n", err)
	}

	clusterStatus := new(ClusterStatus)
	clusterStatus.statusMap = make(map[string]*NodeStatus)

	firstMaster := []HostConfig{clusterCfg.MasterNodes[0]}
	otherMasters := clusterCfg.MasterNodes[1:]

	if err := regenCertsForHosts(clusterCfg, otherMasters); err != nil {
		zap.S().Errorf("Failed to regen certs for masters %+v: err", otherMasters)
		return fmt.Errorf("Failed to regen certs for masters %+v: err", otherMasters)
	}

	if err := regenCertsForHosts(clusterCfg, clusterCfg.WorkerNodes); err != nil {
		zap.S().Errorf("Failed to regen certs for masters %+v: err", otherMasters)
		return fmt.Errorf("Failed to regen certs for masters %+v: err", otherMasters)
	}

	if err := regenCertsForHosts(clusterCfg, firstMaster); err != nil {
		zap.S().Errorf("Failed to regen certs for masters %+v: err", otherMasters)
		return fmt.Errorf("Failed to regen certs for masters %+v: err", otherMasters)
	}

	return nil
}

func regenCertsForHosts(clusterCfg *BootstrapConfig, nodes []HostConfig) error {
	var wg sync.WaitGroup

	for _, host := range nodes {
		nodeletCfg := new(NodeletConfig)
		setNodeletClusterCfg(clusterCfg, nodeletCfg)
		nodeletCfg.HostId = host.NodeName
		if host.NodeIP != nil {
			nodeletCfg.HostIp = *host.NodeIP
		} else {
			nodeletCfg.HostIp = host.NodeName
		}

		deployer, err := GetNodeletDeployer(clusterCfg, nil, nodeletCfg, nodeletCfg.HostIp, "")
		if err != nil {
			zap.S().Errorf("failed to get nodelet deployer: %s", err)
			return fmt.Errorf("failed to get nodelet deployer: %s", err)
		}

		wg.Add(1)
		go deployer.UploadCertsAndRestartStack(&wg)
	}
	wg.Wait()
	if err := SyncNodes(clusterCfg, &nodes); err != nil {
		return err
	}
	return nil
}

func ConfigClusterDNS(cfgPath string) error {
	clusterCfg, err := ParseBootstrapConfig(cfgPath)
	if err != nil {
		zap.S().Infof("Failed to Parse Cluster Config: %s", err)
		return fmt.Errorf("Failed to Parse Cluster Config: %s", err)
	}

	if err := UploadHostsFile(clusterCfg); err != nil {
		zap.S().Infof("Cluster failed: %s\n", err)
		return fmt.Errorf("Cluster failed: %s", err)
	}

	return nil
}

func UploadHostsFile(clusterCfg *BootstrapConfig) error {
	failed := false
	for _, host := range clusterCfg.MasterNodes {
		deployer, err := GetNodeletDeployer(clusterCfg, nil, nil, host.NodeName, "")
		if err != nil {
			zap.S().Errorf("failed to get nodelet deployer: %v", err)
			return fmt.Errorf("failed to get nodelet deployer: %v", err)
		}
		zap.S().Infof("Cleaning up node %s", host.NodeName)
		err = deployer.UploadCoreDNSHostsFile()
		if err != nil {
			zap.S().Warnf("Failed to upload custom hosts file on %s: %s\n", host.NodeName, err)
			failed = false
		}

		err = deployer.RestartNodelet()
		if err != nil {
			zap.S().Warnf("Failed to restart pf9-nodeletd service on %s: %s\n", host.NodeName, err)
			failed = true
		}
	}
	if failed {
		return fmt.Errorf("Failed to upload custom hosts file or restart nodelet. Check node for details")
	}
	return nil
}

func WriteHostsFileForEntries(clusterName string, entries []string) (string, error) {
	hostsFile := filepath.Join(ClusterStateDir, clusterName, "hosts")
	fd, err := os.OpenFile(hostsFile, os.O_TRUNC|os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		zap.S().Errorf("Failed to open hosts file %s for writing: %s", hostsFile, err)
		return "", err
	}

	writer := bufio.NewWriter(fd)
	for _, line := range entries {
		_, err := writer.WriteString(line + "\n")
		if err != nil {
			fmt.Printf("Failed to write line %s\n", line)
			return "", err
		}
	}
	writer.Flush()
	return hostsFile, nil
}

func SyncNodes(clusterCfg *BootstrapConfig, nodes *[]HostConfig) error {
	// If nodes is nil, sync and wait for entire cluster
	var nodesToSync *[]HostConfig
	allNodes := append(clusterCfg.WorkerNodes, clusterCfg.MasterNodes...)
	if nodes != nil {
		nodesToSync = nodes
	} else {
		nodesToSync = &allNodes
	}

	nodeletStatus, err := GetClusterNodeletStatus(clusterCfg, nodesToSync)
	if err != nil {
		return fmt.Errorf("SyncNodes: Failed to populate initial cluster status: %s", err)
	}
	ticker := time.NewTicker(SyncRetrySeconds * time.Second)
	done := make(chan bool)

	go func() {
		for {
			select {
			case <-done:
				zap.S().Infof("Cluster converged successfully\n")
				return
			case <-ticker.C:
				zap.S().Infof("Syncing cluster state...\n")
				SyncAndRetry(clusterCfg, nodeletStatus, nodesToSync, done)
			}
		}
	}()
	<-done
	return nil
}

func isClusterCfgValid(clusterCfg *BootstrapConfig) error {
	if len(clusterCfg.MasterNodes) == 0 {
		return fmt.Errorf("Number of master nodes cannot be zero")
	}
	if (len(clusterCfg.MasterNodes) % 2) == 0 {
		return fmt.Errorf("Number of master nodes cannot be even")
	}
	if !clusterCfg.AllowWorkloadsOnMaster && len(clusterCfg.WorkerNodes) == 0 {
		return fmt.Errorf("Number of worker nodes cannot be zero when no workloads are allowed on masters")
	}
	if clusterCfg.MasterVipEnabled {
		if clusterCfg.MasterVipInterface == "" {
			return fmt.Errorf("MasterVipInterface cannot be empty when MasterVip is enabled")
		}
		if clusterCfg.MasterVipVrouterId < 1 || clusterCfg.MasterVipVrouterId > 255 {
			return fmt.Errorf("MasterVipVrouterId should be in the range 1-255 when MasterVip is enabled")
		}
	}
	return nil
}

func (cfg *BootstrapConfig) saveClusterConfig() error {
	bytes, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("Failed to marshal cluster config into YAML: %s", err)
	}

	clusterFileName := cfg.ClusterId + ".yaml"
	clusterFile := filepath.Join(ClusterStateDir, cfg.ClusterId, clusterFileName)

	if err := ioutil.WriteFile(clusterFile, bytes, 0644); err != nil {
		return fmt.Errorf("Failed to save updated cluster file: %s", err)
	}

	zap.S().Infof("Wrote %s", clusterFile)
	fmt.Printf("Saved updated cluster spec to %s\n", clusterFile)
	fmt.Printf("Please save a copy for further cluster operations\n")
	return nil
}
