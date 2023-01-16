package etcd

import (
	"context"

	"github.com/platform9/nodelet/nodelet/pkg/utils/config"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	"github.com/platform9/nodelet/nodelet/pkg/utils/etcd"
	"github.com/platform9/nodelet/nodelet/pkg/utils/phaseutils"
	"go.uber.org/zap"

	sunpikev1alpha1 "github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike/v1alpha1"
)

type ConfigureEtcdPhase struct {
	HostPhase *sunpikev1alpha1.HostPhase
	log       *zap.SugaredLogger
	etcd      etcd.EtcdUtils
}

func NewConfigureEtcdPhase() *ConfigureEtcdPhase {
	log := zap.S()
	return &ConfigureEtcdPhase{
		HostPhase: &sunpikev1alpha1.HostPhase{
			Name:  "Configure etcd",
			Order: int32(constants.ConfigureEtcdPhaseOrder),
		},
		log:  log,
		etcd: etcd.New(),
	}
}

func (ce *ConfigureEtcdPhase) GetHostPhase() sunpikev1alpha1.HostPhase {
	return *ce.HostPhase
}

func (ce *ConfigureEtcdPhase) GetPhaseName() string {
	return ce.HostPhase.Name
}

func (ce *ConfigureEtcdPhase) GetOrder() int {
	return int(ce.HostPhase.Order)
}

func (ce *ConfigureEtcdPhase) Status(context.Context, config.Config) error {

	ce.log.Infof("Running Status of phase: %s", ce.HostPhase.Name)
	phaseutils.SetHostStatus(ce.HostPhase, constants.RunningState, "")
	return nil
}

func (ce *ConfigureEtcdPhase) Start(ctx context.Context, cfg config.Config) error {

	ce.log.Infof("Running Start of phase: %s", ce.HostPhase.Name)
	exist, err := ce.etcd.EnsureEtcdDataStoredOnHost()
	if err != nil {
		return err
	}
	if !exist {
		zap.S().Errorf("Skipping etcd backup; etcd container does not exist")
		return nil
	}
	// check if etcd backup and raft index check is required
	// Performed once during
	// 1. new cluster
	// 2. cluster upgrade
	etcdUpgrade, err := ce.etcd.IsEligibleForEtcdBackup()
	if err != nil {
		zap.S().Errorf("failed to check if etcd is eligible for backup: %v", err)
		return err
	}
	if etcdUpgrade {
		ce.log.Infof("etcd to be upgraded. performing etcd data backup")
		err = ce.etcd.EnsureEtcdDataBackup(cfg)
		if err != nil {
			zap.S().Errorf("failed to backup etcd: %v", err)
			return err
		}
	}
	phaseutils.SetHostStatus(ce.HostPhase, constants.RunningState, "")
	return nil
}

func (ce *ConfigureEtcdPhase) Stop(ctx context.Context, cfg config.Config) error {

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
