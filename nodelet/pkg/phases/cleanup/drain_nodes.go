package cleanup

import (
	"context"
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
	kubeutils, err := kubeutils.NewClient()
	if err != nil {
		log.Errorf("failed to initiate Drain all pods (stop only operation) phase: %v", err)
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
	err := os.Remove(constants.KubeStackStartFileMarker)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		d.log.Errorf("failed to remove KubeStackStartFileMarker file: %v", err)
		return err
	}
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
			d.log.Errorf("failed to drain node :%v", err)
			return err
		}
		annotsToAdd := map[string]string{
			"KubeStackShutDown": "true",
		}
		err = d.kubeUtils.AddAnnotationsToNode(ctx, nodeIdentifier, annotsToAdd)
		if err != nil {
			d.log.Errorf("failed to add annotations: %v beacause of: %v ", annotsToAdd, err)
			return err
		}
	} else {
		d.log.Errorf("api not avaialble: %v", err)
		return err
	}
	return nil
}
