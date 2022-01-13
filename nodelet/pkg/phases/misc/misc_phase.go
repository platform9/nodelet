package misc

import (
	"path"

	bashscript "github.com/platform9/nodelet/pkg/phases/bash_script_based_phases"
	"github.com/platform9/nodelet/pkg/utils/constants"
	sunpikev1alpha1 "github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike/v1alpha1"
)

func NewMiscPhase(baseDir string) *bashscript.Phase {
	miscPhase := &bashscript.Phase{
		Filename: path.Join(baseDir, "misc_scripts.sh"),
		HostPhase: &sunpikev1alpha1.HostPhase{
			Name:  "Miscellaneous scripts and checks",
			Order: int32(constants.MiscPhaseOrder),
		},
	}
	return miscPhase
}
