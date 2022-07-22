package containerruntime

import (
	"context"
	"fmt"

	"github.com/containerd/containerd/namespaces"
	"github.com/coreos/go-systemd/dbus"
	"github.com/platform9/nodelet/nodelet/pkg/utils/config"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	cr "github.com/platform9/nodelet/nodelet/pkg/utils/container_runtime"
	sunpikev1alpha1 "github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike/v1alpha1"
	"go.uber.org/zap"
)

type ContainerdRunPhase struct {
	hostPhase      *sunpikev1alpha1.HostPhase
	conn           *dbus.Conn
	containerUtils cr.ContainerUtils
}

const timeOut = "10s"

// Extract containerd zip to the specified directory
func NewContainerdRunPhase() *ContainerdRunPhase {

	conn, err := dbus.NewSystemConnection()
	if err != nil {
		zap.S().Errorf("error connecting to dbus: %v", err)
	}

	return newContainerdRunPhaseInternal(conn)
}

func newContainerdRunPhaseInternal(conn *dbus.Conn) *ContainerdRunPhase {

	crUtils, err := cr.NewContainerUtil()
	if err != nil {
		zap.S().Errorf("couldn't create containerd client connection")
	}
	runtimeStartPhase := &ContainerdRunPhase{
		hostPhase: &sunpikev1alpha1.HostPhase{
			Name:  "Start Container Runtime",
			Order: int32(constants.StartRuntimePhaseOrder),
		},
		conn:           conn,
		containerUtils: crUtils,
	}
	return runtimeStartPhase
}

// PhaseInterface is an interface to interact with the phases
func (cp *ContainerdRunPhase) GetHostPhase() sunpikev1alpha1.HostPhase {
	return *cp.hostPhase
}

func (cp *ContainerdRunPhase) GetPhaseName() string {
	return cp.hostPhase.Name
}

func (cp *ContainerdRunPhase) GetOrder() int {
	return int(cp.hostPhase.Order)
}

// This code assumes the containerd version is tied to the nodelet version
// in future version we should break that tie
// Extract the Containerd zip to the specified directory
func (cp *ContainerdRunPhase) Start(context.Context, config.Config) error {
	zap.S().Infof("Starting containerd")
	//TODO: configure_containerd_http_proxy
	//TODO: check if "remove_runtime_sock_dir_if_present $socket"is necessary
	// Starting the containerd service
	jobId, err := cp.conn.StartUnit("containerd.service", "replace", nil)
	if err != nil {
		zap.S().Infof("error starting containerd: %v", err)
		return fmt.Errorf("error starting containerd: %v", err)
	}
	zap.S().Infof("Started containerd with job id %s", jobId)
	//TODO: is login to Dockerhub necessary
	return nil
}

func (cp *ContainerdRunPhase) Stop(ctx context.Context, cfg config.Config) error {
	//TODO: destroy all k8s containers.
	containers, err := cp.containerUtils.GetContainersInNamespace(ctx, constants.K8sNamespace)
	if err != nil {
		zap.S().Infof("error getting containers in namespace: %s :%v", constants.K8sNamespace, err)
		return err
	}
	ctx = namespaces.WithNamespace(ctx, constants.K8sNamespace)
	err = cp.containerUtils.EnsureContainersDestroyed(ctx, containers, timeOut)
	if err != nil {
		zap.S().Infof("error destroying containers in namespace: %s :%v", constants.K8sNamespace, err)
		return err
	}
	// Stop the containerd service
	zap.S().Infof("Stopping containerd")
	_, err = cp.conn.StopUnit("containerd.service", "replace", nil)
	if err != nil {
		zap.S().Infof("error stopping containerd: %v", err)
		return fmt.Errorf("error stopping containerd: %v", err)
	}
	zap.S().Infof("Stopped containerd")
	return nil
}

func (cp *ContainerdRunPhase) Status(context.Context, config.Config) error {
	// Get the containerd service status
	zap.S().Infof("Getting containerd status")
	unitStatuses, err := cp.conn.ListUnitsByNames([]string{"containerd.service"})
	if err != nil {
		zap.S().Infof("error getting containerd status: %v", err)
		return fmt.Errorf("error getting containerd status: %v", err)
	}
	if len(unitStatuses) == 0 {
		zap.S().Infof("containerd service not found")
		return fmt.Errorf("containerd service not found")
	}
	zap.S().Infof("containerd service status: %s", unitStatuses[0].ActiveState)
	// check the actual state of the service
	return nil
}
