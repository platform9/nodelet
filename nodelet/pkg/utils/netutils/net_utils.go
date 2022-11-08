package netutils

import (
	"fmt"
	"net"
	"os"

	"github.com/apparentlymart/go-cidr/cidr"
	"github.com/pkg/errors"
	"github.com/platform9/nodelet/nodelet/pkg/utils/config"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	"github.com/vishvananda/netlink"
	"go.uber.org/zap"
)

type NetImpl struct{}

type NetInterface interface {
	AddrConv(string, int) (string, error)
	IpForHttp(string) (string, error)
	GetNodeIP(bool) (string, error)
	GetHostPrimaryIp(int) (string, error)
	GetNodeIdentifier(config.Config) (string, error)
}

func New() NetInterface {
	return &NetImpl{}
}

// AddrConv generates <pos> th IP from hostCIDR
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
		nodeIdentifier, err = n.GetNodeIP(cfg.IPv6Enabled)
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
func (n *NetImpl) GetNodeIP(v6Enabled bool) (string, error) {
	ipFamily := netlink.FAMILY_V4
	if v6Enabled {
		ipFamily = netlink.FAMILY_V6
	}

	nodeIp, err := n.GetHostPrimaryIp(ipFamily)
	if err != nil {
		return "", errors.Wrap(err, "failed to get node IP")
	}
	return nodeIp, nil
}

// Gets the host's IP address associated with the default gateway
func (n *NetImpl) GetHostPrimaryIp(ipFam int) (string, error) {
	routes, _ := netlink.RouteList(nil, ipFam)
	for _, route := range routes {
		// Skip regular routes, Default routes have Dst as empty
		if route.Dst != nil {
			continue
		}
		ifIndex := route.LinkIndex
		link, err := netlink.LinkByIndex(ifIndex)
		if err != nil {
			zap.S().Warnf("Could not get link for route %+v\n", route)
			continue
		}
		linkAttrs := link.Attrs()
		linkName := linkAttrs.Name
		zap.S().Infof("Route: %+v via %+v dev %s src %+v\n", route.Dst, route.Gw, linkName, route.Src)
		addrs, _ := netlink.AddrList(link, ipFam)
		for _, addr := range addrs {
			ip := addr.IP
			if !ip.IsLinkLocalUnicast() {
				// Return the first non-linklocal IP on NIC that has default route
				// TBD: May not work if IPv6 Privacy Extensions are enabled
				zap.S().Infof("Using external IP = %s\n", ip.String())
				return ip.String(), nil
			}
		}
	}

	err := fmt.Errorf("could not find default external route, please specify externalIp in config")
	return "", err
}
