package misc

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/platform9/nodelet/nodelet/pkg/utils/config"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	"github.com/platform9/nodelet/nodelet/pkg/utils/kubeutils"
	"github.com/platform9/nodelet/nodelet/pkg/utils/masterless"
	"github.com/platform9/nodelet/nodelet/pkg/utils/netutils"
	"github.com/platform9/nodelet/nodelet/pkg/utils/phaseutils"
	sunpikev1alpha1 "github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike/v1alpha1"

	"go.uber.org/zap"
)

type MiscPhase struct {
	HostPhase  *sunpikev1alpha1.HostPhase
	log        *zap.SugaredLogger
	kubeUtils  kubeutils.Utils
	netUtils   netutils.NetInterface
	masterless masterless.Masterless
}

func NewMiscPhase() *MiscPhase {
	log := zap.S()
	return &MiscPhase{
		HostPhase: &sunpikev1alpha1.HostPhase{
			Name:  "Miscellaneous scripts and checks",
			Order: int32(constants.MiscPhaseOrder),
		},
		log:        log,
		kubeUtils:  nil,
		netUtils:   netutils.New(),
		masterless: nil,
	}
}

var err error

func (m *MiscPhase) GetHostPhase() sunpikev1alpha1.HostPhase {
	return *m.HostPhase
}

func (m *MiscPhase) GetPhaseName() string {
	return m.HostPhase.Name
}

func (m *MiscPhase) GetOrder() int {
	return int(m.HostPhase.Order)
}

func (m *MiscPhase) Status(ctx context.Context, cfg config.Config) error {

	m.log.Infof("Running Status of phase: %s", m.HostPhase.Name)

	if cfg.ClusterRole == constants.RoleMaster {
		return nil
	}

	nodeIdentifier, err := m.netUtils.GetNodeIdentifier(cfg)
	if err != nil {
		m.log.Errorf(err.Error())
		phaseutils.SetHostStatus(m.HostPhase, constants.FailedState, err.Error())
		return err
	}
	m.log.Infof("Node name is %v", nodeIdentifier)

	if nodeIdentifier == constants.LoopBackIpString {
		m.log.Errorf("Fetched node endpoint as 127.0.0.1. Node interface might have lost IP address. Failing.")
		phaseutils.SetHostStatus(m.HostPhase, constants.FailedState, "Fetched node endpoint as 127.0.0.1. Node interface might have lost IP address. Failing.")
		return fmt.Errorf("node interface might have lost IP address. Failing")
	}

	err = m.kubeUtils.K8sApiAvailable(cfg)
	if err != nil {
		m.log.Errorf("api not available :%w", err)
		phaseutils.SetHostStatus(m.HostPhase, constants.FailedState, err.Error())
		return err
	}
	nodeIdentifier, err = m.netUtils.GetNodeIdentifier(cfg)
	if err != nil {
		return err
	}
	//checking if node is Up
	_, err = m.kubeUtils.GetNodeFromK8sApi(ctx, nodeIdentifier)
	if err != nil {
		m.log.Errorf("node %s is not up", nodeIdentifier)
		return err
	}
	//TODO: is it needed to check if node is in ready state
	phaseutils.SetHostStatus(m.HostPhase, constants.RunningState, "")
	return nil
}

func (m *MiscPhase) Start(ctx context.Context, cfg config.Config) error {

	m.log.Infof("Running Start of phase: %s", m.HostPhase.Name)

	var err error
	if m.kubeUtils == nil || m.kubeUtils.IsInterfaceNil() {
		m.kubeUtils, err = kubeutils.NewClient()
		if err != nil {
			m.log.Error(errors.Wrap(err, "could not refresh k8s client"))
			phaseutils.SetHostStatus(m.HostPhase, constants.FailedState, err.Error())
			return err
		}
	}
	m.masterless = masterless.New(cfg)
	if cfg.ClusterRole == constants.RoleWorker {
		err = m.masterless.InitMasterlessWorkerIfNecessary(ctx, cfg)
		if err != nil {
			m.log.Errorf("could not init masterless worker")
			return err
		}
	}
	err = m.kubeUtils.WriteCloudProviderConfig(cfg)
	if err != nil {
		m.log.Errorf("could not write cloud config file")
		return err
	}
	phaseutils.SetHostStatus(m.HostPhase, constants.RunningState, "")
	return nil
}

func (m *MiscPhase) Stop(ctx context.Context, cfg config.Config) error {

	m.log.Infof("Running Stop of phase: %s", m.HostPhase.Name)

	m.masterless = masterless.New(cfg)
	if cfg.ClusterRole == constants.RoleWorker {
		err = m.masterless.TearDownMasterlessWorkerIfNecessary(ctx, cfg)
		if err != nil {
			m.log.Errorf("could not tear down masterless worker")
			return err
		}
	}

	phaseutils.SetHostStatus(m.HostPhase, constants.StoppedState, "")
	return nil
}
