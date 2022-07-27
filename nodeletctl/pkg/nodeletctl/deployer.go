package nodeletctl

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	nodeletext "github.com/platform9/nodelet/nodelet/pkg/utils/extensionfile"
	"github.com/platform9/pf9ctl/pkg/ssh"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/util/yaml"
)

/* Everything in deployer is a wrapper around shell commands. This is because it executes on
 * remote machines via an SSH client, so we cannot use native Golang packages.
 * Add any commands to exec here to onboard the node or install nodelet
 */

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

func GetNodeletDeployer(cfg *BootstrapConfig, clusterStatus *ClusterStatus, nodeletCfg *NodeletConfig, nodeName, nodeletSrcFile string) (*NodeletDeployer, error) {
	local, err := isLocal(nodeName)
	if err != nil {
		return nil, err
	}
	var sshClient ssh.Client
	if local {
		sshClient = getLocalClient()
	} else {
		sshKey, err := ioutil.ReadFile(cfg.SSHPrivateKeyFile)
		if err != nil {
			return nil, fmt.Errorf("Failed to read private key: %s", cfg.SSHPrivateKeyFile)
		}
		sshClient, err = CreateSSHClient(nodeName, cfg.SSHUser, sshKey, 22)
		if err != nil {
			return nil, fmt.Errorf("can't create ssh client to host %s, %s", nodeName, err)
		}
	}

	deployer := NewNodeletDeployer(cfg, sshClient, nodeletSrcFile, nodeletCfg, clusterStatus)
	return deployer, nil
}

func isLocal(nodeName string) (bool, error) {
	name, err := os.Hostname()
	if err != nil {
		return false, fmt.Errorf("failed to get hostname: %s", err)
	}
	if name == nodeName {
		return true, nil
	}
	ifaces, err := net.Interfaces()
	if err != nil {
		return false, fmt.Errorf("Can't check local interfaces %v", err)
	}
	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			return false, fmt.Errorf("Can't get addresses for interface %s: %v", iface.Name, err)
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil {
				continue
			}
			if nodeName == ip.String() {
				return true, nil
			}
		}
	}

	return false, nil
}

func (nd *NodeletDeployer) UpgradeMaster() error {

	// Uninstall pf9-kube
	if err := nd.UninstallNodelet("nodelet"); err != nil {
		return fmt.Errorf("Failed to uninstall nodelet: %s", err)
	}

	if err := nd.CopyNodeletConfig(); err != nil {
		return fmt.Errorf("failed to copy nodelet config: %s", err)
	}

	// Install new pf9-kube
	if err := nd.InstallNodelet(); err != nil {
		return fmt.Errorf("Failed to install nodelet: %s", err)
	}

	// Restart nodeletd
	if err := nd.RestartNodelet(); err != nil {
		return fmt.Errorf("Failed to restart nodelet: %s", err)
	}

	return nil
}

func (nd *NodeletDeployer) UpgradeWorker(wg *sync.WaitGroup) {
	defer wg.Done()
	// Uninstall pf9-kube
	if err := nd.UninstallNodelet("nodelet"); err != nil {
		err = fmt.Errorf("Failed to uninstall nodelet: %s", err)
		SetClusterNodeStatus(nd.clusterStatus, nd.nodeletCfg.HostId, "failed", err)
		return
	}

	if err := nd.CopyNodeletConfig(); err != nil {
		err = fmt.Errorf("failed to copy nodelet config: %s", err)
		SetClusterNodeStatus(nd.clusterStatus, nd.nodeletCfg.HostId, "failed", err)
		return
	}

	// Install new pf9-kube
	if err := nd.InstallNodelet(); err != nil {
		err = fmt.Errorf("Failed to install nodelet: %s", err)
		SetClusterNodeStatus(nd.clusterStatus, nd.nodeletCfg.HostId, "failed", err)
		return
	}

	// Restart nodeletd
	if err := nd.RestartNodelet(); err != nil {
		err = fmt.Errorf("Failed to restart nodelet: %s", err)
		SetClusterNodeStatus(nd.clusterStatus, nd.nodeletCfg.HostId, "failed", err)
		return
	}

}

func (nd *NodeletDeployer) SpawnMaster(numMaster int) (string, error) {
	zap.S().Infof("Spawning master: %s", nd.nodeletCfg.HostId)
	// Add any master-specific tasks here before and after
	if err := nd.DeployNodelet(); err != nil {
		zap.S().Errorf("failed to deploy nodelet: %s", err)
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
	zap.S().Infof("Spawning worker: %s", nd.nodeletCfg.HostId)
	if err := nd.DeployNodelet(); err != nil {
		zap.S().Errorf("failed to deploy worker nodelet: %s", err)
		SetClusterNodeStatus(nd.clusterStatus, nd.nodeletCfg.HostId, "failed", err)
		return
	}
	// Add any worker-specific tasks here, if needed
	status, err := nd.RefreshNodeletStatus()
	zap.S().Infof("Done spawning worker: %s\n", nd.nodeletCfg.HostId)

	SetClusterNodeStatus(nd.clusterStatus, nd.nodeletCfg.HostId, status, err)
}

func (nd *NodeletDeployer) DeployNodelet() error {
	zap.S().Infof("Deploying nodelet: %s", nd.nodeletCfg.HostId)
	if err := nd.CreatePf9User(); err != nil {
		return fmt.Errorf("failed to create user: %s", err)
	}
	if err := nd.UploadCerts(); err != nil {
		return fmt.Errorf("failed to upload certs: %s", err)
	}
	if err := nd.UploadUserImages(); err != nil {
		return fmt.Errorf("failed to upload user container images: %s", err)
	}
	if err := nd.CopyNodeletConfig(); err != nil {
		return fmt.Errorf("failed to copy nodelet config: %s", err)
	}
	if err := nd.InstallNodelet(); err != nil {
		return fmt.Errorf("failed to install nodelet: %s", err)
	}
	// TODO: Remove this, move to nodelet afterinstall.sh
	if err := nd.SetPf9Ownerships(); err != nil {
		return fmt.Errorf("failed to set pf9 ownerships: %s", err)
	}
	if err := nd.StartNodelet(); err != nil {
		return fmt.Errorf("failed to start nodelet: %s", err)
	}
	return nil
}

func (nd *NodeletDeployer) ReconfigureNodelet() error {
	zap.S().Infof("Reconfiguring nodelet: %s", nd.nodeletCfg.HostId)
	if err := nd.CopyNodeletConfig(); err != nil {
		return fmt.Errorf("failed to copy nodelet config: %s", err)
	}
	if err := nd.RestartNodelet(); err != nil {
		return fmt.Errorf("failed to restart nodelet: %s", err)
	}
	return nil
}

func (nd *NodeletDeployer) UploadCerts() error {
	zap.S().Infof("Uploading certs to nodelet: %s", nd.nodeletCfg.HostId)
	srcCertPath := filepath.Join(ClusterStateDir, nd.nodeletCfg.ClusterId, "certs", RootCACRT)
	err := UploadFileWrapper(srcCertPath, RootCACRT, RemoteCertsDir, nd.client)
	if err != nil {
		return fmt.Errorf("failed to upload certs: %s", err)
	}

	srcKeyPath := filepath.Join(ClusterStateDir, nd.nodeletCfg.ClusterId, "certs", RootCAKey)
	err = UploadFileWrapper(srcKeyPath, RootCAKey, RemoteCertsDir, nd.client)
	if err != nil {
		return fmt.Errorf("failed to upload CA certs: %s", err)
	}
	return nil
}

func (nd *NodeletDeployer) CopyNodeletConfig() error {
	zap.S().Infof("Copying nodelet config to node: %s", nd.nodeletCfg.HostId)
	err := UploadFileWrapper(nd.nodeletSrcFile, NodeletConfigFile, NodeletConfigDir, nd.client)
	if err != nil {
		return fmt.Errorf("failed to copy nodelet config: %s", err)
	}
	return nil
}

func (nd *NodeletDeployer) SetOsType() {
	// TODO: Find actual OS type of remote node once we add Ubuntu support
	nd.OsType = OsTypeCentos

	stdout, _, err := nd.client.RunCommand("cat /etc/os-release")
	if err != nil {
		zap.S().Infof("failed reading data from file: %s", err)
		return
	}
	osString := strings.ToLower(string(stdout))
	if strings.Contains(osString, "ubuntu") {
		nd.OsType = OsTypeUbuntu
		zap.S().Infof("OS type is Ubuntu")
	} else {
		zap.S().Infof("assuming OS type is CentOS, the os string is %s", osString)
	}

}

func (nd *NodeletDeployer) CreatePf9User() error {
	zap.S().Infof("Checking pf9 user on node: %s", nd.nodeletCfg.HostId)
	_, _, err := nd.client.RunCommand("id -u pf9")
	if err != nil {
		zap.S().Infof("User name doesn't exist, proceeding to create")
	} else {
		zap.S().Infof("User name already exist")
		return nil
	}
	zap.S().Infof("Creating pf9 user")

	_, _, err = nd.client.RunCommand("mkdir -p /opt/pf9/home")
	if err != nil {
		return fmt.Errorf("failed to create home dir: %s", err)
	}

	_, _, err = nd.client.RunCommand("groupadd -f pf9group")
	if err != nil {
		return fmt.Errorf("failed to add pf9group: %s", err)
	}

	_, _, err = nd.client.RunCommand("id -u pf9")
	if err != nil {
		zap.S().Infof("check for id pf9: %s", err)
		_, _, err = nd.client.RunCommand("useradd -d /opt/pf9/home -G pf9group pf9")
		if err != nil {
			return fmt.Errorf("createPf9User: %s", err)
		}
	}
	return nil
}

func (nd *NodeletDeployer) DetermineNodeletPkgName(nodeletPkgsDir string) (string, error) {
	lsNodeletCmd := "ls " + nodeletPkgsDir
	stdOut, stdErr, err := nd.client.RunCommand(lsNodeletCmd)
	if err != nil {
		return "", fmt.Errorf("Failed to run : %s: %s: %s", lsNodeletCmd, err, stdErr)
	}
	nodeletPkgNames := strings.Split(string(stdOut), "\n")
	nodeletPkgName := ""
	for _, pkgName := range nodeletPkgNames {
		pkgName = strings.TrimSpace(pkgName)
		zap.S().Infof("Matching pkg: %s", pkgName)
		if nd.OsType == OsTypeCentos && strings.HasSuffix(pkgName, ".rpm") {
			nodeletPkgName = pkgName
			zap.S().Infof("Match found for pkg: %s and os: %s", pkgName, nd.OsType)
			break
		} else if nd.OsType == OsTypeUbuntu && strings.HasSuffix(pkgName, ".deb") {
			nodeletPkgName = pkgName
			zap.S().Infof("Match found for pkg: %s and os: %s", pkgName, nd.OsType)
			break
		}
	}

	if nodeletPkgName == "" {
		return "", fmt.Errorf("Nodelet package for the OS:  %s, not found", nd.OsType)
	}
	return nodeletPkgName, nil
}

func (nd *NodeletDeployer) UninstallNodelet(nodeletPkgName string) error {
	zap.S().Infof("Uninstalling nodelet with pkg name: %s", nodeletPkgName)
	uninstallCmd := ""
	if nd.OsType == OsTypeCentos {
		uninstallCmd = "yum erase -y " + nodeletPkgName
	} else if nd.OsType == OsTypeUbuntu {
		uninstallCmd = "apt-get uninstall -y " + nodeletPkgName
	} else {
		return fmt.Errorf("OS type not supported")
	}

	if _, _, err := nd.client.RunCommand(uninstallCmd); err != nil {
		return fmt.Errorf("Failed: %s: %s", uninstallCmd, err)
	}

	return nil
}

func (nd *NodeletDeployer) InstallNodelet() error {
	if err := nd.client.UploadFile(nd.pf9KubeTarSrc, NodeletTarDst, 0644, nil); err != nil {
		return fmt.Errorf("Failed to copy nodelet RPM: %s", err)
	}

	clearTmpCmd := "rm -Rf " + NodeletPkgsTmpDir
	if _, stdErr, err := nd.client.RunCommand(clearTmpCmd); err != nil {
		return fmt.Errorf("Failed: %s: %s: %s", clearTmpCmd, err, stdErr)
	}

	createTmpCmd := "mkdir -p " + NodeletPkgsTmpDir
	if _, stdErr, err := nd.client.RunCommand(createTmpCmd); err != nil {
		return fmt.Errorf("Failed: %s: %s: %s", createTmpCmd, err, stdErr)
	}

	unTarCmd := "tar -C " + NodeletPkgsTmpDir + " -xzvf " + NodeletTarDst
	if _, stdErr, err := nd.client.RunCommand(unTarCmd); err != nil {
		return fmt.Errorf("Failed: %s: %s: %s", unTarCmd, err, stdErr)
	}

	nodeletPkgName, err := nd.DetermineNodeletPkgName(NodeletPkgsTmpDir)
	if err != nil {
		return fmt.Errorf("Failed to DetermineNodeletPkgName: %s", err)
	}

	nodeletPkgPath := NodeletPkgsTmpDir + nodeletPkgName
	var installCmd string
	if nd.OsType == OsTypeCentos {
		installCmd = "yum install -y " + nodeletPkgPath
	} else if nd.OsType == OsTypeUbuntu {
		installCmd = "apt-get install -y " + nodeletPkgPath
	} else {
		return fmt.Errorf("OS type not supported")
	}

	if stdOut, stdErr, err := nd.client.RunCommand(installCmd); err != nil {
		return fmt.Errorf("Failed to run %s: %s: %s: %s", installCmd, err, stdErr, stdOut)
	}

	return nil
}

/* This is a temporary workaround, normally done by hostagent
   TODO: Add to the nodelet after-install.sh script
*/
func (nd *NodeletDeployer) SetPf9Ownerships() error {
	zap.S().Infof("Setting pf9 ownership")
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
	zap.S().Infof("Starting nodelet")
	startCmd := "systemctl start pf9-nodeletd"
	if _, _, err := nd.client.RunCommand(startCmd); err != nil {
		return fmt.Errorf("Failed: %s: %s", startCmd, err)
	}
	return nil
}

func (nd *NodeletDeployer) RestartNodelet() error {
	zap.S().Infof("Restarting nodelet")
	startCmd := "systemctl restart pf9-nodeletd"
	if _, _, err := nd.client.RunCommand(startCmd); err != nil {
		return fmt.Errorf("Failed: %s: %s", startCmd, err)
	}
	return nil
}

func (nd *NodeletDeployer) NodeletStackRestart() error {
	var cmdList []string
	cmdList = append(cmdList, "systemctl stop pf9-nodeletd")
	cmdList = append(cmdList, "/opt/pf9/nodelet/nodeletd phases stop --force |& tee /tmp/nodeletStop.log")

	for _, cmd := range cmdList {
		if _, _, err := nd.client.RunCommand(cmd); err != nil {
			return fmt.Errorf("NodeletStackRestart cmd failed: %s: %s", cmd, err)
		}
	}

	return nd.RestartNodelet()
}

func (nd *NodeletDeployer) RefreshNodeletStatus() (string, error) {
	zap.S().Infof("Refreshing nodelet status")
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
	zap.S().Infof("Uploading %s to %s", srcFilePath, dstDir)
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
	zap.S().Infof("Deleting nodelet")
	var eraseCmd string
	if nd.OsType == OsTypeCentos {
		eraseCmd = "yum erase -y nodelet"
	} else if nd.OsType == OsTypeUbuntu {
		eraseCmd = "apt remove -y nodelet"
	} else {
		return fmt.Errorf("OS type not supported")
	}

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

func (nd *NodeletDeployer) DeleteOldCerts() error {
	deleteCmd := "rm -rf /etc/pf9/kube.d/certs"
	if _, _, err := nd.client.RunCommand(deleteCmd); err != nil {
		return fmt.Errorf("Failed to remove old certs: %s", err)
	}
	return nil
}

func (nd *NodeletDeployer) DeleteCniDir() error {
	deleteCmd := "rm -rf /etc/cni/net.d/"
	if _, _, err := nd.client.RunCommand(deleteCmd); err != nil {
		return fmt.Errorf("Failed to remove old certs: %s", err)
	}
	return nil
}

func (nd *NodeletDeployer) UploadCertsAndRestartStack(wg *sync.WaitGroup) error {
	defer wg.Done()

	if err := nd.UploadCerts(); err != nil {
		return fmt.Errorf("failed to upload new CA certs: %s", err)
	}
	if err := nd.DeleteOldCerts(); err != nil {
		return fmt.Errorf("failed to clear old client certs: %s", err)
	}
	if err := nd.DeleteCniDir(); err != nil {
		return fmt.Errorf("failed to clear old client certs: %s", err)
	}
	if err := nd.NodeletStackRestart(); err != nil {
		return fmt.Errorf("failed to stop/start nodelet stack: %s", err)
	}

	return nil
}

func (nd *NodeletDeployer) UploadUserImages() error {
	if nd.nodeletCfg.UserImages == nil {
		zap.S().Infof("No offline container images specified, skipping upload...")
		return nil
	}

	for _, imgTarSrc := range nd.nodeletCfg.UserImages {
		zap.S().Infof("Uploading user images: %s", imgTarSrc)
		if _, err := os.Stat(imgTarSrc); os.IsNotExist(err) {
			zap.S().Errorf("User Images file does not exist: %s", err)
			return fmt.Errorf("User Images file does not exist: %s", err)
		}

		filename := filepath.Base(imgTarSrc)
		err := UploadFileWrapper(imgTarSrc, filename, UserImagesDir, nd.client)
		if err != nil {
			return fmt.Errorf("Failed to upload user images: %s", err)
		}
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
