package cleanup

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/platform9/nodelet/pkg/utils/config"
	"github.com/platform9/nodelet/pkg/utils/constants"
	"github.com/platform9/nodelet/pkg/utils/kubeutils"

	sunpikev1alpha1 "github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike/v1alpha1"
)

type DrainNodePhasev2 struct {
	HostPhase *sunpikev1alpha1.HostPhase
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
		return err
	}
	return nil
}

func (d *DrainNodePhasev2) Stop(context.Context, config.Config) error {

	//TODO : ensure_http_proxy_configured

	if kubeutils.Kubernetes_api_available() {
		node_identifier := os.Getenv("NODE_NAME")
		if os.Getenv("CLOUD_PROVIDER_TYPE") == "local" && os.Getenv("USE_HOSTNAME") == "true" {
			node_identifier = os.Getenv("HOSTNAME")
		}
		err := kubeutils.Drain_node_from_apiserver(node_identifier)
		if err != nil {
			fmt.Println("Warning: failed to drain node")
			log.Println(err)
		}
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
	return &DrainNodePhasev2{
		HostPhase: &sunpikev1alpha1.HostPhase{
			Name:  "Drain all pods (stop only operation)",
			Order: int32(constants.DrainPodsPhaseOrder),
		},
	}
}
