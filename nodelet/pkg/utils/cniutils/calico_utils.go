package cniutils

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/platform9/nodelet/nodelet/pkg/utils/command"
	"github.com/platform9/nodelet/nodelet/pkg/utils/config"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	"github.com/subosito/gotenv"
	"go.uber.org/zap"
)

type CalicoImpl struct{}

type CalicoUtilsInterface interface {
	network_running(config.Config) error
	ensure_network_running(cfg config.Config) error
	write_cni_config_file() error
	ensure_network_config_up_to_date() error
	ensure_network_controller_destroyed() error
}

func New() CalicoUtilsInterface {
	return &CalicoImpl{}
}

func (n *CalicoImpl) network_running(cfg config.Config) error {
	// TODO: Check status of the local pod/app
	// See https://platform9.atlassian.net/browse/PMK-871
	// Work-around: always return desired state until we have a better algorithm.
	// When ROLE==none, report non-running status to make status_none.sh happy.
	if cfg.ClusterRole == "none" {
		zap.S().Warnf("Cluster role is not assigned.")
		return nil
	}

	return nil
}

func (n *CalicoImpl) ensure_network_running(cfg config.Config) error {
	// Bridge for containers is created by CNI. So if docker has created a
	// bridge, in the past, delete it
	// See https://platform9.atlassian.net/browse/IAAS-7740 for more information
	if cfg.Pf9ManagedDocker != false {
		delete_docker0_bridge_if_present()
	}

	if cfg.ClusterRole == "master" {
		deploy_calico_daemonset()
	}

	return nil
}

func (n *CalicoImpl) write_cni_config_file() error {
	return nil
}

func (n *CalicoImpl) ensure_network_config_up_to_date() error {
	return nil
}

func (n *CalicoImpl) ensure_network_controller_destroyed() error {
	remove_cni_config_file()
	remove_ipip_tunnel_iface()
	return nil
}

// Plugin specific methods
func deploy_calico_daemonset() error {
	calico_app := "/opt/pf9/pf9-kube/conf/networkapps/calico-${KUBERNETES_VERSION}.yaml"

	input, err := os.ReadFile(calico_app)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	err = gotenv.Load(constants.KubeEnvPath)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	mtu_size := os.Getenv("MTU_SIZE")
	CALICO_IPV4POOL_CIDR := os.Getenv("CONTAINERS_CIDR")
	IPV4_ENABLED := "true"

	IPV6_ENABLED := os.Getenv("IPV6_ENABLED")

	if IPV6_ENABLED == "true" {
		CALICO_IPV4POOL_CIDR = ""
		IPV4_ENABLED = "false"
	}

	output := strings.ReplaceAll(string(input), "__CALICO_IPV4POOL_CIDR__", CALICO_IPV4POOL_CIDR)
	output = strings.ReplaceAll(string(input), "|__PF9_ETCD_ENDPOINTS__", "https://"+os.Getenv("MASTER_IP")+":4001")
	output = strings.ReplaceAll(string(input), "__MTU_SIZE__", mtu_size)
	output = strings.ReplaceAll(string(input), "__CALICO_IPV4_BLOCK_SIZE__", os.Getenv("CALICO_IPV4_BLOCK_SIZE"))
	output = strings.ReplaceAll(string(input), "__CALICO_IPIP_MODE__", os.Getenv("CALICO_IPIP_MODE"))
	output = strings.ReplaceAll(string(input), "__CALICO_NAT_OUTGOING__", os.Getenv("CALICO_NAT_OUTGOING"))
	output = strings.ReplaceAll(string(input), "__CALICO_IPV4__", os.Getenv("CALICO_IPV4"))
	output = strings.ReplaceAll(string(input), "__CALICO_IPV6__", os.Getenv("CALICO_IPV6"))
	output = strings.ReplaceAll(string(input), "__CALICO_IPV4_DETECTION_METHOD__", os.Getenv("CALICO_IPV4_DETECTION_METHOD"))
	output = strings.ReplaceAll(string(input), "__CALICO_IPV6_DETECTION_METHOD__", os.Getenv("CALICO_IPV6_DETECTION_METHOD"))
	output = strings.ReplaceAll(string(input), "__CALICO_ROUTER_ID__", os.Getenv("CALICO_ROUTER_ID"))
	output = strings.ReplaceAll(string(input), "__CALICO_IPV6POOL_CIDR__", os.Getenv("CALICO_IPV6POOL_CIDR"))
	output = strings.ReplaceAll(string(input), "__CALICO_IPV6POOL_BLOCK_SIZE__", os.Getenv("CALICO_IPV6POOL_BLOCK_SIZE"))
	output = strings.ReplaceAll(string(input), "__CALICO_IPV6POOL_NAT_OUTGOING__", os.Getenv("CALICO_IPV6POOL_NAT_OUTGOING"))
	output = strings.ReplaceAll(string(input), "__FELIX_IPV6SUPPORT__", os.Getenv("FELIX_IPV6SUPPORT"))
	output = strings.ReplaceAll(string(input), "__IPV6_ENABLED__", os.Getenv("IPV6_ENABLED"))
	output = strings.ReplaceAll(string(input), "__IPV4_ENABLED__", IPV4_ENABLED)

	if err = os.WriteFile("/opt/pf9/pf9-kube/conf/networkapps/calico-configured.yaml", []byte(output), 0666); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	// Apply daemon set yaml
	//${KUBECTL_SYSTEM} apply -f ${calico_app_configured}
	cmd := command.New()
	_, err = cmd.RunCommand(context.Background(), nil, 0, "", "KUBECTL apply -f ", calico_app)
	if err != nil {
		zap.S().Warnf("Error running command: %v", cmd)
		return err
	}
	return nil
}

func remove_cni_config_file() {
	//rm -f ${CNI_CONFIG_DIR}/10-calico* || echo "Either file not present or unable to delete. Continuing"
	files, err := ioutil.ReadDir("/etc/cni/net.d")
	if err != nil {
		fmt.Println(err)
	}
	for _, file := range files {
		match, _ := regexp.MatchString("10-calico", file.Name())

		if match == true {
			filepath, err := filepath.Abs(file.Name())
			if err != nil {
				fmt.Println(err)
			}
			err = os.Remove(filepath)
			if err != nil {
				fmt.Println(err)
			} else {
				fmt.Println(file.Name() + "Deleted")
			}

		}
	}
}

func delete_docker0_bridge_if_present() error {
	// Opportunistically delete docker0 bridge
	cmd := command.New()
	_, err := cmd.RunCommand(context.Background(), nil, 0, "", "ip", "link", "set", "dev", "docker0", "down", "||", "true")
	if err != nil {
		zap.S().Warnf("Error running command: %v", cmd)
		return err
	}
	_, err = cmd.RunCommand(context.Background(), nil, 0, "", "ip", "link", "del", "docker0", "||", "true")
	if err != nil {
		zap.S().Warnf("Error running command: %v", cmd)
		return err
	}
	return nil
}

func remove_ipip_tunnel_iface() error {
	cmd := command.New()
	_, err := cmd.RunCommand(context.Background(), nil, 0, "", "ip", "link", "set", "dev", "tunl0", "down", "||", "true")
	if err != nil {
		zap.S().Warnf("Error running command: %v", cmd)
		return err
	}
	_, err = cmd.RunCommand(context.Background(), nil, 0, "", "ip", "link", "del", "tunl0", "||", "true")
	if err != nil {
		zap.S().Warnf("Error running command: %v", cmd)
		return err
	}
	return nil
	//ip link set dev tunl0 down || true
	//ip link del tunl0 || true
}

func local_apiserver_running(cfg config.Config) {

	err := net.Listen("tcp", ":"+cfg.K8sApiPort)
	if err != nil {
		fmt.Println("Connecting error:", err)
	}

}

func ensure_role_binding() error {
	cmd := command.New()
	_, err := cmd.RunCommand(context.Background(), nil, 0, "", "KUBECTL version")
	if err != nil {
		zap.S().Warnf("Error running command: %v", cmd)
		return err
	}
	role_binding := "/etc/pf9/kube.d/rolebindings/"

	_, err = cmd.RunCommand(context.Background(), nil, 0, "", "KUBECTL apply --force -f ", role_binding)
	if err != nil {
		zap.S().Warnf("Error running command: %v", cmd)
		return err
	}
	return nil
}
