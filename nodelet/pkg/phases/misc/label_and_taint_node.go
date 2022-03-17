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

type LabelTaintNodePhasev2 struct {
	HostPhase *sunpikev1alpha1.HostPhase
	log       *zap.SugaredLogger
	kubeUtils kubeutils.Utils
}

func NewLabelTaintNodePhaseV2() *LabelTaintNodePhasev2 {
	log := zap.S()
	kubeutils, err := kubeutils.NewClient()
	if err != nil {
		fmt.Println("failed to initiate label and taint node phase: %w", err)
	}
	return &LabelTaintNodePhasev2{
		HostPhase: &sunpikev1alpha1.HostPhase{
			Name:  "Apply and validate node taints",
			Order: int32(constants.LabelTaintNodePhaseOrder),
		},
		log:       log,
		kubeUtils: kubeutils,
	}
}

func (d *LabelTaintNodePhasev2) GetHostPhase() sunpikev1alpha1.HostPhase {
	return *d.HostPhase
}

func (d *LabelTaintNodePhasev2) GetPhaseName() string {
	return d.HostPhase.Name
}

func (d *LabelTaintNodePhasev2) GetOrder() int {
	return int(d.HostPhase.Order)
}

func (d *LabelTaintNodePhasev2) Status(context.Context, config.Config) error {
	return nil
}

func (d *LabelTaintNodePhasev2) Start(ctx context.Context, cfg config.Config) error {

	nodeIdentifier, err := d.kubeUtils.GetNodeIdentifier(cfg)
	if err != nil {
		d.log.Errorf(err.Error())
		return err
	}
	fmt.Printf("Node name is %v\n", nodeIdentifier)

	if nodeIdentifier == "127.0.0.1" {
		d.log.Errorf("Fetched node endpoint as 127.0.0.1. Node interface might have lost IP address. Failing.")
		fmt.Println("Fetched node endpoint as 127.0.0.1. Node interface might have lost IP address. Failing.")
		return fmt.Errorf("node interface might have lost IP address. Failing")
	}

	labelsToAdd := map[string]string{
		"node-role.kubernetes.io/" + cfg.ClusterRole: "",
	}

	err = d.kubeUtils.AddLabelsToNode(ctx, nodeIdentifier, labelsToAdd)
	if err != nil {
		d.log.Errorf("failed to add labels: %v ,Error: %v", labelsToAdd, err)
		return err
	}

	if !cfg.MasterSchedulable && cfg.ClusterRole == "master" {

		taintsToAdd := []*v1.Taint{
			{
				Key:    "node-role.kubernetes.io/master",
				Value:  "true",
				Effect: "NoSchedule", //use TaintEffect which is enum type
			},
		}
		err = d.kubeUtils.AddTaintsToNode(ctx, nodeIdentifier, taintsToAdd)
		if err != nil {
			d.log.Errorf("failed to add taints: %v, Error: %v", labelsToAdd, err)
			return err
		}
	}
	return nil
}

func (d *LabelTaintNodePhasev2) Stop(ctx context.Context, cfg config.Config) error {
	return nil
}
