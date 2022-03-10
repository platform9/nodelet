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

func (d *DrainNodePhasev2) GetHostPhase() sunpikev1alpha1.HostPhase {
	return *d.HostPhase
}

func (d *DrainNodePhasev2) Status(context.Context, config.Config) error {
	return nil
}

func (d *DrainNodePhasev2) Start(context.Context, config.Config) error {

	err := os.Remove(constants.KubeStackStartFileMarker)
	if err != nil {
		d.log.Errorf("failed to remove KubeStackStartFileMarker: %v", err)
		return err
	}
	return nil
}

func (d *DrainNodePhasev2) Stop(ctx context.Context, cfg config.Config) error {

	//TODO : ensure_http_proxy_configured
	err := d.kubeUtils.KubernetesApiAvailable(cfg)
	if err == nil {

		var err error
		var nodeIdentifier string
		if cfg.CloudProviderType == "local" && cfg.UseHostname == "true" {
			nodeIdentifier, err = os.Hostname()
			if err != nil {
				d.log.Errorf("failed to get hostName for node identification: %w", err)
				return err
			}
		} else {
			nodeIdentifier, err = kubeutils.GetNodeIP()
			if err != nil {
				d.log.Errorf("failed to get node IP address for node identification: %w", err)
				return err
			}
		}
		client, err := kubeutils.NewClient()
		if err != nil {
			d.log.Errorf("failed to get client: %v", err)
			return err
		}
		err = client.DrainNodeFromApiServer(nodeIdentifier)
		if err != nil {
			fmt.Println("Warning: failed to drain node")
			d.log.Errorf("failed to drain node :%v", err)
			return err
		}
	} else {
		d.log.Errorf("api not avaialble: %v", err)
		return fmt.Errorf("api not avaialble: %v", err)
	}
	return nil
}

func (d *DrainNodePhasev2) GetPhaseName() string {
	return d.HostPhase.Name
}

func (d *DrainNodePhasev2) GetOrder() int {
	return int(d.HostPhase.Order)
}

func NewDrainNodePhaseV2() *DrainNodePhasev2 {
	log := zap.S()
	// TODO: handle err
	kubeutils, _ := kubeutils.NewClient()
	return &DrainNodePhasev2{
		HostPhase: &sunpikev1alpha1.HostPhase{
			Name:  "Drain all pods (stop only operation)",
			Order: int32(constants.DrainPodsPhaseOrder),
		},
		log:       log,
		kubeUtils: kubeutils,
	}

}
