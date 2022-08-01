package containerruntime

import (
	"context"
	"fmt"

	"github.com/coreos/go-systemd/dbus"
	"github.com/platform9/nodelet/nodelet/pkg/utils/command"
	"github.com/platform9/nodelet/nodelet/pkg/utils/config"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	cr "github.com/platform9/nodelet/nodelet/pkg/utils/container_runtime"
	sunpikev1alpha1 "github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike/v1alpha1"
	"go.uber.org/zap"
)

type ContainerdConfigPhase struct {
	hostPhase         *sunpikev1alpha1.HostPhase
	containerdInstall cr.InstallRuntime
}

func NewContainerdConfigPhase() *ContainerdConfigPhase {

	runtimeConfigPhase := &ContainerdConfigPhase{
		hostPhase: &sunpikev1alpha1.HostPhase{
			Name:  "Configure Container Runtime",
			Order: int32(constants.ConfigureRuntimePhaseOrder),
		},
		containerdInstall: cr.NewContainerd(),
	}
	return runtimeConfigPhase
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

	cmd := command.New()
	exitCode, output, err := cmd.RunCommandWithStdOut(ctx, nil, 0, "", "containerd", "--version")

	containerdInstalled := false
	if err == nil && output != nil && exitCode == 0 {
		containerdInstalled = true
	}

	// first make sure if the service exists it is stopped
	if containerdInstalled {

		zap.S().Errorf("containerd already present so first stopping it: %v", err)
		conn, err := dbus.NewSystemConnection()
		if err != nil {
			zap.S().Errorf("error connecting to dbus: %v", err)
		}
		defer conn.Close()

		// Stop the containerd service
		zap.S().Infof("Stopping containerd")
		_, err = conn.StopUnit("containerd.service", "replace", nil)
		if err != nil {
			zap.S().Infof("error stopping containerd: %v", err)
			return fmt.Errorf("error stopping containerd: %v", err)
		}
		zap.S().Infof("Stopped containerd")

	}

	zap.S().Infof("installing containerd")
	err = cp.containerdInstall.EnsureContainerdInstalled(ctx)
	if err != nil {
		zap.S().Infof("Error installing containerd: %v", err)
		return err
	}

	return nil
}

func (cp *ContainerdConfigPhase) Stop(context.Context, config.Config) error {
	return nil
}

func (cp *ContainerdConfigPhase) Status(context.Context, config.Config) error {
	return nil
}
