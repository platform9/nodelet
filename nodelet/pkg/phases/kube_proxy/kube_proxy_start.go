package kubeproxy

import (
	"path"

	bashscript "github.com/platform9/nodelet/nodelet/pkg/phases/bash_script_based_phases"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	sunpikev1alpha1 "github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike/v1alpha1"
)

func NewKubeProxyStartPhase(baseDir string) *bashscript.Phase {
	proxyPhase := &bashscript.Phase{
		Filename: path.Join(baseDir, "kube_proxy_start.sh"),
		HostPhase: &sunpikev1alpha1.HostPhase{
			Name:  "Configure and start kube-proxy",
			Order: int32(constants.KubeProxyPhaseOrder),
		},
	}
	return proxyPhase
}
