package addons

import (
	"path"

	bashscript "github.com/platform9/nodelet/nodelet/pkg/phases/bash_script_based_phases"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	sunpikev1alpha1 "github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike/v1alpha1"
)

func NewPF9CoreDNSPhase(baseDir string) *bashscript.Phase {
	coreDNSPhase := &bashscript.Phase{
		Filename: path.Join(baseDir, "enable_coredns.sh"),
		HostPhase: &sunpikev1alpha1.HostPhase{
			Name:  "Configure and start coredns",
			Order: int32(constants.PF9CoreDNSPhaseOrder),
		},
	}
	return coreDNSPhase
}
