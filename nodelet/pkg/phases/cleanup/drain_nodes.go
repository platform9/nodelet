package cleanup

import (
	"context"
	"os"

	"github.com/platform9/nodelet/nodelet/pkg/utils/config"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	"github.com/platform9/nodelet/nodelet/pkg/utils/kubeutils"
	"go.uber.org/zap"

	sunpikev1alpha1 "github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike/v1alpha1"
)

type DrainNodePhase struct {
	HostPhase *sunpikev1alpha1.HostPhase
	log       *zap.SugaredLogger
	kubeUtils kubeutils.Utils
}

func NewDrainNodePhase() *DrainNodePhase {
	log := zap.S()
	kubeutils, err := kubeutils.NewClient()
	if err != nil {
		log.Errorf("failed to initiate Drain all pods (stop only operation) phase: %w", err)
	}
	return &DrainNodePhase{
		HostPhase: &sunpikev1alpha1.HostPhase{
			Name:  "Drain all pods (stop only operation)",
			Order: int32(constants.DrainPodsPhaseOrder),
		},
		log:       log,
		kubeUtils: kubeutils,
	}

}

func (d *DrainNodePhase) GetHostPhase() sunpikev1alpha1.HostPhase {
	return *d.HostPhase
}

func (d *DrainNodePhase) GetPhaseName() string {
	return d.HostPhase.Name
}

func (d *DrainNodePhase) GetOrder() int {
	return int(d.HostPhase.Order)
}

func (d *DrainNodePhase) Status(context.Context, config.Config) error {
	return nil
}

func (d *DrainNodePhase) Start(context.Context, config.Config) error {
	err := os.Remove(constants.KubeStackStartFileMarker)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		d.log.Errorf("failed to remove KubeStackStartFileMarker file: %w", err)
		return err
	}
	return nil
}

func (d *DrainNodePhase) Stop(ctx context.Context, cfg config.Config) error {

	//TODO : ensure_http_proxy_configured
	err := d.kubeUtils.K8sApiAvailable(cfg)
	if err != nil {
		d.log.Errorf("api not available :%w", err)
		return err
	}
	nodeIdentifier, err := d.kubeUtils.GetNodeIdentifier(cfg)
	if err != nil {
		d.log.Errorf(err.Error())
		return err
	}
	err = d.kubeUtils.DrainNodeFromApiServer(ctx, nodeIdentifier)
	if err != nil {
		d.log.Errorf(err.Error())
		return err
	}
	annotsToAdd := map[string]string{
		"KubeStackShutDown": "true",
	}
	err = d.kubeUtils.AddAnnotationsToNode(ctx, nodeIdentifier, annotsToAdd)
	if err != nil {
		d.log.Errorf(err.Error())
		return err
	}
	return nil
}
