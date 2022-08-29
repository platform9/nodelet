package etcd

import (
	"context"

	"github.com/platform9/nodelet/nodelet/pkg/utils/config"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	"github.com/platform9/nodelet/nodelet/pkg/utils/phaseutils"
	"go.uber.org/zap"

	sunpikev1alpha1 "github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike/v1alpha1"
)

type ConfigureEtcdPhase struct {
	HostPhase *sunpikev1alpha1.HostPhase
	log       *zap.SugaredLogger
}

func NewConfigureEtcdPhase() *ConfigureEtcdPhase {
	log := zap.S()
	return &ConfigureEtcdPhase{
		HostPhase: &sunpikev1alpha1.HostPhase{
			Name:  "Configure etcd",
			Order: int32(constants.ConfigureEtcdPhaseOrder),
		},
		log: log,
	}

}

func (d *ConfigureEtcdPhase) GetHostPhase() sunpikev1alpha1.HostPhase {
	return *d.HostPhase
}

func (d *ConfigureEtcdPhase) GetPhaseName() string {
	return d.HostPhase.Name
}

func (d *ConfigureEtcdPhase) GetOrder() int {
	return int(d.HostPhase.Order)
}

func (d *ConfigureEtcdPhase) Status(context.Context, config.Config) error {

	d.log.Infof("Running Status of phase: %s", d.HostPhase.Name)

	phaseutils.SetHostStatus(d.HostPhase, constants.RunningState, "")
	return nil
}

func (d *ConfigureEtcdPhase) Start(context.Context, config.Config) error {

	d.log.Infof("Running Start of phase: %s", d.HostPhase.Name)

	phaseutils.SetHostStatus(d.HostPhase, constants.RunningState, "")
	return nil
}

func (d *ConfigureEtcdPhase) Stop(ctx context.Context, cfg config.Config) error {

	d.log.Infof("Running Stop of phase: %s", d.HostPhase.Name)

	phaseutils.SetHostStatus(d.HostPhase, constants.StoppedState, "")
	return nil
}
