package addons

import (
	"path"

	bashscript "github.com/platform9/nodelet/nodelet/pkg/phases/bash_script_based_phases"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	sunpikev1alpha1 "github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike/v1alpha1"
)

func NewDeployAppCatalogPhase(baseDir string) *bashscript.Phase {
	deployAppCatalogPhase := &bashscript.Phase{
		Filename: path.Join(baseDir, "deploy_app_catalog.sh"),
		HostPhase: &sunpikev1alpha1.HostPhase{
			Name:  "Deploy app catalog",
			Order: int32(constants.DeployAppCatalogPhaseOrder),
		},
	}
	return deployAppCatalogPhase
}
