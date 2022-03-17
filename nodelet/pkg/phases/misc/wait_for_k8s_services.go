package misc

import (
	"path"

	bashscript "github.com/platform9/nodelet/nodelet/pkg/phases/bash_script_based_phases"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	sunpikev1alpha1 "github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike/v1alpha1"
)

func NewWaitForK8sSvcPhase(baseDir string) *bashscript.Phase {
	k8sServicePhase := &bashscript.Phase{
		Filename: path.Join(baseDir, "wait_for_k8s_services.sh"),
		HostPhase: &sunpikev1alpha1.HostPhase{
			Name:  "Wait for k8s services and network to be up",
			Order: int32(constants.WaitForK8sSvcPhaseOrder),
		},
	}
	return k8sServicePhase
}
