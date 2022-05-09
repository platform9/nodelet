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

type PF9CoreDNSPhase struct {
	HostPhase *sunpikev1alpha1.HostPhase
	log       *zap.SugaredLogger
	kubeUtils kubeutils.Utils
}

func NewPF9CoreDNSPhase() *PF9CoreDNSPhase {
	log := zap.S()
	return &PF9CoreDNSPhase{
		HostPhase: &sunpikev1alpha1.HostPhase{
			Name:  "Configure and start coredns",
			Order: int32(constants.PF9CoreDNSPhaseOrder),
		},
		log:       log,
		kubeUtils: nil,
	}
}

func (c *PF9CoreDNSPhase) GetHostPhase() sunpikev1alpha1.HostPhase {
	return *c.HostPhase
}

func (c *PF9CoreDNSPhase) GetPhaseName() string {
	return c.HostPhase.Name
}

func (c *PF9CoreDNSPhase) GetOrder() int {
	return int(c.HostPhase.Order)
}

func (c *PF9CoreDNSPhase) Status(context.Context, config.Config) error {

	c.log.Infof("Running Status of phase: %s", c.HostPhase.Name)

	phaseutils.SetHostStatus(c.HostPhase, constants.RunningState, "")
	return nil
}

func (c *PF9CoreDNSPhase) Start(ctx context.Context, cfg config.Config) error {

	c.log.Infof("Running Start of phase: %s", c.HostPhase.Name)

	var err error
	if c.kubeUtils == nil || c.kubeUtils.IsInterfaceNil() {
		c.kubeUtils, err = kubeutils.NewClient()
		if err != nil {
			c.log.Error(errors.Wrap(err, "could not refresh k8s client"))
			phaseutils.SetHostStatus(c.HostPhase, constants.FailedState, err.Error())
			return err
		}
	}

	err = c.kubeUtils.EnsureDns(cfg)
	if err != nil {
		c.log.Error(err)
		phaseutils.SetHostStatus(c.HostPhase, constants.FailedState, err.Error())
		return err
	}

	phaseutils.SetHostStatus(c.HostPhase, constants.RunningState, "")
	return nil
}

func (c *PF9CoreDNSPhase) Stop(ctx context.Context, cfg config.Config) error {

	c.log.Infof("Running Stop of phase: %s", c.HostPhase.Name)

	phaseutils.SetHostStatus(c.HostPhase, constants.StoppedState, "")
	return nil
}
