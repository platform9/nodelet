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

func NewUncordonNodePhaseV2() *UncordonNodePhasev2 {
	log := zap.S()
	return &UncordonNodePhasev2{
		HostPhase: &sunpikev1alpha1.HostPhase{
			Name:  "Uncordon the node",
			Order: int32(constants.UncordonNodePhaseOrder),
		},
		log: log,
	}
}

func (d *UncordonNodePhasev2) Status(ctx context.Context, cfg config.Config) error {

	var err error
	var nodeIdentifier string
	if cfg.CloudProviderType == "local" && cfg.UseHostname == "true" {

		nodeIdentifier, err = os.Hostname()
		if err != nil {
			d.log.Errorf("failed to get hostName for node_identification: %v", err)
			return err
		}

	} else {

		nodeIdentifier, err = kubeutils.GetNodeIP()
		if err != nil {
			d.log.Errorf("failed to get hostName for node identification: %v", err)
			return err
		}
	}

	if _, err := os.Stat(constants.KubeStackStartFileMarker); err == nil {
		fmt.Println("Kube stack is still booting up, nodes not ready yet")
		d.log.Infof("Kube stack is still booting up, nodes not ready yet")
		return nil
	}

	client, err := kubeutils.NewClient()
	if err != nil {
		d.log.Errorf("failed to get client: %v", err)
		return err
	}
	node, _ := client.GetNodeFromK8sApi(nodeIdentifier)
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
		err = client.AddAnnotationsToNode(nodeIdentifier, annotsToAdd)
		if err != nil {
			d.log.Errorf("failed to add annotations: %v, Error: %v", annotsToAdd, err)
			return err
		}
	} else if !nodeCordoned {
		annotsToRemove := []string{"UserNodeCordon"}

		err = client.RemoveAnnotationsFromNode(nodeIdentifier, annotsToRemove)
		if err != nil {
			d.log.Errorf("failed to remove annotations: %v, Error: %v", annotsToRemove, err)
			return err
		}
	}
	return nil
}

func (d *UncordonNodePhasev2) Start(ctx context.Context, cfg config.Config) error {
	var err error
	var nodeIdentifier string

	if cfg.CloudProviderType == "local" && cfg.UseHostname == "true" {
		nodeIdentifier, err = os.Hostname()
		if err != nil {
			d.log.Errorf("failed to get hostName for node_identification: %v", err)
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
		fmt.Println("Fetched node endpoint as 127.0.0.1. Node interface might have lost IP address. Failing.")
		return fmt.Errorf("node interface might have lost IP address. Failing")
	}

	client, err := kubeutils.NewClient()
	if err != nil {
		d.log.Errorf("failed to get client: %v", err)
		return err
	}
	node, _ := client.GetNodeFromK8sApi(nodeIdentifier)
	metadata := node.ObjectMeta

	//remove KubeStackShutDown annotation (if present) as this is kube stack startup
	annotsToRemove := []string{"KubeStackShutDown"}

	err = client.RemoveAnnotationsFromNode(nodeIdentifier, annotsToRemove)
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

	err = client.UncordonNode(nodeIdentifier)
	if err != nil {
		d.log.Errorf("failed to uncordon: %v", err)
		return err
	}
	err = kubeutils.PreventAutoReattach()
	if err != nil {
		return err
	}
	//post_upgrade_cleanup (not implemented ,not needed)
	return nil
}

func (d *UncordonNodePhasev2) Stop(ctx context.Context, cfg config.Config) error {
	return nil
}
