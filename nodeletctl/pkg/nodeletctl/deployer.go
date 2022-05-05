package nodeletctl

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"path/filepath"
	"sync"
	"time"

	"github.com/platform9/pf9ctl/pkg/ssh"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/util/yaml"
    nodeletext "github.com/platform9/nodelet/nodelet/pkg/utils/extensionfile"
)

type NodeletDeployer struct {
	user           string
	OsType         string
	client         ssh.Client
	nodeletSrcFile string
	nodeletCfg     *NodeletConfig
	pf9KubeTarSrc  string
	clusterStatus  *ClusterStatus
}

func NewNodeletDeployer(cfg *BootstrapConfig, sshClient ssh.Client,
	srcFile string, nodeletCfg *NodeletConfig, clusterStatus *ClusterStatus) *NodeletDeployer {
	deployer := new(NodeletDeployer)
	deployer.user = cfg.SSHUser
	deployer.client = sshClient
	deployer.nodeletSrcFile = srcFile
	deployer.nodeletCfg = nodeletCfg
	deployer.pf9KubeTarSrc = cfg.Pf9KubePkg
	deployer.clusterStatus = clusterStatus
	deployer.SetOsType()

	return deployer
}

func GetNodeletDeployer(cfg *BootstrapConfig, clusterStatus *ClusterStatus, nodeletCfg *NodeletConfig, nodeName, nodeletSrcFile string, sshKey []byte) (*NodeletDeployer, error) {
	sshClient, err := CreateSSHClient(nodeName, cfg.SSHUser, sshKey, 22)
	if err != nil {
		return nil, fmt.Errorf("can't create ssh client to host %s, %s", nodeName, err)
	}

	deployer := NewNodeletDeployer(cfg, sshClient, nodeletSrcFile, nodeletCfg, clusterStatus)

	return deployer, nil
}

func (nd *NodeletDeployer) SpawnMaster(numMaster int) (string, error) {
	// Add any master-specific tasks here before and after
	if err := nd.DeployNodelet(); err != nil {
		SetClusterNodeStatus(nd.clusterStatus, nd.nodeletCfg.HostId, "failed", err)
		return "failed", err
	}

	if numMaster == 0 {
		// Multi-master: Add anything here only happen after deploying first master
		zap.S().Infof("First master being spawned")
	}
	status, err := nd.RefreshNodeletStatus()
	zap.S().Infof("Done spawning master")

	SetClusterNodeStatus(nd.clusterStatus, nd.nodeletCfg.HostId, status, err)
	return status, err
}

func (nd *NodeletDeployer) SpawnWorker(wg *sync.WaitGroup) {
	defer wg.Done()
	if err := nd.DeployNodelet(); err != nil {
		SetClusterNodeStatus(nd.clusterStatus, nd.nodeletCfg.HostId, "failed", err)
		return
	}
	// Add any worker-specific tasks here, if needed
	status, err := nd.RefreshNodeletStatus()
	zap.S().Infof("Done spawning worker: %s\n", nd.nodeletCfg.HostId)

	SetClusterNodeStatus(nd.clusterStatus, nd.nodeletCfg.HostId, status, err)
}

func (nd *NodeletDeployer) DeployNodelet() error {
	if err := nd.CreatePf9User(); err != nil {
		return err
	}
	if err := nd.UploadCerts(); err != nil {
		return err
	}
	if err := nd.CopyNodeletConfig(); err != nil {
		return err
	}
	if err := nd.InstallNodelet(); err != nil {
		return err
	}
	// TODO: Remove this, move to nodelet afterinstall.sh
	if err := nd.SetPf9Ownerships(); err != nil {
		return err
	}
	if err := nd.StartNodelet(); err != nil {
		return err
	}
	return nil
}

func (nd *NodeletDeployer) ReconfigureNodelet() error {
	if err := nd.CopyNodeletConfig(); err != nil {
		return err
	}
	if err := nd.RestartNodelet(); err != nil {
		return err
	}
	return nil
}

func (nd *NodeletDeployer) UploadCerts() error {
	srcCertPath := filepath.Join(ClusterStateDir, nd.nodeletCfg.ClusterId, "certs", RootCACRT)
	err := UploadFileWrapper(srcCertPath, RootCACRT, RemoteCertsDir, nd.client)
	if err != nil {
		return err
	}

	srcKeyPath := filepath.Join(ClusterStateDir, nd.nodeletCfg.ClusterId, "certs", RootCAKey)
	err = UploadFileWrapper(srcKeyPath, RootCAKey, RemoteCertsDir, nd.client)
	if err != nil {
		return err
	}
	return nil
}

func (nd *NodeletDeployer) CopyNodeletConfig() error {
	err := UploadFileWrapper(nd.nodeletSrcFile, NodeletConfigFile, NodeletConfigDir, nd.client)
	if err != nil {
		return err
	}
	return nil
}

func (nd *NodeletDeployer) SetOsType() {
	// TODO: Find actual OS type of remote node once we add Ubuntu support
	nd.OsType = OsTypeCentos
}

func (nd *NodeletDeployer) CreatePf9User() error {
	var cmdList []string

	cmdList = append(cmdList, "mkdir -p /opt/pf9/home")
	cmdList = append(cmdList, "groupadd -f pf9group")
	cmdList = append(cmdList, "id -u pf9 &>/dev/null || useradd -d /opt/pf9/home -G pf9group pf9")

	for _, cmd := range cmdList {
		if _, _, err := nd.client.RunCommand(cmd); err != nil {
			return fmt.Errorf("CreatePf9User: %s: %s", cmd, err)
		}
	}
	return nil
}

func (nd *NodeletDeployer) InstallNodelet() error {
	if err := nd.client.UploadFile(nd.pf9KubeTarSrc, NodeletTarDst, 0644, nil); err != nil {
		return fmt.Errorf("Failed to copy pf9-kube(nodelet) RPM: %s", err)
	}

	unTarCmd := "tar -C /tmp/ -xzvf " + NodeletTarDst
	if _, _, err := nd.client.RunCommand(unTarCmd); err != nil {
		return fmt.Errorf("Failed: %s: %s", unTarCmd, err)
	}

	var installCmd string
	if nd.OsType == OsTypeCentos {
		installCmd = "yum install -y " + filepath.Join("/tmp/", NodeletRpmName)
	} else if nd.OsType == OsTypeUbuntu {
		installCmd = "apt-get install -y " + filepath.Join("/tmp/", NodeletDebName)
	} else {
		return fmt.Errorf("OS type not supported")
	}

	if _, _, err := nd.client.RunCommand(installCmd); err != nil {
		return fmt.Errorf("Failed: %s: %s", installCmd, err)
	}

	return nil
}

/* This is a temporary workaround, normally done by hostagent
   TODO: Add to the nodelet after-install.sh script
*/
func (nd *NodeletDeployer) SetPf9Ownerships() error {
	chownCmd := fmt.Sprintf("chown %s:pf9group %s ", NodeletUser, "/var/log/pf9")
	if _, _, err := nd.client.RunCommand(chownCmd); err != nil {
		return fmt.Errorf("Failed: %s: %s", chownCmd, err)
	}

	chownCmd = fmt.Sprintf("chown %s:pf9group %s ", NodeletUser, "/var/opt/pf9")
	if _, _, err := nd.client.RunCommand(chownCmd); err != nil {
		return fmt.Errorf("Failed: %s: %s", chownCmd, err)
	}

	chownCmd = fmt.Sprintf("chown %s:pf9group %s ", NodeletUser, "/etc/pf9")
	if _, _, err := nd.client.RunCommand(chownCmd); err != nil {
		return fmt.Errorf("Failed: %s: %s", chownCmd, err)
	}

	chownCmd = fmt.Sprintf("chown %s:pf9group %s ", NodeletUser, "/opt/pf9")
	if _, _, err := nd.client.RunCommand(chownCmd); err != nil {
		return fmt.Errorf("Failed: %s: %s", chownCmd, err)
	}
	return nil
}

func (nd *NodeletDeployer) StartNodelet() error {
	startCmd := "systemctl start pf9-nodeletd"
	if _, _, err := nd.client.RunCommand(startCmd); err != nil {
		return fmt.Errorf("Failed: %s: %s", startCmd, err)
	}
	return nil
}

func (nd *NodeletDeployer) RestartNodelet() error {
	startCmd := "systemctl restart pf9-nodeletd"
	if _, _, err := nd.client.RunCommand(startCmd); err != nil {
		return fmt.Errorf("Failed: %s: %s", startCmd, err)
	}
	return nil
}

func (nd *NodeletDeployer) RefreshNodeletStatus() (string, error) {
	cpCmd := fmt.Sprintf("cp %s /tmp/", KubeStatusFile)
	if _, _, err := nd.client.RunCommand(cpCmd); err != nil {
		return "", fmt.Errorf("RefreshNodeletStatus failed: %s", err)
	}

	clusterStateDir := filepath.Join(ClusterStateDir, nd.nodeletCfg.ClusterId)
	nodeStateDir := filepath.Join(clusterStateDir, nd.nodeletCfg.HostId)
	nodeStatusFile := filepath.Join(nodeStateDir, "kube_status.json")

	err := nd.client.DownloadFile("/var/opt/pf9/kube_status", nodeStatusFile, 0755, nil)
	if err != nil {
		return "", fmt.Errorf("RefreshNodeletStatus failed: %s", err)
	}

	nodeletData := new(nodeletext.ExtensionData)

	extFile, err := ioutil.ReadFile(nodeStatusFile)
	if err != nil {
		return "", fmt.Errorf("RefreshNodeletStatus: Failed to read %s: %s", nodeStatusFile, err)
	}
	decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(extFile), 4096)
	if err := decoder.Decode(nodeletData); err != nil {
		return "", fmt.Errorf("RefreshNodeletStatus: Failed to decode: %s", err)
	}

	return nodeletData.NodeState, nil
}

/* Copying files follows a predictable pattern
   1. Make the remote directory, if it doesn't exist
   2. Set permissions to pf9:pf9group
   3. Upload to /tmp/ because ssh user may be different, and SFTP client
      used by pf9ctl.UploadFile only allows access to home directory and /tmp
   4. Finally, move to target directory
*/
func UploadFileWrapper(srcFilePath, fileName, dstDir string, client ssh.Client) error {
	mkdirCmd := "mkdir -p " + dstDir
	if _, _, err := client.RunCommand(mkdirCmd); err != nil {
		return fmt.Errorf("Failed: %s: %s", mkdirCmd, err)
	}

	chownCmd := fmt.Sprintf("chown %s %s ", NodeletUser, dstDir)
	if _, _, err := client.RunCommand(chownCmd); err != nil {
		return fmt.Errorf("Failed: %s: %s", chownCmd, err)
	}

	tmpDst := filepath.Join("/tmp/", fileName)
	err := client.UploadFile(srcFilePath, tmpDst, 0644, nil)
	if err != nil {
		return fmt.Errorf("Failed to upload: %s", err)
	}

	dstFilePath := filepath.Join(dstDir, fileName)
	mvCmd := fmt.Sprintf("mv %s %s", tmpDst, dstFilePath)
	if _, _, err := client.RunCommand(mvCmd); err != nil {
		return fmt.Errorf("Failed to mv %s to %s", tmpDst, dstFilePath)
	}
	return nil
}

func (nd *NodeletDeployer) DeleteNodelet() error {
	eraseCmd := "yum erase -y pf9-kube"
	zap.S().Infof("Removing nodelet with cmd: %s", eraseCmd)
	if _, _, err := nd.client.RunCommand(eraseCmd); err != nil {
		return fmt.Errorf("Failed: %s: %s", eraseCmd, err)
	}
	cleanupCmd := "rm -rf /etc/pf9 && rm -rf /var/opt/pf9 && rm -rf /var/log/pf9 && rm -rf /var/spool/mail/pf9 && rm -rf /opt/cni/bin/* && rm -rf /etc/cni"
	zap.S().Infof("Cleaning up pf9 and nodelet directories:\n%s", cleanupCmd)
	if _, _, err := nd.client.RunCommand(cleanupCmd); err != nil {
		return fmt.Errorf("Failed: %s: %s", cleanupCmd, err)
	}
	return nil
}

/*
 * CreateSSHClientRaw: connects to given host on port with given user and key
 */
func CreateSSHClientRaw(host string, user string, privateKey []byte, port int) (ssh.Client, error) {
	if isv6 := IsIpV6Addr(host); isv6 {
		host = EncloseIpV6(host)
	}
	client, err := ssh.NewClient(host, port, user, privateKey, "")
	return client, err
}

/*
 * CreateSSHClient: connects to given host on port with given user and key.
 * The function retries the operation total 5 times on failure attempt after waiting for 60 seconds.
 */
func CreateSSHClient(host string, user string, privateKey []byte, port int) (ssh.Client, error) {
	client, err := CreateSSHClientRaw(host, user, privateKey, port)
	retryCount := 5
	for retryCount > 0 {
		if err != nil {
			time.Sleep(60 * time.Second)
			client, err = CreateSSHClientRaw(host, user, privateKey, port)
			retryCount--
			continue
		}
		zap.S().Debugf("connection successful")
		break
	}
	return client, err
}

func IsIpV6Addr(host string) bool {
	ip := net.ParseIP(host)
	if ip == nil {
		//Not a Valid V4 or V6 address, so likely a hostname
		return false
	}

	if ip.To4() == nil {
		return true
	}
	return false
}

func EncloseIpV6(ip string) string {
	return "[" + ip + "]"
}
