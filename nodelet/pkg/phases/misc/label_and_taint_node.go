package misc

import (
	"context"
	"fmt"

	"github.com/platform9/nodelet/nodelet/pkg/utils/config"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	"github.com/platform9/nodelet/nodelet/pkg/utils/kubeutils"
	"go.uber.org/zap"

	sunpikev1alpha1 "github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike/v1alpha1"
	v1 "k8s.io/api/core/v1"
)

type LabelTaintNodePhase struct {
	HostPhase *sunpikev1alpha1.HostPhase
	log       *zap.SugaredLogger
	kubeUtils kubeutils.Utils
}

func NewLabelTaintNodePhase() *LabelTaintNodePhase {
	log := zap.S()
	kubeutils, err := kubeutils.NewClient()
	if err != nil {
		log.Errorf("failed to initiate Apply and validate node taints phase: %w", err)
	}
	return &LabelTaintNodePhase{
		HostPhase: &sunpikev1alpha1.HostPhase{
			Name:  "Apply and validate node taints",
			Order: int32(constants.LabelTaintNodePhaseOrder),
		},
		log:       log,
		kubeUtils: kubeutils,
	}
}

func (d *LabelTaintNodePhase) GetHostPhase() sunpikev1alpha1.HostPhase {
	return *d.HostPhase
}

func (d *LabelTaintNodePhase) GetPhaseName() string {
	return d.HostPhase.Name
}

func (d *LabelTaintNodePhase) GetOrder() int {
	return int(d.HostPhase.Order)
}

func (d *LabelTaintNodePhase) Status(context.Context, config.Config) error {
	return nil
}

func (d *LabelTaintNodePhase) Start(ctx context.Context, cfg config.Config) error {

	nodeIdentifier, err := d.kubeUtils.GetNodeIdentifier(cfg)
	if err != nil {
		d.log.Errorf(err.Error())
		return err
	}
	d.log.Infof("Node name is %v", nodeIdentifier)

	if nodeIdentifier == constants.LoopBackIpString {
		d.log.Errorf("Fetched node endpoint as 127.0.0.1. Node interface might have lost IP address. Failing.")
		return fmt.Errorf("node interface might have lost IP address. Failing")
	}

	labelsToAdd := map[string]string{
		"node-role.kubernetes.io/" + cfg.ClusterRole: "",
	}

	err = d.kubeUtils.AddLabelsToNode(ctx, nodeIdentifier, labelsToAdd)
	if err != nil {
		d.log.Errorf(err.Error())
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
		err = d.kubeUtils.AddTaintsToNode(ctx, nodeIdentifier, taintsToAdd)
		if err != nil {
			d.log.Errorf(err.Error())
			return err
		}
	}
	return nil
}

func (d *LabelTaintNodePhase) Stop(ctx context.Context, cfg config.Config) error {
	return nil
}
