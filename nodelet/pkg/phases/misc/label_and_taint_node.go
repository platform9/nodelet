package misc

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/platform9/nodelet/nodelet/pkg/utils/config"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	"github.com/platform9/nodelet/nodelet/pkg/utils/kubeutils"
	"github.com/platform9/nodelet/nodelet/pkg/utils/netutils"
	"github.com/platform9/nodelet/nodelet/pkg/utils/phaseutils"
	"go.uber.org/zap"

	sunpikev1alpha1 "github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike/v1alpha1"
	v1 "k8s.io/api/core/v1"
)

type LabelTaintNodePhase struct {
	HostPhase *sunpikev1alpha1.HostPhase
	log       *zap.SugaredLogger
	kubeUtils kubeutils.Utils
	netUtils  netutils.NetInterface
}

func NewLabelTaintNodePhase() *LabelTaintNodePhase {
	log := zap.S()
	return &LabelTaintNodePhase{
		HostPhase: &sunpikev1alpha1.HostPhase{
			Name:  "Apply and validate node taints",
			Order: int32(constants.LabelTaintNodePhaseOrder),
		},
		log: log,
		// When k8s node is being brought up for first time,
		// admin.yaml is not present so its not possible to create k8s client.
		// Lazily create k8s client when needed.
		kubeUtils: nil,
		netUtils:  netutils.New(),
	}
}

//var netUtil = netutils.New()

func (l *LabelTaintNodePhase) GetHostPhase() sunpikev1alpha1.HostPhase {
	return *l.HostPhase
}

func (l *LabelTaintNodePhase) GetPhaseName() string {
	return l.HostPhase.Name
}

func (l *LabelTaintNodePhase) GetOrder() int {
	return int(l.HostPhase.Order)
}

func (l *LabelTaintNodePhase) Status(context.Context, config.Config) error {

	l.log.Infof("Running Status of phase: %s", l.HostPhase.Name)

	phaseutils.SetHostStatus(l.HostPhase, constants.RunningState, "")
	return nil
}

func (l *LabelTaintNodePhase) Start(ctx context.Context, cfg config.Config) error {

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
	nodeIdentifier, err := l.netUtils.GetNodeIdentifier(cfg)
	if err != nil {
		l.log.Errorf(err.Error())
		phaseutils.SetHostStatus(l.HostPhase, constants.FailedState, err.Error())
		return err
	}
	l.log.Infof("Node name is %v", nodeIdentifier)

	if nodeIdentifier == constants.LoopBackIpString {
		l.log.Errorf("Fetched node endpoint as 127.0.0.1. Node interface might have lost IP address. Failing.")
		phaseutils.SetHostStatus(l.HostPhase, constants.FailedState, "Fetched node endpoint as 127.0.0.1. Node interface might have lost IP address. Failing.")
		return fmt.Errorf("node interface might have lost IP address. Failing")
	}

	labelsToAdd := map[string]string{
		"node-role.kubernetes.io/" + cfg.ClusterRole: "",
	}

	err = l.kubeUtils.AddLabelsToNode(ctx, nodeIdentifier, labelsToAdd)
	if err != nil {
		l.log.Errorf(err.Error())
		phaseutils.SetHostStatus(l.HostPhase, constants.FailedState, err.Error())
		return err
	}

	if !cfg.MasterSchedulable && cfg.ClusterRole == constants.RoleMaster {

		taintsToAdd := []*v1.Taint{
			{
				Key:    "node-role.kubernetes.io/master",
				Value:  "true",
				Effect: "NoSchedule",
			},
		}
		err = l.kubeUtils.AddTaintsToNode(ctx, nodeIdentifier, taintsToAdd)
		if err != nil {
			l.log.Errorf(err.Error())
			phaseutils.SetHostStatus(l.HostPhase, constants.FailedState, err.Error())
			return err
		}
	}
	phaseutils.SetHostStatus(l.HostPhase, constants.RunningState, "")
	return nil
}

func (l *LabelTaintNodePhase) Stop(ctx context.Context, cfg config.Config) error {

	l.log.Infof("Running Stop of phase: %s", l.HostPhase.Name)

	phaseutils.SetHostStatus(l.HostPhase, constants.StoppedState, "")
	return nil
}
