package containerruntime

import (
	"context"

	"github.com/coreos/go-systemd/dbus"
	"github.com/pkg/errors"
	"github.com/platform9/nodelet/nodelet/pkg/utils/command"
	"github.com/platform9/nodelet/nodelet/pkg/utils/config"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	cr "github.com/platform9/nodelet/nodelet/pkg/utils/container_runtime"
	"github.com/platform9/nodelet/nodelet/pkg/utils/phaseutils"
	sunpikev1alpha1 "github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike/v1alpha1"
	"go.uber.org/zap"
)

type ContainerdConfigPhase struct {
	hostPhase         *sunpikev1alpha1.HostPhase
	containerdInstall cr.InstallRuntime
	log               *zap.SugaredLogger
	cmd               command.CLI
}

func NewContainerdConfigPhase() *ContainerdConfigPhase {

	containerdConfigPhase := &ContainerdConfigPhase{
		hostPhase: &sunpikev1alpha1.HostPhase{
			Name:  "Configure Container Runtime",
			Order: int32(constants.ConfigureRuntimePhaseOrder),
		},
		containerdInstall: cr.NewContainerd(),
		log:               zap.S(),
		cmd:               command.New(),
	}
	return containerdConfigPhase
}

func (cp *ContainerdConfigPhase) GetHostPhase() sunpikev1alpha1.HostPhase {
	return *cp.hostPhase
}

func (cp *ContainerdConfigPhase) GetPhaseName() string {
	return cp.hostPhase.Name
}

func (cp *ContainerdConfigPhase) GetOrder() int {
	return int(cp.hostPhase.Order)
}

func (cp *ContainerdConfigPhase) Start(ctx context.Context, cfg config.Config) error {

	cp.log.Infof("Running Start of phase: %s", cp.hostPhase.Name)

	exitCode, output, err := cp.cmd.RunCommandWithStdOut(ctx, nil, 0, "", "containerd", "--version")
	if err != nil {
		cp.log.Warnf("error running command:%v", err)
	}

	containerdInstalled := false
	if output != nil && exitCode == 0 {
		containerdInstalled = true
	}

	// first make sure if the service exists it is stopped
	if containerdInstalled {

		cp.log.Warn("containerd already present so first stopping it")

		conn, err := dbus.NewSystemConnection()
		if err != nil {
			cp.log.Errorf("error connecting to dbus:%v", err)
			phaseutils.SetHostStatus(cp.hostPhase, constants.FailedState, err.Error())
			return errors.Wrap(err, "error connecting to dbus")
		}
		defer conn.Close()

		// Stop the containerd service
		cp.log.Infof("Stopping containerd")
		_, err = conn.StopUnit("containerd.service", "replace", nil)
		if err != nil {
			cp.log.Errorf("error stopping containerd: %v", err)
			phaseutils.SetHostStatus(cp.hostPhase, constants.FailedState, err.Error())
			return errors.Wrap(err, "error stopping containerd")
		}
	}

	cp.log.Infof("Installing containerd")
	err = cp.containerdInstall.EnsureContainerdInstalled(ctx)
	if err != nil {
		cp.log.Errorf("error installing containerd: %v", err)
		phaseutils.SetHostStatus(cp.hostPhase, constants.FailedState, err.Error())
		return errors.Wrap(err, "error installing containerd")
	}

	phaseutils.SetHostStatus(cp.hostPhase, constants.RunningState, "")
	return nil
}

func (cp *ContainerdConfigPhase) Stop(context.Context, config.Config) error {
	cp.log.Infof("Running Stop of phase: %s", cp.hostPhase.Name)
	phaseutils.SetHostStatus(cp.hostPhase, constants.StoppedState, "")
	return nil
}

func (cp *ContainerdConfigPhase) Status(context.Context, config.Config) error {
	cp.log.Infof("Running Status of phase: %s", cp.hostPhase.Name)
	phaseutils.SetHostStatus(cp.hostPhase, constants.RunningState, "")
	return nil
}
