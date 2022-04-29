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

	var err error
	if l.kubeUtils == nil || l.kubeUtils.IsInterfaceNil() {
		l.kubeUtils, err = kubeutils.NewClient()
		if err != nil {
			l.log.Error(errors.Wrap(err, "could not refresh k8s client"))
			phaseutils.SetHostStatus(l.HostPhase, constants.FailedState, err.Error())
			return err
		}
	}

	err = l.kubeUtils.EnsureDns(cfg)
	if err != nil {
		l.log.Error(err)
		phaseutils.SetHostStatus(l.HostPhase, constants.FailedState, err.Error())
		return err
	}

	phaseutils.SetHostStatus(l.HostPhase, constants.RunningState, "")
	return nil
}

func (l *PF9CoreDNSPhase) Stop(ctx context.Context, cfg config.Config) error {

	l.log.Infof("Running Stop of phase: %s", l.HostPhase.Name)

	phaseutils.SetHostStatus(l.HostPhase, constants.StoppedState, "")
	return nil
}
