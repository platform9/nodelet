package masterless

import (
	"context"

	"github.com/pkg/errors"
	"github.com/platform9/nodelet/nodelet/pkg/utils/config"
	"github.com/platform9/nodelet/nodelet/pkg/utils/kubeutils"
	"github.com/platform9/nodelet/nodelet/pkg/utils/netutils"
	"go.uber.org/zap"
)

const (
	veth0 = "pa-proxy-veth0"
	veth1 = "pa-proxy-veth1"
)

type MasterlessImpl struct {
	kubeUtils kubeutils.Utils
	netUtils  netutils.NetInterface
}
type Masterless interface {
	InitMasterlessWorkerIfNecessary(context.Context, config.Config) error
	StartPodToApiserverProxy(ctx context.Context) error
}

func New() Masterless {
	kube, _ := kubeutils.NewClient()
	return &MasterlessImpl{
		kubeUtils: kube,
		netUtils:  netutils.New(),
	}
}
func (m *MasterlessImpl) InitMasterlessWorkerIfNecessary(ctx context.Context, cfg config.Config) error {
	if cfg.MasterlessEnabled {
		err := m.StartPodToApiserverProxy(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *MasterlessImpl) StartPodToApiserverProxy(ctx context.Context) error {
	ip, err := m.kubeUtils.GetApiserverEndpointIp(ctx, "default", "kubernetes")
	if err != nil {
		return errors.Wrap(err, "failed to get endpoint ip for kubernetes")
	}
	zap.S().Infof("apiserver internal endoint ip is: %s", ip)
	// FIXME: KPLAN-72: detect whether this apiserver IP address conflicts
	// with another interface or overlaps a subnet accessible by the host.
	err = m.netUtils.SetUpVeth(ip, veth0, veth1)
	if err != nil {
		return errors.Wrapf(err, "failed to set up %s", veth0)
	}
}
