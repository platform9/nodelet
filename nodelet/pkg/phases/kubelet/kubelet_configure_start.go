package kubelet

import (
	"context"
	"github.com/platform9/nodelet/nodelet/pkg/utils/config"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	"github.com/platform9/nodelet/nodelet/pkg/utils/kubeletutils"
	"github.com/platform9/nodelet/nodelet/pkg/utils/netutils"
	"github.com/platform9/nodelet/nodelet/pkg/utils/phaseutils"
	sunpikev1alpha1 "github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike/v1alpha1"
	"go.uber.org/zap"
)

type KubeletConfigureStartPhase struct {
	HostPhase    *sunpikev1alpha1.HostPhase
	log          *zap.SugaredLogger
	kubeletUtils kubeletutils.KubeletUtilsInterface
	netUtils     netutils.NetInterface
}

func NewKubeletConfigureStartPhase() *KubeletConfigureStartPhase {
	return &KubeletConfigureStartPhase{
		HostPhase: &sunpikev1alpha1.HostPhase{
			Name:  "Configure and Start kubelet",
			Order: constants.ConfigureKubeletPhaseOrder,
		},
		log:          zap.S(),
		kubeletUtils: kubeletutils.New(),
		netUtils:     netutils.New(),
	}
}

func (k *KubeletConfigureStartPhase) GetHostPhase() sunpikev1alpha1.HostPhase {
	return *k.HostPhase
}

func (k *KubeletConfigureStartPhase) GetPhaseName() string {
	return k.HostPhase.Name
}

func (k *KubeletConfigureStartPhase) GetOrder() int {
	return int(k.HostPhase.Order)
}

func (k *KubeletConfigureStartPhase) Status(ctx context.Context, cfg config.Config) error {

	k.log.Infof("Running Status of phase: %s", k.HostPhase.Name)
	if !k.kubeletUtils.IsKubeletRunning() {
		phaseutils.SetHostStatus(k.HostPhase, constants.StoppedState, "")
		return nil
	}

	phaseutils.SetHostStatus(k.HostPhase, constants.RunningState, "")
	return nil
}

func (k *KubeletConfigureStartPhase) Start(ctx context.Context, cfg config.Config) error {
	k.log.Infof("Running Status of phase: %s", k.HostPhase.Name)
	err := k.kubeletUtils.EnsureKubeletRunning(cfg)
	if err != nil {
		phaseutils.SetHostStatus(k.HostPhase, constants.FailedState, err.Error())
		return err
	}
	phaseutils.SetHostStatus(k.HostPhase, constants.RunningState, "")
	return nil
}

func (k *KubeletConfigureStartPhase) Stop(ctx context.Context, cfg config.Config) error {
	k.log.Infof("Running Status of phase: %s", k.HostPhase.Name)
	err := k.kubeletUtils.EnsureKubeletStopped()
	if err != nil {
		phaseutils.SetHostStatus(k.HostPhase, constants.FailedState, err.Error())
		return err
	}
	phaseutils.SetHostStatus(k.HostPhase, constants.StoppedState, "")
	return nil
}
