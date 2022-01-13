package containerruntime

import (
	"path"

	bashscript "github.com/platform9/nodelet/pkg/phases/bash_script_based_phases"
	"github.com/platform9/nodelet/pkg/utils/constants"
	sunpikev1alpha1 "github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike/v1alpha1"
)

func NewConfigureContainerRuntimePhase(baseDir string) *bashscript.Phase {
	runtimeConfigPhase := &bashscript.Phase{
		Filename: path.Join(baseDir, "runtime_configure.sh"),
		HostPhase: &sunpikev1alpha1.HostPhase{
			Name:  "Configure Container Runtime",
			Order: int32(constants.ConfigureRuntimePhaseOrder),
		},
	}
	return runtimeConfigPhase
}
