package misc

import (
	"context"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/pkg/errors"
	"github.com/platform9/nodelet/nodelet/pkg/utils/calicoutils"
	"github.com/platform9/nodelet/nodelet/pkg/utils/config"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	"github.com/platform9/nodelet/nodelet/pkg/utils/kubeutils"
	"github.com/platform9/nodelet/nodelet/pkg/utils/netutils"
	"github.com/platform9/nodelet/nodelet/pkg/utils/phaseutils"
	sunpikev1alpha1 "github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike/v1alpha1"
	"go.uber.org/zap"
)

type WaitforK8sPhase struct {
	HostPhase   *sunpikev1alpha1.HostPhase
	log         *zap.SugaredLogger
	kubeUtils   kubeutils.Utils
	calicoUtils calicoutils.CalicoUtilsInterface
	netUtils    netutils.NetInterface
	Filename    string
	Retry       int
}

func NewWaitForK8sSvcPhase() *WaitforK8sPhase {
	return &WaitforK8sPhase{
		HostPhase: &sunpikev1alpha1.HostPhase{
			Name:  "Wait for k8s services",
			Order: int32(constants.WaitForK8sSvcPhaseOrder),
		},
		log:         zap.S(),
		calicoUtils: calicoutils.New(),
		kubeUtils:   nil,
		netUtils:    netutils.New(),
	}
}

func (w *WaitforK8sPhase) GetHostPhase() sunpikev1alpha1.HostPhase {
	return *w.HostPhase
}

func (w *WaitforK8sPhase) GetPhaseName() string {
	return w.HostPhase.Name
}

func (w *WaitforK8sPhase) GetOrder() int {
	return int(w.HostPhase.Order)
}

func (w *WaitforK8sPhase) Status(ctx context.Context, cfg config.Config) error {

	w.log.Infof("Running Status of phase: %s", w.HostPhase.Name)
	err := w.calicoUtils.NetworkRunning(cfg)
	if err != nil {
		phaseutils.SetHostStatus(w.HostPhase, constants.FailedState, err.Error())
		return nil
	}
	//var err error
	if w.kubeUtils == nil || w.kubeUtils.IsInterfaceNil() {
		w.kubeUtils, err = kubeutils.NewClient()
		if err != nil {
			w.log.Error(errors.Wrap(err, "could not refresh k8s client"))
			phaseutils.SetHostStatus(w.HostPhase, constants.StoppedState, "")
			return err
		}
	}

	err = w.kubeUtils.K8sApiAvailable(cfg)
	if err != nil {
		w.log.Error(errors.Wrapf(err, "api not available"))
		phaseutils.SetHostStatus(w.HostPhase, constants.FailedState, err.Error())
		return err
	}
	phaseutils.SetHostStatus(w.HostPhase, constants.RunningState, "")
	return nil
}

func (w *WaitforK8sPhase) Start(ctx context.Context, cfg config.Config) error {

	statusFn := func() error {
		err := w.kubeUtils.K8sApiAvailable(cfg)
		if err != nil {
			return err
		}
		return nil
	}
	statusFn = func() error {
		err := w.calicoUtils.LocalApiserverRunning(cfg)
		if err != nil {
			return err
		}
		return nil
	}
	statusFn = func() error {
		err := w.calicoUtils.EnsureRoleBinding()
		if err != nil {
			return err
		}
		return nil
	}
	statusBackoff := getBackOff(w.Retry - 1)
	backoff.Retry(statusFn, statusBackoff)
	if statusFn != nil {
		return statusFn()
	}
	err := w.calicoUtils.EnsureNetworkRunning(cfg)
	if err != nil {
		w.log.Error(errors.Wrapf(err, "api not available"))
		phaseutils.SetHostStatus(w.HostPhase, constants.FailedState, err.Error())
		return err
	}
	phaseutils.SetHostStatus(w.HostPhase, constants.RunningState, "")
	return nil
	// retry logic

}

func (w *WaitforK8sPhase) Stop(ctx context.Context, cfg config.Config) error {
	w.log.Infof("Running Stop of phase: %s", w.HostPhase.Name)
	phaseutils.SetHostStatus(w.HostPhase, constants.StoppedState, "")
	return nil
}

func getBackOff(retry int) backoff.BackOff {
	backof := backoff.NewExponentialBackOff()
	backof.InitialInterval = 1 * time.Second
	backof.Multiplier = 2
	if retry <= 0 {
		retry = 1
	}
	return backoff.WithMaxRetries(backoff.NewExponentialBackOff(), uint64(retry))
}
