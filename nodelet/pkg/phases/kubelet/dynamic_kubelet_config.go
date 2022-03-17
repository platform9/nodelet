package kubelet

import (
	"path"

	bashscript "github.com/platform9/nodelet/nodelet/pkg/phases/bash_script_based_phases"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	sunpikev1alpha1 "github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike/v1alpha1"
)

func NewDynamicKubeletConfigPhase(baseDir string) *bashscript.Phase {
	dynamicKubeletPhase := &bashscript.Phase{
		Filename: path.Join(baseDir, "dynamic_kubelet_config.sh"),
		HostPhase: &sunpikev1alpha1.HostPhase{
			Name:  "Apply dynamic kubelet configuration",
			Order: int32(constants.DynamicKubeletConfigPhaseOrder),
		},
	}
	return dynamicKubeletPhase
}
