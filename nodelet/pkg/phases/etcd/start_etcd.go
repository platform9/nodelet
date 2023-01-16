package etcd

import (
	"context"
	"fmt"
	"time"

	"github.com/platform9/nodelet/nodelet/pkg/utils/config"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	"github.com/platform9/nodelet/nodelet/pkg/utils/etcd"
	"github.com/platform9/nodelet/nodelet/pkg/utils/netutils"
	"github.com/platform9/nodelet/nodelet/pkg/utils/phaseutils"
	"go.uber.org/zap"

	sunpikev1alpha1 "github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike/v1alpha1"
)

type StartEtcdPhase struct {
	HostPhase *sunpikev1alpha1.HostPhase
	log       *zap.SugaredLogger
	etcd      etcd.EtcdUtils
}

func NewStartEtcdPhase() *StartEtcdPhase {
	log := zap.S()
	return &StartEtcdPhase{
		HostPhase: &sunpikev1alpha1.HostPhase{
			Name:  "Start etcd",
			Order: int32(constants.StartEtcdPhaseOrder),
		},
		log:  log,
		etcd: etcd.New(),
	}
}

func (ce *StartEtcdPhase) GetHostPhase() sunpikev1alpha1.HostPhase {
	return *ce.HostPhase
}

func (ce *StartEtcdPhase) GetPhaseName() string {
	return ce.HostPhase.Name
}

func (ce *StartEtcdPhase) GetOrder() int {
	return int(ce.HostPhase.Order)
}

func (ce *StartEtcdPhase) Status(ctx context.Context, cfg config.Config) error {

	ce.log.Infof("Running Status of phase: %s", ce.HostPhase.Name)
	running, err := ce.etcd.IsEtcdRunning(ctx)
	if err != nil {
		phaseutils.SetHostStatus(ce.HostPhase, constants.FailedState, err.Error())
		return err
	}
	if !running {
		phaseutils.SetHostStatus(ce.HostPhase, constants.FailedState, "etcd not running")
		return fmt.Errorf("etcd not running")
	}
	phaseutils.SetHostStatus(ce.HostPhase, constants.RunningState, "")
	return nil
}

func (ce *StartEtcdPhase) Start(ctx context.Context, cfg config.Config) error {

	ce.log.Infof("Running Start of phase: %s", ce.HostPhase.Name)
	// check if etcd backup and raft index check is required
	// Performed once during
	// 1. new cluster
	// 2. cluster upgrade
	etcdUpgrade, err := ce.etcd.IsEligibleForEtcdBackup()
	if err != nil {
		zap.S().Errorf("failed to check if etcd is eligible for backup: %v", err)
		return err
	}
	netUtils := netutils.New()
	nodeIdentifier, err := netUtils.GetNodeIdentifier(cfg)
	if err != nil {
		ce.log.Errorf(err.Error())
		phaseutils.SetHostStatus(ce.HostPhase, constants.FailedState, err.Error())
		return err
	}
	ce.log.Infof("Node name is %v", nodeIdentifier)

	if nodeIdentifier == constants.LoopBackIpString {
		ce.log.Errorf("Fetched node endpoint as 127.0.0.1. Node interface might have lost IP address. Failing.")
		phaseutils.SetHostStatus(ce.HostPhase, constants.FailedState, "Fetched node endpoint as 127.0.0.1. Node interface might have lost IP address. Failing.")
		return fmt.Errorf("node interface might have lost IP address. Failing")
	}

	err = ce.etcd.EnsureEtcdRunning(ctx, cfg)
	if err != nil {
		zap.S().Errorf("failed to run etcd: %v", err)
		return err
	}
	if etcdUpgrade {
		zap.S().Info("etcd upgrade done. performing etcd raft index check")
		for i := 0; i < 18; i++ {
			err = ce.etcd.EnsureEtcdClusterStatus()
			if err != nil {
				zap.S().Errorf("failed to ensure etcd cluster status: %v", err)
				time.Sleep(10 * time.Second)
				continue
			}
			break
		}
	}
	phaseutils.SetHostStatus(ce.HostPhase, constants.RunningState, "")
	return nil
}

func (ce *StartEtcdPhase) Stop(ctx context.Context, cfg config.Config) error {

	ce.log.Infof("Running Stop of phase: %s", ce.HostPhase.Name)
	zap.S().Info("Destroying etcd container")
	err := ce.etcd.EnsureEtcdDestroyed(ctx)
	if err != nil {
		zap.S().Errorf("could not destroy etcd container: %v", err)
		return err
	}

	phaseutils.SetHostStatus(ce.HostPhase, constants.StoppedState, "")
	return nil
}
