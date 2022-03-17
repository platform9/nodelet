package cleanup

import (
	"context"
	"fmt"
	"os"

	"go.uber.org/zap"

	"github.com/platform9/nodelet/nodelet/pkg/utils/config"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	"github.com/platform9/nodelet/nodelet/pkg/utils/kubeutils"

	sunpikev1alpha1 "github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike/v1alpha1"
)

type DrainNodePhasev2 struct {
	HostPhase *sunpikev1alpha1.HostPhase
	log       *zap.SugaredLogger
	kubeUtils kubeutils.Utils
}

func NewDrainNodePhaseV2() *DrainNodePhasev2 {
	log := zap.S()
	// TODO: handle err
	kubeutils, err := kubeutils.NewClient()
	if err != nil {
		fmt.Println("failed to initiate drain node phase: %w", err)
	}
	return &DrainNodePhasev2{
		HostPhase: &sunpikev1alpha1.HostPhase{
			Name:  "Drain all pods (stop only operation)",
			Order: int32(constants.DrainPodsPhaseOrder),
		},
		log:       log,
		kubeUtils: kubeutils,
	}

}

func (d *DrainNodePhasev2) GetHostPhase() sunpikev1alpha1.HostPhase {
	return *d.HostPhase
}

func (d *DrainNodePhasev2) GetPhaseName() string {
	return d.HostPhase.Name
}

func (d *DrainNodePhasev2) GetOrder() int {
	return int(d.HostPhase.Order)
}

func (d *DrainNodePhasev2) Status(context.Context, config.Config) error {
	return nil
}

func (d *DrainNodePhasev2) Start(context.Context, config.Config) error {
	//fmt.Println("\n\n=========================initiating drain node start====================")
	err := os.Remove(constants.KubeStackStartFileMarker)
	if err != nil {
		d.log.Errorf("failed to remove KubeStackStartFileMarker: %v", err)
		fmt.Printf("failed to remove KubeStackStartFileMarker: %v", err)
		return err
	}
	//fmt.Println("\n\n=========================drain node start succed====================")
	return nil
}

func (d *DrainNodePhasev2) Stop(ctx context.Context, cfg config.Config) error {

	//TODO : ensure_http_proxy_configured
	err := d.kubeUtils.K8sApiAvailable(cfg)
	if err == nil {

		nodeIdentifier, err := d.kubeUtils.GetNodeIdentifier(cfg)
		if err != nil {
			d.log.Errorf(err.Error())
			return err
		}

		err = d.kubeUtils.DrainNodeFromApiServer(ctx, nodeIdentifier)
		if err != nil {
			fmt.Println("Warning: failed to drain node")
			d.log.Errorf("failed to drain node :%v", err)
			return err
		}
	} else {
		d.log.Errorf("api not avaialble: %v", err)
		return err
	}
	return nil
}
