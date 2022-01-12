package addons

import (
	"path"

	bashscript "github.com/platform9/nodelet/pkg/phases/bash_script_based_phases"
	"github.com/platform9/nodelet/pkg/utils/constants"
	sunpikev1alpha1 "github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike/v1alpha1"
)

func NewPF9AddonOperatorPhase(baseDir string) *bashscript.Phase {
	addonOperatorPhase := &bashscript.Phase{
		Filename: path.Join(baseDir, "enable_pf9_addon_operator.sh"),
		HostPhase: &sunpikev1alpha1.HostPhase{
			Name:  "Configure and start pf9-addon-operator",
			Order: int32(constants.PF9AddonOperatorPhaseOrder),
		},
	}
	return addonOperatorPhase
}
