package phases

import (
	"fmt"

	"github.com/platform9/nodelet/nodelet/pkg/phases/addons"
	authwebhhook "github.com/platform9/nodelet/nodelet/pkg/phases/auth_webhook"
	"github.com/platform9/nodelet/nodelet/pkg/phases/certs"
	"github.com/platform9/nodelet/nodelet/pkg/phases/cleanup"
	containerruntime "github.com/platform9/nodelet/nodelet/pkg/phases/container_runtime"
	"github.com/platform9/nodelet/nodelet/pkg/phases/etcd"
	"github.com/platform9/nodelet/nodelet/pkg/phases/keepalived"
	kubeproxy "github.com/platform9/nodelet/nodelet/pkg/phases/kube_proxy"
	"github.com/platform9/nodelet/nodelet/pkg/phases/kubeconfig"
	"github.com/platform9/nodelet/nodelet/pkg/phases/kubelet"
	"github.com/platform9/nodelet/nodelet/pkg/phases/misc"
	"github.com/platform9/nodelet/nodelet/pkg/phases/network"
	norole "github.com/platform9/nodelet/nodelet/pkg/phases/no_role"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
)

// TODO: get rid of phase order

func GetMasterPhases() ([]PhaseInterface, error) {
	masterPhaseList := []PhaseInterface{
		certs.NewGenCertsPhase(constants.PhaseBaseDir),                // Order 20
		kubeconfig.NewPrepareKubeconfigsPhase(constants.PhaseBaseDir), // Order 30
		containerruntime.NewContainerdConfigPhase(),                   // Order 40
		containerruntime.NewContainerdRunPhase(),                      // Order 45
		containerruntime.NewLoadImagePhase(),                          // Order 48
		etcd.NewConfigureEtcdPhase(constants.PhaseBaseDir),            // Order 50
		etcd.NewStartEtcdPhase(constants.PhaseBaseDir),                // Order 55
		network.NewConfigureNetworkPhase(constants.PhaseBaseDir),      // Order 60
		network.NewConfigureCNIPhase(constants.PhaseBaseDir),          // Order 65
		authwebhhook.NewAuthWebhookPhase(constants.PhaseBaseDir),      // Order 70
		misc.NewMiscPhase(), // Order 75
		kubelet.NewKubeletConfigureStartPhase(constants.PhaseBaseDir),       // Order 80
		kubeproxy.NewKubeProxyStartPhase(constants.PhaseBaseDir),            // Order 90
		misc.NewWaitForK8sSvcPhase(constants.PhaseBaseDir),                  // Order 100
		misc.NewLabelTaintNodePhase(),                                       // Order 110
		kubelet.NewDynamicKubeletConfigPhase(constants.PhaseBaseDir),        // Order 120
		misc.NewUncordonNodePhase(),                                         // Order 130
		addons.NewDeployAppCatalogPhase(),                                   // Order 160
		keepalived.NewConfigureStartKeepalivedPhase(constants.PhaseBaseDir), // Order 180
		addons.NewPF9CoreDNSPhase(),                                         // Order 206
		cleanup.NewDrainNodePhase(),                                         // Order 210
	}
	if err := validatePhaseOrdering(masterPhaseList); err != nil {
		return []PhaseInterface{}, err
	}
	return masterPhaseList, nil
}

func GetWorkerPhases() ([]PhaseInterface, error) {
	workerPhaseList := []PhaseInterface{
		certs.NewGenCertsPhase(constants.PhaseBaseDir),                // Order 20
		kubeconfig.NewPrepareKubeconfigsPhase(constants.PhaseBaseDir), // Order 30
		containerruntime.NewContainerdConfigPhase(),                   // Order 40
		containerruntime.NewContainerdRunPhase(),                      // Order 45
		containerruntime.NewLoadImagePhase(),                          // Order 48
		network.NewConfigureNetworkPhase(constants.PhaseBaseDir),      // Order 60
		network.NewConfigureCNIPhase(constants.PhaseBaseDir),          // Order 65
		misc.NewMiscPhase(),                                           // Order 75
		kubelet.NewKubeletConfigureStartPhase(constants.PhaseBaseDir), // Order 80
		kubeproxy.NewKubeProxyStartPhase(constants.PhaseBaseDir),      // Order 90
		misc.NewWaitForK8sSvcPhase(constants.PhaseBaseDir),            // Order 100
		misc.NewLabelTaintNodePhase(),                                 // Order 110
		kubelet.NewDynamicKubeletConfigPhase(constants.PhaseBaseDir),  // Order 120
		misc.NewUncordonNodePhase(),                                   // Order 130
		cleanup.NewDrainNodePhase(),                                   // Order 210
	}
	if err := validatePhaseOrdering(workerPhaseList); err != nil {
		return []PhaseInterface{}, err
	}
	return workerPhaseList, nil
}

func GetNoRolePhases() ([]PhaseInterface, error) {
	return []PhaseInterface{
		norole.NewNoRolePhase(constants.PhaseBaseDir), // Order 10
	}, nil
}

func validatePhaseOrdering(phases []PhaseInterface) error {
	for i := 0; i < len(phases)-1; i++ {
		currPhase := phases[i]
		nextPhase := phases[i+1]
		if currPhase.GetOrder() >= nextPhase.GetOrder() {
			return fmt.Errorf("found 2 phases with same or incorrect order : %s %s", currPhase.GetPhaseName(), nextPhase.GetPhaseName())
		}
	}
	return nil
}
