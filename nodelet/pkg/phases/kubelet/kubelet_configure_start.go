package kubelet

import (
	"path"

	bashscript "github.com/platform9/nodelet/pkg/phases/bash_script_based_phases"
	"github.com/platform9/nodelet/pkg/utils/constants"
	sunpikev1alpha1 "github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike/v1alpha1"
)

func NewKubeletConfigureStartPhase(baseDir string) *bashscript.Phase {
	kubeletConfigurePhase := &bashscript.Phase{
		Filename: path.Join(baseDir, "kubelet_configure_start.sh"),
		HostPhase: &sunpikev1alpha1.HostPhase{
			Name:  "Configure and start kubelet",
			Order: int32(constants.ConfigureKubeletPhaseOrder),
		},
	}
	return kubeletConfigurePhase
}
