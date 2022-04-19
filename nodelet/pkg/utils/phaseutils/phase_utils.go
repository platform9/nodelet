package phaseutils

import (
	"net"

	"github.com/apparentlymart/go-cidr/cidr"
	sunpikev1alpha1 "github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike/v1alpha1"
)

func SetHostStatus(hostPhase *sunpikev1alpha1.HostPhase, status string, message string) {
	//TODO: Retry
	hostPhase.Status = status
	hostPhase.Message = message
}

func AddrConv(hostCIDR string, pos int) (string, error) {
	_, ipnet, errCidr := net.ParseCIDR(hostCIDR)
	if errCidr != nil {
		return "", errCidr
	}
	ip, errPos := cidr.Host(ipnet, pos)
	if errPos != nil {
		return "", errPos
	}
	return ip, nil
}
