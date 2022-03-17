package certs

import (
	"path"

	bashscript "github.com/platform9/nodelet/nodelet/pkg/phases/bash_script_based_phases"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	sunpikev1alpha1 "github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike/v1alpha1"
)

func NewGenCertsPhase(baseDir string) *bashscript.Phase {
	genCertsPhase := &bashscript.Phase{
		Filename: path.Join(baseDir, "gen_certs.sh"),
		HostPhase: &sunpikev1alpha1.HostPhase{
			Name:  "Generate certs / Send signing request to CA",
			Order: int32(constants.GenCertsPhaseOrder),
		},
	}
	return genCertsPhase
}
