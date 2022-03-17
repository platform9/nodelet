package misc

import (
	"context"
	"fmt"
	"os"

	"github.com/platform9/nodelet/nodelet/pkg/utils/config"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	"github.com/platform9/nodelet/nodelet/pkg/utils/kubeutils"
	"go.uber.org/zap"

	sunpikev1alpha1 "github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike/v1alpha1"
)

type UncordonNodePhasev2 struct {
	HostPhase *sunpikev1alpha1.HostPhase
	log       *zap.SugaredLogger
	kubeUtils kubeutils.Utils
}

func NewUncordonNodePhaseV2() *UncordonNodePhasev2 {
	log := zap.S()
	kubeutils, err := kubeutils.NewClient()
	if err != nil {
		fmt.Println("failed to initiate uncordon node phase: %w", err)
	}
	return &UncordonNodePhasev2{
		HostPhase: &sunpikev1alpha1.HostPhase{
			Name:  "Uncordon node",
			Order: int32(constants.UncordonNodePhaseOrder),
		},
		log:       log,
		kubeUtils: kubeutils,
	}
}

func (d *UncordonNodePhasev2) GetHostPhase() sunpikev1alpha1.HostPhase {
	return *d.HostPhase
}

func (d *UncordonNodePhasev2) GetPhaseName() string {
	return d.HostPhase.Name
}

func (d *UncordonNodePhasev2) GetOrder() int {
	return int(d.HostPhase.Order)
}

func (d *UncordonNodePhasev2) Status(ctx context.Context, cfg config.Config) error {

	nodeIdentifier, err := d.kubeUtils.GetNodeIdentifier(cfg)
	if err != nil {
		d.log.Errorf(err.Error())
		return err
	}

	if _, err := os.Stat(constants.KubeStackStartFileMarker); err == nil {
		fmt.Println("Kube stack is still booting up, nodes not ready yet")
		d.log.Infof("Kube stack is still booting up, nodes not ready yet")
		return nil
	}

	node, _ := d.kubeUtils.GetNodeFromK8sApi(ctx, nodeIdentifier)
	metadata := &node.ObjectMeta

	//if KubeStackShutDown is present then node was cordoned by PF9
	if metadata.Annotations != nil {
		kubeStackShutDown := metadata.Annotations["KubeStackShutDown"]
		if kubeStackShutDown == "true" {
			return nil
		}
	}

	nodeCordoned := node.Spec.Unschedulable
	if nodeCordoned {
		annotsToAdd := map[string]string{
			"UserNodeCordon": "true",
		}
		err = d.kubeUtils.AddAnnotationsToNode(ctx, nodeIdentifier, annotsToAdd)
		if err != nil {
			d.log.Errorf("failed to add annotations: %v, Error: %v", annotsToAdd, err)
			return err
		}
	} else if !nodeCordoned {
		annotsToRemove := []string{"UserNodeCordon"}

		err = d.kubeUtils.RemoveAnnotationsFromNode(ctx, nodeIdentifier, annotsToRemove)
		if err != nil {
			d.log.Errorf("failed to remove annotations: %v, Error: %v", annotsToRemove, err)
			return err
		}
	}
	return nil
}

func (d *UncordonNodePhasev2) Start(ctx context.Context, cfg config.Config) error {

	nodeIdentifier, err := d.kubeUtils.GetNodeIdentifier(cfg)
	if err != nil {
		d.log.Errorf(err.Error())
		return err
	}

	fmt.Printf("Node name is %v\n", nodeIdentifier)

	if nodeIdentifier == "127.0.0.1" {
		fmt.Println("Fetched node endpoint as 127.0.0.1. Node interface might have lost IP address. Failing.")
		return fmt.Errorf("node interface might have lost IP address. Failing")
	}

	node, _ := d.kubeUtils.GetNodeFromK8sApi(ctx, nodeIdentifier)
	metadata := node.ObjectMeta

	//remove KubeStackShutDown annotation (if present) as this is kube stack startup
	annotsToRemove := []string{"KubeStackShutDown"}

	err = d.kubeUtils.RemoveAnnotationsFromNode(ctx, nodeIdentifier, annotsToRemove)
	if err != nil {
		d.log.Errorf("failed to remove annotations: %v, Error: %v", annotsToRemove, err)
		return err
	}

	//check if node cordoned (By User)
	if metadata.Annotations != nil {
		userNodeCordon := metadata.Annotations["UserNodeCordon"]
		//If cordoned by user DO NOT uncordon, exit
		if userNodeCordon == "true" {
			return nil
		}
	}

	err = d.kubeUtils.UncordonNode(ctx, nodeIdentifier)
	if err != nil {
		d.log.Errorf("failed to uncordon: %v", err)
		return err
	}
	err = d.kubeUtils.PreventAutoReattach()
	if err != nil {
		return err
	}
	//post_upgrade_cleanup (not implemented ,not needed)
	return nil
}

func (d *UncordonNodePhasev2) Stop(ctx context.Context, cfg config.Config) error {
	return nil
}
