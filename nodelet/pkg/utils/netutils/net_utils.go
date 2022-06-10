package netutils

import (
	"fmt"
	"net"
	"os"

	"github.com/apparentlymart/go-cidr/cidr"
	"github.com/docker/libcontainer/netlink"
	"github.com/pkg/errors"
	"github.com/platform9/nodelet/nodelet/pkg/utils/config"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	"golang.org/x/net/nettest"
)

type NetImpl struct{}

type NetInterface interface {
	AddrConv(string, int) (string, error)
	IpForHttp(string) (string, error)
	GetRoutedNetworkInterFace() (string, error)
	GetIPv4ForInterfaceName(string) (string, error)
	GetNodeIP() (string, error)
	GetNodeIdentifier(config.Config) (string, error)
	SetUpVeth(string, string, string) error
	TearDownVeth(string) error
}

func New() NetInterface {
	return &NetImpl{}
}

//AddrConv generates <pos> th IP from hostCIDR
func (n *NetImpl) AddrConv(hostCIDR string, pos int) (string, error) {

	_, ipnet, errCidr := net.ParseCIDR(hostCIDR)
	if errCidr != nil {
		return "", errCidr
	}
	ip, errPos := cidr.Host(ipnet, pos)
	if errPos != nil {
		return "", errPos
	}
	return ip.String(), nil
}

// IpForHttp returns formatted Ip
// If IP is IPv4 returns as it is, If IP is IPv6 returns IP with square bracks
func (n *NetImpl) IpForHttp(masterIp string) (string, error) {

	if net.ParseIP(masterIp).To4() != nil {
		return masterIp, nil
	} else if net.ParseIP(masterIp).To16() != nil {
		return "[" + masterIp + "]", nil
	}
	return "", fmt.Errorf("invalid IP")
}

// GetNodeIdentifier returns node identifier as Hostname / Node IP
func (n *NetImpl) GetNodeIdentifier(cfg config.Config) (string, error) {

	var err error
	var nodeIdentifier string
	if cfg.CloudProviderType == constants.LocalCloudProvider && cfg.UseHostname == constants.TrueString {
		nodeIdentifier, err = os.Hostname()
		if err != nil {
			return nodeIdentifier, errors.Wrap(err, "failed to get hostName for node identification")
		}
		if nodeIdentifier == "" {
			return nodeIdentifier, fmt.Errorf("nodeIdentifier is null")
		}
	} else {
		nodeIdentifier, err = n.GetNodeIP()
		if err != nil {
			return nodeIdentifier, errors.Wrap(err, "failed to get node IP address for node identification")
		}
		if nodeIdentifier == "" {
			return nodeIdentifier, fmt.Errorf("nodeIdentifier is null")
		}
	}
	return nodeIdentifier, nil
}

// GetNodeIP returns routed network interface IP
func (n *NetImpl) GetNodeIP() (string, error) {
	var err error
	routedInterfaceName, err := n.GetRoutedNetworkInterFace()
	if err != nil {
		return "", errors.Wrapf(err, "failed to get routed network interface")
	}
	routedIp, err := n.GetIPv4ForInterfaceName(routedInterfaceName)
	if err != nil {
		return "", errors.Wrap(err, "failed to get node IP")
	}
	return routedIp, nil
}

// GetRoutedNetworkInterFace returns roted network interface
func (n *NetImpl) GetRoutedNetworkInterFace() (string, error) {
	routedInterface, err := nettest.RoutedInterface("ip", net.FlagUp|net.FlagBroadcast)
	if err != nil {
		return "", err
	}
	routedInterfaceName := routedInterface.Name
	return routedInterfaceName, nil
}

// GetIPv4ForInterfaceName returns IPv4 for given interface name
func (n *NetImpl) GetIPv4ForInterfaceName(interfaceName string) (string, error) {
	interfaces, _ := net.Interfaces()
	for _, inter := range interfaces {
		if inter.Name == interfaceName {
			addrs, err := inter.Addrs()
			if err != nil {
				return "", err
			}
			for _, addr := range addrs {
				switch ip := addr.(type) {
				case *net.IPNet:
					if ip.IP.DefaultMask() != nil {
						return ip.IP.String(), nil
					}
				}
			}
		}
	}
	return "", fmt.Errorf("routedinterface not found so can't find ip")
}

func (n *NetImpl) SetUpVeth(ip string, veth0 string, veth1 string) error {
	_, err := net.InterfaceByName(veth0)
	if err == nil {
		err = netlink.NetworkLinkDel(veth0)
		if err != nil {
			return errors.Wrapf(err, "could not delete: %s ", veth0)
		}
	}
	err = netlink.NetworkCreateVethPair(veth0, veth1, 1000)
	if err != nil {
		return errors.Wrapf(err, "could not create veth pair: %s and %s ", veth0, veth1)
	}
	var veth0Interface *net.Interface
	veth0Interface, err = net.InterfaceByName(veth0)
	if err != nil {
		return errors.Wrapf(err, "could not get veth0 interface")
	}
	cidr := fmt.Sprintf("%s/32", ip)
	ipAddr, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return errors.Wrapf(err, "could not parse: %s ", cidr)
	}
	err = netlink.NetworkLinkAddIp(veth0Interface, ipAddr, ipNet)
	if err != nil {
		return errors.Wrapf(err, "could not add ip: %s to %s ", ipAddr, veth0)
	}
	err = netlink.NetworkLinkUp(veth0Interface)
	if err != nil {
		return errors.Wrapf(err, "could not bring up: %s ", veth0)
	}
	return nil
}

func (n *NetImpl) TearDownVeth(veth0 string) error {
	var veth0Interface *net.Interface
	var err error
	veth0Interface, err = net.InterfaceByName(veth0)
	if err != nil {
		return errors.Wrapf(err, "could not get veth0 interface")
	}
	err = netlink.NetworkLinkDown(veth0Interface)
	if err != nil {
		return errors.Wrapf(err, "could not bring down: %s ", veth0)
	}
	return nil
}
