package containerruntime

import (
	"context"
	"fmt"


	"github.com/coreos/go-systemd/dbus"
	"github.com/platform9/nodelet/nodelet/pkg/utils/config"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	sunpikev1alpha1 "github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike/v1alpha1"
	"go.uber.org/zap"
)


type ContainerdRunPhase struct {
	baseDir   string
	hostPhase *sunpikev1alpha1.HostPhase
	conn	  *dbus.Conn
}


// Extract containerd zip to the specified directory
func NewContainerdRunPhase(baseDir string) (*ContainerdRunPhase, error) {

	conn, err:= dbus.NewSystemConnection()
	if err != nil {
		return nil, fmt.Errorf("error connecting to dbus: %v", err)
	}	

	return newContainerdRunPhaseInternal(conn, baseDir)
}
	
func newContainerdRunPhaseInternal(conn *dbus.Conn, baseDir string) (*ContainerdRunPhase, error) {

	runtimeConfigPhase := &ContainerdRunPhase{
		baseDir: baseDir,
		hostPhase: &sunpikev1alpha1.HostPhase{
			Name:  "Configure Container Runtime",
			Order: int32(constants.ConfigureRuntimePhaseOrder),
		},
		conn: conn,
	}
	return runtimeConfigPhase, nil
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

	// Start the containerd service
	jobId, err := cp.conn.StartUnit("containerd.service", "replace", nil)
	if err != nil {
		return fmt.Errorf("error starting containerd: %v", err)
	}
	zap.S().Infof("Started containerd with job id %s", jobId)
	return nil
}

func (cp *ContainerdRunPhase) Stop(context.Context, config.Config) error {
	// Stop the containerd service
	zap.S().Infof("Stopping containerd")
	_, err := cp.conn.StopUnit("containerd.service", "replace", nil)
	if err != nil {
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
		return fmt.Errorf("error getting containerd status: %v", err)
	}
	if len(unitStatuses) == 0 {
		return fmt.Errorf("containerd service not found")
	}	
	zap.S().Infof("containerd service status: %s", unitStatuses[0].ActiveState)
	// check the actual state of the service
	return nil
}
