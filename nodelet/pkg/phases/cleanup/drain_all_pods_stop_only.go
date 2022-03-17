package cleanup

import (
	"path"

	bashscript "github.com/platform9/nodelet/nodelet/pkg/phases/bash_script_based_phases"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	sunpikev1alpha1 "github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike/v1alpha1"
)

func NewDrainPodsPhase(baseDir string) *bashscript.Phase {
	drainNodePhase := &bashscript.Phase{
		Filename: path.Join(baseDir, "drain_all_pods.sh"),
		HostPhase: &sunpikev1alpha1.HostPhase{
			Name:  "Drain all pods (stop only operation)",
			Order: int32(constants.DrainPodsPhaseOrder),
		},
	}
	return drainNodePhase
}
