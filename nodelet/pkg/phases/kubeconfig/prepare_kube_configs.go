package kubeconfig

import (
	"path"

	bashscript "github.com/platform9/nodelet/pkg/phases/bash_script_based_phases"
	"github.com/platform9/nodelet/pkg/utils/constants"
	sunpikev1alpha1 "github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike/v1alpha1"
)

func NewPrepareKubeconfigsPhase(baseDir string) *bashscript.Phase {
	prepKubeConfigPhase := &bashscript.Phase{
		Filename: path.Join(baseDir, "prepare_kube_configs.sh"),
		HostPhase: &sunpikev1alpha1.HostPhase{
			Name:  "Prepare configuration",
			Order: int32(constants.KubeconfigPhaseOrder),
		},
	}
	return prepKubeConfigPhase
}
