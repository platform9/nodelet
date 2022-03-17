package containerruntime

import (
	"path"

	bashscript "github.com/platform9/nodelet/nodelet/pkg/phases/bash_script_based_phases"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	sunpikev1alpha1 "github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike/v1alpha1"
)

func NewStartContainerRuntimePhase(baseDir string) *bashscript.Phase {
	startContainerRuntimePhase := &bashscript.Phase{
		Filename: path.Join(baseDir, "start_container_runtime.sh"),
		HostPhase: &sunpikev1alpha1.HostPhase{
			Name:  "Start Container Runtime",
			Order: int32(constants.StartRuntimePhaseOrder),
		},
	}
	return startContainerRuntimePhase
}
