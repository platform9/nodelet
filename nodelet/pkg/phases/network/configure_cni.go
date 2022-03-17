package network

import (
	"path"

	bashscript "github.com/platform9/nodelet/nodelet/pkg/phases/bash_script_based_phases"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	sunpikev1alpha1 "github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike/v1alpha1"
)

func NewConfigureCNIPhase(baseDir string) *bashscript.Phase {
	configureCNIPhase := &bashscript.Phase{
		Filename: path.Join(baseDir, "cni_configure.sh"),
		HostPhase: &sunpikev1alpha1.HostPhase{
			Name:  "Configure CNI plugin",
			Order: int32(constants.ConfigureCNIPhaseOrder),
		},
	}
	return configureCNIPhase
}
