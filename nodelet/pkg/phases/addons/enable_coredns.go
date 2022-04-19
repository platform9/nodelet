package addons

import (
	"context"
	"fmt"
	"net"
	"os"

	"github.com/apparentlymart/go-cidr/cidr"
	"github.com/platform9/nodelet/nodelet/pkg/utils/config"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	"github.com/platform9/nodelet/nodelet/pkg/utils/phaseutils"
	sunpikev1alpha1 "github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike/v1alpha1"
	"go.uber.org/zap"
)

type PF9CoreDNSPhase struct {
	HostPhase *sunpikev1alpha1.HostPhase
	log       *zap.SugaredLogger
}

func NewPF9CoreDNSPhase() *PF9CoreDNSPhase {
	log := zap.S()
	return &PF9CoreDNSPhase{
		HostPhase: &sunpikev1alpha1.HostPhase{
			Name:  "Apply and validate node taints",
			Order: int32(constants.PF9CoreDNSPhaseOrder),
		},
		log: log,
	}
}

func (l *PF9CoreDNSPhase) GetHostPhase() sunpikev1alpha1.HostPhase {
	return *l.HostPhase
}

func (l *PF9CoreDNSPhase) GetPhaseName() string {
	return l.HostPhase.Name
}

func (l *PF9CoreDNSPhase) GetOrder() int {
	return int(l.HostPhase.Order)
}

func (l *PF9CoreDNSPhase) Status(context.Context, config.Config) error {

	l.log.Infof("Running Status of phase: %s", l.HostPhase.Name)

	phaseutils.SetHostStatus(l.HostPhase, constants.RunningState, "")
	return nil
}

func (l *PF9CoreDNSPhase) Start(ctx context.Context, cfg config.Config) error {

	l.log.Infof("Running Start of phase: %s", l.HostPhase.Name)

	err := ensureDns(cfg)

	phaseutils.SetHostStatus(l.HostPhase, constants.RunningState, "")
	return nil
}

func (l *PF9CoreDNSPhase) Stop(ctx context.Context, cfg config.Config) error {

	l.log.Infof("Running Stop of phase: %s", l.HostPhase.Name)

	phaseutils.SetHostStatus(l.HostPhase, constants.StoppedState, "")
	return nil
}

func ensureDns(cfg config.Config) error {
	coreDNSTemplate := fmt.Sprintf("%s/networkapps/coredns.yaml", constants.ConfigSrcDir)
	coreDNSFile := fmt.Sprintf("%s/networkapps/coredns-applied.yaml", constants.ConfigSrcDir)
	var k8sRegistry string
	if cfg.K8sPrivateRegistry == "" {
		k8sRegistry = "k8s.gcr.io"
	} else {
		k8sRegistry = cfg.K8sPrivateRegistry
	}
	//SERVICES_CIDR: 10.21.0.0/22
	//DNS_IP=`bin/addr_conv -cidr "$SERVICES_CIDR" -pos 10`
	DnsIP, _ := phaseutils.AddrConv(cfg.ServicesCIDR)

}

func addrConv(string hostCIDR, int pos) string {
	_, ipnet, errCidr := net.ParseCIDR(hostCIDR)
	if errCidr != nil {
		fmt.Println("None")
		os.Exit(0)
	}
	ip, errPos := cidr.Host(ipnet, pos)
	if errPos != nil {
		fmt.Println("None")
		os.Exit(0)
	}
}
