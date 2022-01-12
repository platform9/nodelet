package norole

import (
	"path"

	bashscript "github.com/platform9/nodelet/pkg/phases/bash_script_based_phases"
	"github.com/platform9/nodelet/pkg/utils/constants"
	sunpikev1alpha1 "github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike/v1alpha1"
)

func NewNoRolePhase(baseDir string) *bashscript.Phase {
	noRolePhase := &bashscript.Phase{
		Filename: path.Join(baseDir, "all-none-scripts.sh"),
		HostPhase: &sunpikev1alpha1.HostPhase{
			Name:  "No role assigned. (Cleanup scripts only)",
			Order: int32(constants.NoRolePhaseOrder),
		},
	}
	return noRolePhase
}
