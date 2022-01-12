package keepalived

import (
	"path"

	bashscript "github.com/platform9/nodelet/pkg/phases/bash_script_based_phases"
	"github.com/platform9/nodelet/pkg/utils/constants"
	sunpikev1alpha1 "github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike/v1alpha1"
)

func NewConfigureStartKeepalivedPhase(baseDir string) *bashscript.Phase {
	keepalivedPhase := &bashscript.Phase{
		Filename: path.Join(baseDir, "configure_start_keepalived.sh"),
		HostPhase: &sunpikev1alpha1.HostPhase{
			Name:  "Configure and start Keepalived",
			Order: int32(constants.ConfigureStartKeepalivedPhaseOrder),
		},
	}
	return keepalivedPhase
}
