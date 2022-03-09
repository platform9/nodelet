package misc

import (
	"context"
	"fmt"
	"os"

	"github.com/platform9/nodelet/pkg/utils/config"
	"github.com/platform9/nodelet/pkg/utils/constants"
	"github.com/platform9/nodelet/pkg/utils/kubeutils"
	"go.uber.org/zap"

	sunpikev1alpha1 "github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike/v1alpha1"
	//corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	corev1 "k8s.io/api/core/v1"
)

type LabelTaintNodePhasev2 struct {
	HostPhase *sunpikev1alpha1.HostPhase
	log       *zap.SugaredLogger
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
	var err error
	var nodeIdentifier string
	if cfg.CloudProviderType == "local" && cfg.UseHostname == "true" {
		nodeIdentifier, err = os.Hostname()
		if err != nil {
			d.log.Errorf("failed to get hostName for node identification: %v", err)
			return err
		}
	} else {
		nodeIdentifier, err = kubeutils.GetNodeIP()
		if err != nil {
			d.log.Errorf("failed to get hostName for node identification: %v", err)
			return err
		}
	}

	fmt.Printf("Node name is %v\n", nodeIdentifier)

	if nodeIdentifier == "127.0.0.1" {
		d.log.Errorf("Fetched node endpoint as 127.0.0.1. Node interface might have lost IP address. Failing.")
		fmt.Println("Fetched node endpoint as 127.0.0.1. Node interface might have lost IP address. Failing.")
		return fmt.Errorf("node interface might have lost IP address. Failing")
	}

	client, err := kubeutils.NewClient()
	if err != nil {
		d.log.Errorf("failed to get client: %v", err)
		return err
	}
	labelsToAdd := map[string]string{
		"node-role.kubernetes.io/" + cfg.ClusterRole: "",
	}

	err = client.AddLabelsToNode(nodeIdentifier, labelsToAdd)
	if err != nil {
		d.log.Errorf("failed to add labels: %v ,Error: %v", labelsToAdd, err)
		return err
	}

	if !cfg.MasterSchedulable && cfg.ClusterRole == "master" {

		taintsToAdd := []*corev1.Taint{
			&corev1.Taint{
				Key:    "node-role.kubernetes.io/master",
				Value:  "true",
				Effect: "NoSchedule", //use TaintEffect which is enum type
			},
		}
		err = client.AddTaintsToNode(nodeIdentifier, taintsToAdd)
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

func NewLabelTaintNodePhaseV2() *LabelTaintNodePhasev2 {
	log := zap.S()
	return &LabelTaintNodePhasev2{
		HostPhase: &sunpikev1alpha1.HostPhase{
			Name:  "label and taint the node",
			Order: int32(constants.LabelTaintNodePhaseOrder),
		},
		log: log,
	}
}
