package misc

import (
	"path"

	bashscript "github.com/platform9/nodelet/pkg/phases/bash_script_based_phases"
	"github.com/platform9/nodelet/pkg/utils/constants"
	sunpikev1alpha1 "github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike/v1alpha1"
)

func NewLabelTaintNodePhase(baseDir string) *bashscript.Phase {
	labelTaintNodePhase := &bashscript.Phase{
		Filename: path.Join(baseDir, "label_and_taint_node.sh"),
		HostPhase: &sunpikev1alpha1.HostPhase{
			Name:  "Apply and validate node taints",
			Order: int32(constants.LabelTaintNodePhaseOrder),
		},
	}
	return labelTaintNodePhase
}
