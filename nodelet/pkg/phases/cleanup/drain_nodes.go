package cleanup

import (
	"context"
	"fmt"
	"os"

	"go.uber.org/zap"

	"github.com/platform9/nodelet/pkg/utils/config"
	"github.com/platform9/nodelet/pkg/utils/constants"
	"github.com/platform9/nodelet/pkg/utils/kubeutils"

	sunpikev1alpha1 "github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike/v1alpha1"
)

type DrainNodePhasev2 struct {
	HostPhase *sunpikev1alpha1.HostPhase
	log       *zap.SugaredLogger
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
	err := kubeutils.Kubernetes_api_available(cfg)
	if err == nil {

		var err error
		routedInterfaceName, err := kubeutils.GetRoutedNetworkInterFace()
		if err != nil {
			d.log.Errorf("failed to get routedNetworkinterface: %v", err)
			return err
		}
		routedIp, err := kubeutils.GetIPv4ForInterfaceName(routedInterfaceName)
		if err != nil {
			d.log.Errorf("failed to get IPv4 for node_identification: %v")
			return err
		}

		var node_identifier string
		if cfg.CloudProviderType == "local" && cfg.UseHostname == "true" {
			node_identifier, err = os.Hostname()
			if err != nil {
				d.log.Errorf("failed to get hostName for node_identification: %v", err)
				return err
			}
		} else {
			node_identifier = routedIp
		}
		err = kubeutils.Drain_node_from_apiserver(node_identifier)
		if err != nil {
			fmt.Println("Warning: failed to drain node")
			d.log.Errorf("failed to drain node :%v", err)
			return err
		}

	} else {

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
	return &DrainNodePhasev2{
		HostPhase: &sunpikev1alpha1.HostPhase{
			Name:  "Drain all pods (stop only operation)",
			Order: int32(constants.DrainPodsPhaseOrder),
		},
		log: log,
	}
}
