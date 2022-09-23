package misc

import (
	"context"
	"fmt"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/pkg/errors"
	"github.com/platform9/nodelet/nodelet/pkg/utils/cniutils"
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
	calicoutils cniutils.CalicoUtilsInterface
	netUtils    netutils.NetInterface
}

func NewWaitforK8sPhase() *WaitforK8sPhase {
	return &WaitforK8sPhase{
		HostPhase: &sunpikev1alpha1.HostPhase{
			Name:  "Wait for k8s services",
			Order: int32(constants.WaitForK8sSvcPhaseOrder),
		},
		log:         zap.S(),
		calicoutils: cniutils.New(),
		kubeUtils:   nil,
		netUtils:    netutils.New(),
	}
}

func (k *WaitforK8sPhase) GetHostPhase() sunpikev1alpha1.HostPhase {
	return *k.HostPhase
}

func (k *WaitforK8sPhase) GetPhaseName() string {
	return k.HostPhase.Name
}

func (k *WaitforK8sPhase) GetOrder() int {
	return int(k.HostPhase.Order)
}

func (k *WaitforK8sPhase) Status(ctx context.Context, cfg config.Config) error {

	k.log.Infof("Running Status of phase: %s", k.HostPhase.Name)
	if !k.calicoutils.network_running(cfg) {
		phaseutils.SetHostStatus(k.HostPhase, constants.StoppedState, "")
		return nil
	}
	err := k.kubeUtils.kubernetes_api_available(cfg)
	if err != nil {
		k.log.Error(errors.Wrapf(err, "api not available"))
		phaseutils.SetHostStatus(k.HostPhase, constants.FailedState, err.Error())
		return err
	}
	phaseutils.SetHostStatus(k.HostPhase, constants.RunningState, "")
	return nil
}

func (k *WaitforK8sPhase) Start(ctx context.Context, cfg config.Config) error {
	k.log.Infof("Running Status of phase: %s", k.HostPhase.Name)
	err := k.kubeUtils.kubernetes_api_available(cfg)
	if err != nil {
		fmt.Println(err.Error())
		fmt.Println("!!!! Error occurred - sleep for 5 second")
		time.Sleep(time.Second * 5)
		err = k.kubeUtils.kubernetes_api_available(cfg)

	}
	phaseutils.SetHostStatus(k.HostPhase, constants.RunningState, "")
	return nil

	statusFn := func() error {
		err = k.kubeUtils.kubernetes_api_available(cfg)
		if err != nil {
			return fmt.Errorf("non-zero exit code running status check: %s", phaseutils.Filename)
		}
		return nil
	}
	statusFn = func() error {
		err = k.kubeUtils.local_apiserver_running(cfg)
		if err != nil {
			return fmt.Errorf("non-zero exit code running status check: %s", phaseutils.Filename)
		}
		return nil
	}
	statusFn = func() error {
		err = k.kubeUtils.ensure_role_binding()
		if err != nil {
			return fmt.Errorf("non-zero exit code running status check: %s", phaseutils.Filename)
		}
		return nil
	}
	statusBackoff := getBackOff(phaseutils.Retry - 1)
	backoff.Retry(statusFn, statusBackoff)
	if err != nil {
		return err
	}
	err = k.calicoutils.ensure_network_running(cfg)
	if err != nil {
		k.log.Error(errors.Wrapf(err, "api not available"))
		phaseutils.SetHostStatus(k.HostPhase, constants.FailedState, err.Error())
		return err
	}
	phaseutils.SetHostStatus(k.HostPhase, constants.RunningState, "")
	return nil
	// retry logic

}

func (k *WaitforK8sPhase) Stop(ctx context.Context, cfg config.Config) error {
	k.log.Infof("Running Stop of phase: %s", k.HostPhase.Name)
	phaseutils.SetHostStatus(k.HostPhase, constants.StoppedState, "")
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
