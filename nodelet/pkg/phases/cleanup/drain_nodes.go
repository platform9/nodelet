package cleanup

import (
	"context"

	"github.com/platform9/nodelet/pkg/utils/config"
	"github.com/platform9/nodelet/pkg/utils/constants"
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
	return nil
}

func (d *DrainNodePhasev2) Stop(context.Context, config.Config) error {
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
