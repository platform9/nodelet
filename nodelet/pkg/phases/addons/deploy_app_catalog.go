package addons

import (
	"context"

	"github.com/platform9/nodelet/nodelet/pkg/utils/config"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	"github.com/platform9/nodelet/nodelet/pkg/utils/kubeutils"
	"github.com/platform9/nodelet/nodelet/pkg/utils/phaseutils"
	sunpikev1alpha1 "github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike/v1alpha1"

	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type DeployAppCatalogPhase struct {
	HostPhase *sunpikev1alpha1.HostPhase
	log       *zap.SugaredLogger
	kubeUtils kubeutils.Utils
}

func NewDeployAppCatalogPhase() *DeployAppCatalogPhase {
	log := zap.S()
	return &DeployAppCatalogPhase{
		HostPhase: &sunpikev1alpha1.HostPhase{
			Name:  "Deploy app catalog",
			Order: int32(constants.DeployAppCatalogPhaseOrder),
		},
		log:       log,
		kubeUtils: nil,
	}
}

func (d *DeployAppCatalogPhase) GetHostPhase() sunpikev1alpha1.HostPhase {
	return *d.HostPhase
}

func (d *DeployAppCatalogPhase) GetPhaseName() string {
	return d.HostPhase.Name
}

func (d *DeployAppCatalogPhase) GetOrder() int {
	return int(d.HostPhase.Order)
}

func (d *DeployAppCatalogPhase) Status(context.Context, config.Config) error {

	d.log.Infof("Running Status of phase: %s", d.HostPhase.Name)

	phaseutils.SetHostStatus(d.HostPhase, constants.RunningState, "")
	return nil
}

func (d *DeployAppCatalogPhase) Start(ctx context.Context, cfg config.Config) error {

	d.log.Infof("Running Start of phase: %s", d.HostPhase.Name)

	var err error
	if d.kubeUtils == nil || d.kubeUtils.IsInterfaceNil() {
		d.kubeUtils, err = kubeutils.NewClient()
		if err != nil {
			d.log.Error(errors.Wrap(err, "could not refresh k8s client"))
			phaseutils.SetHostStatus(d.HostPhase, constants.FailedState, err.Error())
			return err
		}
	}

	if !cfg.AppCatalogEnabled {
		d.log.Warn("app catalog is not enabled")
		phaseutils.SetHostStatus(d.HostPhase, constants.RunningState, "")
		return nil
	}
	err = d.kubeUtils.EnsureAppCatalog()
	if err != nil {
		d.log.Error(err)
		phaseutils.SetHostStatus(d.HostPhase, constants.FailedState, err.Error())
		return err
	}

	phaseutils.SetHostStatus(d.HostPhase, constants.RunningState, "")
	return nil
}

func (d *DeployAppCatalogPhase) Stop(ctx context.Context, cfg config.Config) error {

	d.log.Infof("Running Stop of phase: %s", d.HostPhase.Name)

	phaseutils.SetHostStatus(d.HostPhase, constants.StoppedState, "")
	return nil
}
