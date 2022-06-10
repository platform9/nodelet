package masterless

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/platform9/nodelet/nodelet/pkg/utils/config"
	containerruntime "github.com/platform9/nodelet/nodelet/pkg/utils/container_runtime"
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
	runtime   containerruntime.Runtime
}
type Masterless interface {
	InitMasterlessWorkerIfNecessary(context.Context, config.Config) error
	StartPodToApiserverProxy(ctx context.Context, cfg config.Config) error
	TearDownMasterlessWorkerIfNecessary(context.Context, config.Config) error
}

func New(cfg config.Config) Masterless {
	kube, _ := kubeutils.NewClient()
	rt := containerruntime.NewContainerd()
	if cfg.Runtime == "docker" {
		rt = containerruntime.NewDocker()
	}
	return &MasterlessImpl{
		kubeUtils: kube,
		netUtils:  netutils.New(),
		runtime:   rt,
	}

}
func (m *MasterlessImpl) InitMasterlessWorkerIfNecessary(ctx context.Context, cfg config.Config) error {
	if !cfg.MasterlessEnabled {
		return nil
	}
	err := m.StartPodToApiserverProxy(ctx, cfg)
	if err != nil {
		return err
	}
	return nil
}

func (m *MasterlessImpl) TearDownMasterlessWorkerIfNecessary(ctx context.Context, cfg config.Config) error {
	if !cfg.MasterlessEnabled {
		return nil
	}
	err := m.runtime.EnsureContainerStoppedOrNonExistent(ctx, cfg, "pa-proxy")
	if err != nil {
		return err
	}
	err = m.netUtils.TearDownVeth(veth0)
	if err != nil {
		return errors.Wrap(err, "failed to tear down masterless worker")
	}
	return nil
}

func (m *MasterlessImpl) StartPodToApiserverProxy(ctx context.Context, cfg config.Config) error {
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
	containerImage := fmt.Sprintf("%s/platform9/pa-proxy:latest", cfg.DockerPrivateRegistry)
	if cfg.DockerPrivateRegistry == "" {
		containerImage = "platform9/pa-proxy:latest"
	}
	//TODO: how to send -dest ${EXTERNAL_DNS_NAME} flag
	err = m.runtime.EnsureFreshContainerRunning(ctx, cfg, "pa-proxy", containerImage, ip, "8443")
	if err != nil {
		return err
	}
	return nil
}
