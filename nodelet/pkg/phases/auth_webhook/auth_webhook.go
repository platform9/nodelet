package authwebhook

import (
	"path"

	bashscript "github.com/platform9/nodelet/nodelet/pkg/phases/bash_script_based_phases"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	sunpikev1alpha1 "github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike/v1alpha1"
)

func NewAuthWebhookPhase(baseDir string) *bashscript.Phase {
	authWebhookPhase := &bashscript.Phase{
		Filename: path.Join(baseDir, "auth_webhook.sh"),
		HostPhase: &sunpikev1alpha1.HostPhase{
			Name:  "Configure and start pf9-bouncer",
			Order: int32(constants.AuthWebHookPhaseOrder),
		},
	}
	return authWebhookPhase
}
