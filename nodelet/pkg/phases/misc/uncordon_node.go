package misc

import (
	"path"

	bashscript "github.com/platform9/nodelet/pkg/phases/bash_script_based_phases"
	"github.com/platform9/nodelet/pkg/utils/constants"
	sunpikev1alpha1 "github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike/v1alpha1"
)

func NewUncordonNodePhase(baseDir string) *bashscript.Phase {
	uncordonNodePhase := &bashscript.Phase{
		Filename: path.Join(baseDir, "uncordon_node.sh"),
		HostPhase: &sunpikev1alpha1.HostPhase{
			Name:  "Uncordon node",
			Order: int32(constants.UncordonNodePhaseOrder),
		},
	}
	return uncordonNodePhase
}
