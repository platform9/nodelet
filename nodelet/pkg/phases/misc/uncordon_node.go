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

type UncordonNodePhase struct {
	HostPhase *sunpikev1alpha1.HostPhase
	log       *zap.SugaredLogger
	kubeUtils kubeutils.Utils
}

func NewUncordonNodePhase() *UncordonNodePhase {
	log := zap.S()
	kubeutils, err := kubeutils.NewClient()
	if err != nil {
		log.Errorf("failed to initiate Uncordon node phase: %w", err)
	}
	return &UncordonNodePhase{
		HostPhase: &sunpikev1alpha1.HostPhase{
			Name:  "Uncordon node",
			Order: int32(constants.UncordonNodePhaseOrder),
		},
		log:       log,
		kubeUtils: kubeutils,
	}
}

func (d *UncordonNodePhase) GetHostPhase() sunpikev1alpha1.HostPhase {
	return *d.HostPhase
}

func (d *UncordonNodePhase) GetPhaseName() string {
	return d.HostPhase.Name
}

func (d *UncordonNodePhase) GetOrder() int {
	return int(d.HostPhase.Order)
}

func (d *UncordonNodePhase) Status(ctx context.Context, cfg config.Config) error {

	nodeIdentifier, err := d.kubeUtils.GetNodeIdentifier(cfg)
	if err != nil {
		d.log.Errorf(err.Error())
		return err
	}

	if _, err := os.Stat(constants.KubeStackStartFileMarker); err == nil {
		d.log.Infof("kube stack is still booting up, nodes not ready yet")
		return nil
	}

	node, err := d.kubeUtils.GetNodeFromK8sApi(ctx, nodeIdentifier)
	if err != nil {
		d.log.Errorf(err.Error())
		return err
	}
	metadata := &node.ObjectMeta

	//if KubeStackShutDown is present then node was cordoned by PF9
	if metadata.Annotations != nil {
		kubeStackShutDown := metadata.Annotations["KubeStackShutDown"]
		if kubeStackShutDown == constants.TrueString {
			return nil
		}
	}
	//if KubeStackShutDown is not present then node was cordoned by the User
	nodeCordoned := node.Spec.Unschedulable
	if nodeCordoned {
		annotsToAdd := map[string]string{
			"UserNodeCordon": "true",
		}
		err = d.kubeUtils.AddAnnotationsToNode(ctx, nodeIdentifier, annotsToAdd)
		if err != nil {
			d.log.Errorf(err.Error())
			return err
		}
	} else if !nodeCordoned {
		annotsToRemove := []string{"UserNodeCordon"}
		err = d.kubeUtils.RemoveAnnotationsFromNode(ctx, nodeIdentifier, annotsToRemove)
		if err != nil {
			d.log.Errorf(err.Error())
			return err
		}
	}
	return nil
}

func (d *UncordonNodePhase) Start(ctx context.Context, cfg config.Config) error {

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

	//remove KubeStackShutDown annotation (if present) as this is kube stack startup
	annotsToRemove := []string{"KubeStackShutDown"}

	err = d.kubeUtils.RemoveAnnotationsFromNode(ctx, nodeIdentifier, annotsToRemove)
	if err != nil {
		d.log.Errorf(err.Error())
		return err
	}

	node, err := d.kubeUtils.GetNodeFromK8sApi(ctx, nodeIdentifier)
	if err != nil {
		d.log.Errorf(err.Error())
		return err
	}
	metadata := node.ObjectMeta

	//check if node cordoned (By User)
	if metadata.Annotations != nil {
		userNodeCordon := metadata.Annotations["UserNodeCordon"]
		//If cordoned by user DO NOT uncordon, exit
		if userNodeCordon == constants.TrueString {
			return nil
		}
	}

	err = d.kubeUtils.UncordonNode(ctx, nodeIdentifier)
	if err != nil {
		d.log.Errorf(err.Error())
		return err
	}
	err = d.kubeUtils.PreventAutoReattach()
	if err != nil {
		d.log.Errorf(err.Error())
		return err
	}
	//post_upgrade_cleanup (not implemented ,not needed)
	return nil
}

func (d *UncordonNodePhase) Stop(ctx context.Context, cfg config.Config) error {
	return nil
}
