package cleanup

import (
	"context"
	"os"

	"github.com/pkg/errors"
	"github.com/platform9/nodelet/nodelet/pkg/utils/config"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	"github.com/platform9/nodelet/nodelet/pkg/utils/kubeutils"
	"github.com/platform9/nodelet/nodelet/pkg/utils/phaseutils"
	"go.uber.org/zap"

	sunpikev1alpha1 "github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike/v1alpha1"
)

type DrainNodePhase struct {
	HostPhase *sunpikev1alpha1.HostPhase
	log       *zap.SugaredLogger
	kubeUtils kubeutils.Utils
}

func NewDrainNodePhase() *DrainNodePhase {
	log := zap.S()
	return &DrainNodePhase{
		HostPhase: &sunpikev1alpha1.HostPhase{
			Name:  "Drain all pods (stop only operation)",
			Order: int32(constants.DrainPodsPhaseOrder),
		},
		log: log,
		// When k8s node is being brought up for first time,
		// admin.yaml is not present so its not possible to create k8s client.
		// Lazily create k8s client when needed.
		kubeUtils: nil,
	}

}

func (d *DrainNodePhase) GetHostPhase() sunpikev1alpha1.HostPhase {
	return *d.HostPhase
}

func (d *DrainNodePhase) GetPhaseName() string {
	return d.HostPhase.Name
}

func (d *DrainNodePhase) GetOrder() int {
	return int(d.HostPhase.Order)
}

func (d *DrainNodePhase) Status(context.Context, config.Config) error {
	phaseutils.SetHostStatus(d.HostPhase, constants.RunningState, "")
	return nil
}

func (d *DrainNodePhase) Start(context.Context, config.Config) error {
	err := os.Remove(constants.KubeStackStartFileMarker)
	if err != nil {
		if os.IsNotExist(err) {
			phaseutils.SetHostStatus(d.HostPhase, constants.RunningState, "")
			return nil
		}
		d.log.Errorf("failed to remove KubeStackStartFileMarker file: %w", err)
		phaseutils.SetHostStatus(d.HostPhase, constants.FailedState, err.Error())
		return err
	}
	phaseutils.SetHostStatus(d.HostPhase, constants.RunningState, "")
	return nil
}

func (d *DrainNodePhase) Stop(ctx context.Context, cfg config.Config) error {
	var err error
	if d.kubeUtils == nil || d.kubeUtils.IsInterfaceNil() {
		d.kubeUtils, err = kubeutils.NewClient()
		if err != nil {
			return errors.Wrap(err, "could not refresh k8s client")
		}
	}
	//TODO : ensure_http_proxy_configured
	err = d.kubeUtils.K8sApiAvailable(cfg)
	if err != nil {
		d.log.Errorf("api not available :%w", err)
		phaseutils.SetHostStatus(d.HostPhase, constants.StoppedState, err.Error())
		return err
	}
	nodeIdentifier, err := d.kubeUtils.GetNodeIdentifier(cfg)
	if err != nil {
		d.log.Errorf(err.Error())
		phaseutils.SetHostStatus(d.HostPhase, constants.StoppedState, err.Error())
		return err
	}
	err = d.kubeUtils.DrainNodeFromApiServer(ctx, nodeIdentifier)
	if err != nil {
		d.log.Errorf(err.Error())
		phaseutils.SetHostStatus(d.HostPhase, constants.StoppedState, err.Error())
		return err
	}
	annotsToAdd := map[string]string{
		"KubeStackShutDown": "true",
	}
	err = d.kubeUtils.AddAnnotationsToNode(ctx, nodeIdentifier, annotsToAdd)
	if err != nil {
		d.log.Errorf(err.Error())
		phaseutils.SetHostStatus(d.HostPhase, constants.StoppedState, err.Error())
		return err
	}
	phaseutils.SetHostStatus(d.HostPhase, constants.StoppedState, "")
	return nil
}
