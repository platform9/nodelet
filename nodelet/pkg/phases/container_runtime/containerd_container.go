package containerruntime

import (
	"context"
	"embed"
	"fmt"
	"os"
	"path"
	"path/filepath"

	systemd "github.com/coreos/go-systemd/dbus"
	"github.com/platform9/nodelet/nodelet/pkg/embedutil"
	"github.com/platform9/nodelet/nodelet/pkg/utils/config"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	sunpikev1alpha1 "github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike/v1alpha1"
	"go.uber.org/zap"
	"golang.org/x/build/internal/untar"
)

//go:embed containerd/*
var containerdZip embed.FS

type ContainerdPhase struct {
	baseDir   string
	hostPhase *sunpikev1alpha1.HostPhase
	embedFs   *embedutil.EmbedFS
	conn	  *systemd.Conn
}

// Extract containerd zip to the specified directory
func NewContainerdPhase(baseDir string) *ContainerdPhase {

	conn, err:= systemd.NewSystemdConnection()
	if err != nil {
		return fmt.Errorf("error connecting to systemd: %v", err)
	}	
	
	embedFs := embedutil.EmbedFS{
		Fs:   containerdZip,
		Root: baseDir,
	}
	runtimeConfigPhase := &ContainerdPhase{
		baseDir: baseDir,
		hostPhase: &sunpikev1alpha1.HostPhase{
			Name:  "Configure Container Runtime",
			Order: int32(constants.ConfigureRuntimePhaseOrder),
		},
		embedFs: &embedFs,
		conn: conn,
	}
	return runtimeConfigPhase
}

// PhaseInterface is an interface to interact with the phases
func (cp *ContainerdPhase) GetHostPhase() sunpikev1alpha1.HostPhase {
	return *cp.hostPhase
}


func (cp *ContainerdPhase) GetPhaseName() string {
	return cp.hostPhase.Name
}

func (cp *ContainerdPhase) GetOrder() int {
	return int(cp.hostPhase.Order)
}


// This code assumes the containerd version is tied to the nodelet version
// in future version we should break that tie
// Extract the Containerd zip to the specified directory
func (cp *ContainerdPhase) Start(context.Context, config.Config) error {
	zap.S().Infof("Extracting containerd zip to %s", cp.baseDir)
	err := cp.embedFs.Extract(cp.baseDir)
	if err != nil {
		return fmt.Errorf("error extracting containerd zip: to baseDir %s,  %v", cp.baseDir, err)
	}

	// now extract the tar files
	zap.S().Infof("Untarring containerd zip to %s", cp.baseDir)

	matches, err := filepath.Glob(path.Join(cp.baseDir, "containerd-*.tar.gz"))
	if err != nil {
		fmt.Errorf("error finding containerd tar files: %v", err)
	}

	for _, match := range matches {
		zap.S().Infof("Untarring %s", match)
		f, err := os.Open(match)
		if err != nil {
			return fmt.Errorf("error opening containerd tar file: %v", err)
		}
		defer f.Close()
		err = untar.Untar(f, cp.baseDir)
		if err != nil {
			return fmt.Errorf("error untarring containerd tar file: %v", err)
		}
	}
	// now reload the systemd daemon and start the service

	// Reload systemd daemon to load new services
	zap.S().Infof("Reloading systemd")
	err = cp.conn.Reload()
	if err != nil {
		return fmt.Errorf("error reloading systemd: %v", err)
	}
	zap.S().Infof("Starting containerd")

	// Start the containerd service
	jobId, err := cp.conn.StartUnit("containerd.service", "replace", nil)
	if err != nil {
		return fmt.Errorf("error starting containerd: %v", err)
	}
	zap.S().Infof("Started containerd with job id %s", jobId)

}

func (cp *ContainerdPhase) Stop(context.Context, config.Config) error {
	// Stop the containerd service
	zap.S().Infof("Stopping containerd")
	err := cp.conn.StopUnit("containerd.service", "replace", nil)
	if err != nil {
		return fmt.Errorf("error stopping containerd: %v", err)
	}
	zap.S().Infof("Stopped containerd")
}

func (cp *ContainerdPhase) Status(context.Context, config.Config) error {
	// Get the containerd service status
	zap.S().Infof("Getting containerd status")
	unitStatuses, err := cp.conn.ListUnitsByNames("containerd.service")
	if err != nil {
		return fmt.Errorf("error getting containerd status: %v", err)
	}
	if len(unitStatuses) == 0 {
		return fmt.Errorf("containerd service not found")
	}	
	zap.S().Infof("containerd service status: %s", unitStatuses[0].ActiveState)
}
