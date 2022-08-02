package containerruntime

import (
	"context"

	"github.com/containerd/containerd/namespaces"
	"github.com/coreos/go-systemd/dbus"
	"github.com/pkg/errors"
	"github.com/platform9/nodelet/nodelet/pkg/utils/command"
	"github.com/platform9/nodelet/nodelet/pkg/utils/config"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	cr "github.com/platform9/nodelet/nodelet/pkg/utils/container_runtime"
	sunpikev1alpha1 "github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike/v1alpha1"
	"go.uber.org/zap"
)

type ContainerdRunPhase struct {
	hostPhase *sunpikev1alpha1.HostPhase
	cmd       command.CLI
	log       *zap.SugaredLogger
}

const timeOut = "10s"

func NewContainerdRunPhase() *ContainerdRunPhase {

	runtimeStartPhase := &ContainerdRunPhase{
		hostPhase: &sunpikev1alpha1.HostPhase{
			Name:  "Start Container Runtime",
			Order: int32(constants.StartRuntimePhaseOrder),
		},
		cmd: command.New(),
		log: zap.S(),
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

func (cp *ContainerdRunPhase) Start(ctx context.Context, cfg config.Config) error {

	cp.log.Infof("Running Start of phase: %s", cp.hostPhase.Name)
	//TODO: configure_containerd_http_proxy

	conn, err := dbus.NewSystemConnection()
	if err != nil {
		cp.log.Errorf("error connecting to dbus: %v", err)
	}
	defer conn.Close()

	cp.log.Infof("Starting containerd")
	jobId, err := conn.StartUnit("containerd.service", "replace", nil)
	if err != nil {
		cp.log.Errorf("error starting containerd: %v", err)
		return errors.Wrap(err, "error starting containerd")
	}
	cp.log.Infof("Started containerd with job id: %d", jobId)

	//TODO: login to Dockerhub(if necessary)
	return nil
}

func (cp *ContainerdRunPhase) Stop(ctx context.Context, cfg config.Config) error {

	//TODO: destroy all k8s containers.
	cp.log.Infof("Running Stop of phase: %s", cp.hostPhase.Name)

	exitCode, _, err := cp.cmd.RunCommandWithStdOut(ctx, nil, 0, "", "containerd", "--version")

	if err != nil || exitCode != 0 {
		cp.log.Warn("containerd not present so cant destroy containers: %v", err)
		return nil
	}

	containerUtil, err := cr.NewContainerUtil()
	if err != nil {
		cp.log.Errorf("could not initialise container utility: %v", err)
		return errors.Wrap(err, "could not initialise container utility")
	}

	containers, err := containerUtil.GetContainersInNamespace(ctx, constants.K8sNamespace)
	if err != nil {
		cp.log.Errorf("error getting containers in namespace: %s :%v", constants.K8sNamespace, err)
		return errors.Wrapf(err, "error getting containers in namespace: %s ")
	}

	ctx = namespaces.WithNamespace(ctx, constants.K8sNamespace)

	err = containerUtil.EnsureContainersDestroyed(ctx, containers, timeOut)
	if err != nil {
		cp.log.Errorf("could not destroy containers in namespace: %s :%v", constants.K8sNamespace, err)
		return errors.Wrapf(err, "could not destroy containers in namespace: %s", constants.K8sNamespace)
	}

	conn, err := dbus.NewSystemConnection()
	if err != nil {
		cp.log.Errorf("error connecting to dbus: %v", err)
		return errors.Wrap(err, "error connecting to dbus")
	}
	defer conn.Close()

	// Stop the containerd service
	cp.log.Infof("Stopping containerd")
	_, err = conn.StopUnit("containerd.service", "replace", nil)
	if err != nil {
		cp.log.Errorf("error stopping containerd: %v", err)
		return errors.Wrap(err, "error stopping containerd")
	}
	cp.log.Infof("Stopped containerd")

	return nil
}

func (cp *ContainerdRunPhase) Status(ctx context.Context, cfg config.Config) error {

	cp.log.Infof("Running Status of phase: %s", cp.hostPhase.Name)

	conn, err := dbus.NewSystemConnection()
	if err != nil {
		cp.log.Errorf("error connecting to dbus: %v", err)
		return errors.Wrap(err, "error connecting to dbus")
	}
	defer conn.Close()

	cp.log.Infof("Getting containerd status")
	unitStatuses, err := conn.ListUnitsByNames([]string{"containerd.service"})
	if err != nil {
		cp.log.Infof("error getting containerd status: %v", err)
		return errors.Wrap(err, "error getting containerd status")
	}
	if len(unitStatuses) == 0 {
		cp.log.Infof("Containerd service not found")
		return errors.New("containerd service not found")
	}
	cp.log.Infof("Containerd service status: %s", unitStatuses[0].ActiveState)

	return nil
}
