package containerruntime

import (
	"context"
	"fmt"

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

	// chown "/run/containerd/containerd.sock" to pf9 user
	user := fmt.Sprintf("%s:%s", constants.Pf9User, constants.Pf9Group)
	exitCode, err := cp.cmd.RunCommand(ctx, nil, 0, "", "/usr/bin/sudo", "/usr/bin/chown", user, constants.ContainerdSocket)
	if err != nil || exitCode != 0 {
		phaseutils.SetHostStatus(cp.hostPhase, constants.FailedState, fmt.Sprintf("couldn't own containerd socket:%s to pf9 user: %v", constants.ContainerdSocket, err))
		return errors.Wrapf(err, "couldn't own containerd socket:%s to pf9", constants.ContainerdSocket)
	}

	//TODO: login to Dockerhub(if necessary)
	phaseutils.SetHostStatus(cp.hostPhase, constants.RunningState, "")
	return nil
}

func (cp *ContainerdRunPhase) Stop(ctx context.Context, cfg config.Config) error {

	cp.log.Infof("Running Stop of phase: %s", cp.hostPhase.Name)

	exitCode, err := cp.cmd.RunCommand(ctx, nil, 0, "", constants.ContainerdBinPath, "--version")

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
	defer containerUtil.CloseClient()

	namespaces := []string{constants.K8sNamespace, constants.MobyNamespace}
	err = containerUtil.DestroyContainersInNamespacesList(ctx, namespaces)
	if err != nil {
		cp.log.Errorf("could not destroy containers :%v", err)
		phaseutils.SetHostStatus(cp.hostPhase, constants.FailedState, err.Error())
		return err
	}

	/* deleting /var/lib/nerdctl because, some of the containers are created using nerdctl cli like 'proxy container'
	and some will be created using go-client and we are deleting containers here with go-client.
	so the /var/lib/nerdctl will consist names for ones which are created using nerdctl and deleted using go-client
	and it will give error when created container in next iterarion saying this name is being used by some id. so cleaning this dir.
	TODO: remove this when all phases are converted and using go-client for container creation. */

	exitCode, err = cp.cmd.RunCommand(ctx, nil, 0, "", "/usr/bin/sudo", "/usr/bin/rm", "-rf", constants.NerdctlDir)
	if err != nil || exitCode != 0 {
		phaseutils.SetHostStatus(cp.hostPhase, constants.FailedState, fmt.Sprintf("couldn't delete %s: %v", constants.NerdctlDir, err))
		return errors.Wrapf(err, "couldn't delete %s", constants.NerdctlDir)
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
