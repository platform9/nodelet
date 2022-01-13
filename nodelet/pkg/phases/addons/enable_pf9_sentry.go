package addons

import (
	"path"

	bashscript "github.com/platform9/nodelet/pkg/phases/bash_script_based_phases"
	"github.com/platform9/nodelet/pkg/utils/constants"
	sunpikev1alpha1 "github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike/v1alpha1"
)

func NewPF9SentryPhase(baseDir string) *bashscript.Phase {
	pf9SentryPhase := &bashscript.Phase{
		Filename: path.Join(baseDir, "enable_pf9_sentry.sh"),
		HostPhase: &sunpikev1alpha1.HostPhase{
			Name:  "Configure and start pf9-sentry",
			Order: int32(constants.PF9SentryPhaseOrder),
		},
	}
	return pf9SentryPhase
}
