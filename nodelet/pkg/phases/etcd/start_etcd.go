package etcd

import (
	"path"

	bashscript "github.com/platform9/nodelet/nodelet/pkg/phases/bash_script_based_phases"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	sunpikev1alpha1 "github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike/v1alpha1"
)

func NewStartEtcdPhase(baseDir string) *bashscript.Phase {
	startEtcdPhase := &bashscript.Phase{
		Filename: path.Join(baseDir, "etcd_run.sh"),
		HostPhase: &sunpikev1alpha1.HostPhase{
			Name:  "Start etcd",
			Order: int32(constants.StartEtcdPhaseOrder),
		},
	}
	return startEtcdPhase
}
