package misc

import (
	"context"
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/platform9/nodelet/nodelet/pkg/utils/config"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	"github.com/platform9/nodelet/nodelet/pkg/utils/kubeutils"
	"github.com/platform9/nodelet/nodelet/pkg/utils/phaseutils"
	"go.uber.org/zap"

	sunpikev1alpha1 "github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike/v1alpha1"
)

type UncordonNodePhase struct {
	HostPhase *sunpikev1alpha1.HostPhase
	log       *zap.SugaredLogger
	kubeUtils kubeutils.Utils
}

func NewUncordonNodePhase() *UncordonNodePhase {
	log := zap.S()
	return &UncordonNodePhase{
		HostPhase: &sunpikev1alpha1.HostPhase{
			Name:  "Uncordon node",
			Order: int32(constants.UncordonNodePhaseOrder),
		},
		log: log,
		// When k8s node is being brought up for first time,
		// admin.yaml is not present so its not possible to create k8s client.
		// Lazily create k8s client when needed.
		kubeUtils: nil,
	}
}

func (u *UncordonNodePhase) GetHostPhase() sunpikev1alpha1.HostPhase {
	return *u.HostPhase
}

func (u *UncordonNodePhase) GetPhaseName() string {
	return u.HostPhase.Name
}

func (u *UncordonNodePhase) GetOrder() int {
	return int(u.HostPhase.Order)
}

func (u *UncordonNodePhase) Status(ctx context.Context, cfg config.Config) error {

	u.log.Infof("Running Status of phase: %s", u.HostPhase.Name)

	var err error
	if u.kubeUtils == nil || u.kubeUtils.IsInterfaceNil() {
		u.kubeUtils, err = kubeutils.NewClient()
		if err != nil {
			return errors.Wrap(err, "could not refresh k8s client")
		}
	}
	nodeIdentifier, err := u.kubeUtils.GetNodeIdentifier(cfg)
	if err != nil {
		u.log.Errorf(err.Error())
		phaseutils.SetHostStatus(u.HostPhase, constants.FailedState, err.Error())
		return err
	}

	if _, err := os.Stat(constants.KubeStackStartFileMarker); err == nil {
		u.log.Infof("kube stack is still booting up, nodes not ready yet")
		phaseutils.SetHostStatus(u.HostPhase, constants.RunningState, "")
		return nil
	}

	node, err := u.kubeUtils.GetNodeFromK8sApi(ctx, nodeIdentifier)
	if err != nil {
		u.log.Errorf(err.Error())
		phaseutils.SetHostStatus(u.HostPhase, constants.FailedState, err.Error())
		return err
	}
	metadata := &node.ObjectMeta

	//if KubeStackShutDown is present then node was cordoned by PF9
	if metadata.Annotations != nil {
		kubeStackShutDown := metadata.Annotations["KubeStackShutDown"]
		if kubeStackShutDown == constants.TrueString {
			phaseutils.SetHostStatus(u.HostPhase, constants.RunningState, "")
			return nil
		}
	}
	//if KubeStackShutDown is not present then node was cordoned by the User
	nodeCordoned := node.Spec.Unschedulable
	if nodeCordoned {
		annotsToAdd := map[string]string{
			"UserNodeCordon": "true",
		}
		err = u.kubeUtils.AddAnnotationsToNode(ctx, nodeIdentifier, annotsToAdd)
		if err != nil {
			u.log.Errorf(err.Error())
			phaseutils.SetHostStatus(u.HostPhase, constants.FailedState, err.Error())
			return err
		}
	} else if !nodeCordoned {
		annotsToRemove := []string{"UserNodeCordon"}
		err = u.kubeUtils.RemoveAnnotationsFromNode(ctx, nodeIdentifier, annotsToRemove)
		if err != nil {
			u.log.Errorf(err.Error())
			phaseutils.SetHostStatus(u.HostPhase, constants.FailedState, err.Error())
			return err
		}
	}
	phaseutils.SetHostStatus(u.HostPhase, constants.RunningState, "")
	return nil
}

func (u *UncordonNodePhase) Start(ctx context.Context, cfg config.Config) error {

	u.log.Infof("Running Start of phase: %s", u.HostPhase.Name)

	var err error
	if u.kubeUtils == nil {
		u.kubeUtils, err = kubeutils.NewClient()
		if err != nil {
			return errors.Wrap(err, "could not refresh k8s client")
		}
	}
	nodeIdentifier, err := u.kubeUtils.GetNodeIdentifier(cfg)
	if err != nil {
		u.log.Errorf(err.Error())
		phaseutils.SetHostStatus(u.HostPhase, constants.FailedState, err.Error())
		return err
	}

	u.log.Infof("Node name is %v", nodeIdentifier)

	if nodeIdentifier == constants.LoopBackIpString {
		u.log.Errorf("Fetched node endpoint as 127.0.0.1. Node interface might have lost IP address. Failing.")
		phaseutils.SetHostStatus(u.HostPhase, constants.FailedState, "Fetched node endpoint as 127.0.0.1. Node interface might have lost IP address. Failing.")
		return fmt.Errorf("node interface might have lost IP address. Failing")
	}

	//remove KubeStackShutDown annotation (if present) as this is kube stack startup
	annotsToRemove := []string{"KubeStackShutDown"}

	err = u.kubeUtils.RemoveAnnotationsFromNode(ctx, nodeIdentifier, annotsToRemove)
	if err != nil {
		u.log.Errorf(err.Error())
		phaseutils.SetHostStatus(u.HostPhase, constants.FailedState, err.Error())
		return err
	}

	node, err := u.kubeUtils.GetNodeFromK8sApi(ctx, nodeIdentifier)
	if err != nil {
		u.log.Errorf(err.Error())
		phaseutils.SetHostStatus(u.HostPhase, constants.FailedState, err.Error())
		return err
	}
	metadata := node.ObjectMeta

	//check if node cordoned (By User)
	if metadata.Annotations != nil {
		userNodeCordon := metadata.Annotations["UserNodeCordon"]
		//If cordoned by user DO NOT uncordon, exit
		if userNodeCordon == constants.TrueString {
			phaseutils.SetHostStatus(u.HostPhase, constants.RunningState, "")
			return nil
		}
	}

	err = u.kubeUtils.UncordonNode(ctx, nodeIdentifier)
	if err != nil {
		u.log.Errorf(err.Error())
		phaseutils.SetHostStatus(u.HostPhase, constants.FailedState, err.Error())
		return err
	}
	err = u.kubeUtils.PreventAutoReattach()
	if err != nil {
		u.log.Errorf(err.Error())
		phaseutils.SetHostStatus(u.HostPhase, constants.FailedState, err.Error())
		return err
	}
	//post_upgrade_cleanup (not implemented ,not needed)
	phaseutils.SetHostStatus(u.HostPhase, constants.RunningState, "")
	return nil
}

func (u *UncordonNodePhase) Stop(ctx context.Context, cfg config.Config) error {

	u.log.Infof("Running Stop of phase: %s", u.HostPhase.Name)

	phaseutils.SetHostStatus(u.HostPhase, constants.StoppedState, "")
	return nil
}
