package network

import (
	"path"

	bashscript "github.com/platform9/nodelet/nodelet/pkg/phases/bash_script_based_phases"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	sunpikev1alpha1 "github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike/v1alpha1"
)

func NewConfigureNetworkPhase(baseDir string) *bashscript.Phase {
	networkConfigurePhase := &bashscript.Phase{
		Filename: path.Join(baseDir, "network_configure.sh"),
		HostPhase: &sunpikev1alpha1.HostPhase{
			Name:  "Network configuration",
			Order: int32(constants.ConfigureNetworkPhaseOrder),
		},
	}
	return networkConfigurePhase
}
