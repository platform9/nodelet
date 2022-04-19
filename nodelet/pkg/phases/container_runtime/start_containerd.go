package containerruntime

import (
	"context"

	"github.com/pkg/errors"
	"github.com/platform9/nodelet/nodelet/pkg/utils/command"
	"github.com/platform9/nodelet/nodelet/pkg/utils/config"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	cr "github.com/platform9/nodelet/nodelet/pkg/utils/container_runtime"
	"github.com/platform9/nodelet/nodelet/pkg/utils/phaseutils"
	sunpikev1alpha1 "github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike/v1alpha1"
	"go.uber.org/zap"
)

type ContainerdRunPhase struct {
	hostPhase   *sunpikev1alpha1.HostPhase
	cmd         command.CLI
	log         *zap.SugaredLogger
	serviceUtil command.ServiceUtil
}

func NewContainerdRunPhase() *ContainerdRunPhase {

	runtimeStartPhase := &ContainerdRunPhase{
		hostPhase: &sunpikev1alpha1.HostPhase{
			Name:  "Start Container Runtime",
			Order: int32(constants.StartRuntimePhaseOrder),
		},
		cmd:         command.New(),
		log:         zap.S(),
		serviceUtil: command.NewServiceUtil(constants.RuntimeContainerd),
	}
	return runtimeStartPhase
}

func (cp *ContainerdRunPhase) GetHostPhase() sunpikev1alpha1.HostPhase {
	return *cp.hostPhase
}

func (cp *ContainerdRunPhase) GetPhaseName() string {
	return cp.hostPhase.Name
}

func (cp *ContainerdRunPhase) GetOrder() int {
	return int(cp.hostPhase.Order)
}

// TODO: check PF9_MANAGED_DOCKER var
func (cp *ContainerdRunPhase) Start(ctx context.Context, cfg config.Config) error {

	cp.log.Infof("Running Start of phase: %s", cp.hostPhase.Name)
	//TODO: configure_containerd_http_proxy

	_, err := cp.serviceUtil.RunAction(ctx, constants.StartOp)
	if err != nil {
		cp.log.Errorf("could not start containerd: %v", err)
		phaseutils.SetHostStatus(cp.hostPhase, constants.FailedState, err.Error())
		return err
	}
	//TODO: login to Dockerhub(if necessary)
	phaseutils.SetHostStatus(cp.hostPhase, constants.RunningState, "")
	return nil
}

func (cp *ContainerdRunPhase) Stop(ctx context.Context, cfg config.Config) error {

	cp.log.Infof("Running Stop of phase: %s", cp.hostPhase.Name)

	exitCode, err := cp.cmd.RunCommand(ctx, nil, 0, "", constants.RuntimeContainerd, "--version")

	if exitCode != 0 {
		cp.log.Warn("containerd not present so cant destroy containers: %v, exited with exitcode:%d", err, exitCode)
		phaseutils.SetHostStatus(cp.hostPhase, constants.StoppedState, "")
		return nil
	}

	containerUtil, err := cr.NewContainerUtil()
	if err != nil {
		cp.log.Errorf("could not initialise container utility: %v", err)
		phaseutils.SetHostStatus(cp.hostPhase, constants.FailedState, err.Error())
		return err
	}

	namespaces := []string{constants.K8sNamespace, constants.MobyNamespace}
	err = containerUtil.DestroyContainersInNamespacesList(ctx, namespaces)
	if err != nil {
		cp.log.Errorf("could not destroy containers :%v", err)
		phaseutils.SetHostStatus(cp.hostPhase, constants.FailedState, err.Error())
		return err
	}

	// Stop the containerd service
	_, err = cp.serviceUtil.RunAction(ctx, constants.StopOp)
	if err != nil {
		cp.log.Errorf("could not stop containerd: %v", err)
		phaseutils.SetHostStatus(cp.hostPhase, constants.FailedState, err.Error())
		return err
	}

	phaseutils.SetHostStatus(cp.hostPhase, constants.StoppedState, "")
	return nil
}

func (cp *ContainerdRunPhase) Status(ctx context.Context, cfg config.Config) error {

	cp.log.Infof("Running Status of phase: %s", cp.hostPhase.Name)

	output, err := cp.serviceUtil.RunAction(ctx, constants.IsActiveOp)
	if err != nil {
		cp.log.Errorf("could not check containerd status: %v", err)
		phaseutils.SetHostStatus(cp.hostPhase, constants.FailedState, err.Error())
		return err
	}
	if output != nil {
		cp.log.Infof("containerd status:%v", output[0])
		if output[0] == constants.ActiveState {
			phaseutils.SetHostStatus(cp.hostPhase, constants.RunningState, "")
			return nil
		}
		phaseutils.SetHostStatus(cp.hostPhase, constants.FailedState, "containerd not running")
		return errors.Errorf("containerd is not active")
	}
	phaseutils.SetHostStatus(cp.hostPhase, constants.RunningState, "")
	return nil
}
