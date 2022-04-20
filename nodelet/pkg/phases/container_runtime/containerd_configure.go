package containerruntime

import (
	"context"
	"embed"
	"fmt"

	"path"
	"path/filepath"

	"github.com/coreos/go-systemd/dbus"
	"github.com/platform9/nodelet/nodelet/pkg/untar"

	"github.com/platform9/nodelet/nodelet/pkg/embedutil"
	"github.com/platform9/nodelet/nodelet/pkg/utils/config"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	sunpikev1alpha1 "github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike/v1alpha1"
	"go.uber.org/zap"
)

//go:embed containerd/containerd.tar.gz
var containerdZip embed.FS

type ContainerdConfigPhase struct {
	baseDir   string
	hostPhase *sunpikev1alpha1.HostPhase
	embedFs   *embedutil.EmbedFS
	conn      *dbus.Conn
	containerdRunPhase *ContainerdRunPhase
}

// Extract containerd zip to the specified directory
func NewContainerdConfigPhase(baseDir string) (*ContainerdConfigPhase, error) {

	conn, err := dbus.NewSystemConnection()
	if err != nil {
		return nil, fmt.Errorf("error connecting to dbus: %v", err)
	}

	embedFs := embedutil.EmbedFS{
		Fs:   containerdZip,
		Root: baseDir,
	}
	containerdRunPhase := newContainerdRunPhaseInternal(conn, baseDir)
	runtimeConfigPhase := &ContainerdConfigPhase{
		baseDir: baseDir,
		hostPhase: &sunpikev1alpha1.HostPhase{
			Name:  "Configure Container Runtime",
			Order: int32(constants.ConfigureRuntimePhaseOrder),
		},
		embedFs: &embedFs,
		conn:    conn,
		containerdRunPhase: containerdRunPhase,
	}
	return runtimeConfigPhase, nil
}

// PhaseInterface is an interface to interact with the phases
func (cp *ContainerdConfigPhase) GetHostPhase() sunpikev1alpha1.HostPhase {
	return *cp.hostPhase
}

func (cp *ContainerdConfigPhase) GetPhaseName() string {
	return cp.hostPhase.Name
}

func (cp *ContainerdConfigPhase) GetOrder() int {
	return int(cp.hostPhase.Order)
}

// This code assumes the containerd version is tied to the nodelet version
// in future version we should break that tie
// Extract the Containerd zip to the specified directory
func (cp *ContainerdConfigPhase) Start(context.Context, config.Config) error {

	// first make sure if the service exists it is stopped
	cp.containerdRunPhase.Stop(context.Background(), config.Config{})
	
	zap.S().Infof("Extracting containerd zip to %s", cp.baseDir)
	err := cp.embedFs.Extract(cp.baseDir)
	if err != nil {
		return fmt.Errorf("error extracting containerd zip: to baseDir %s,  %v", cp.baseDir, err)
	}

	// now extract the tar files
	zap.S().Infof("Untarring containerd zip to %s", cp.baseDir)

	matches, err := filepath.Glob(path.Join(cp.baseDir, "containerd*.tar.gz"))
	if err != nil {
		fmt.Errorf("error finding containerd tar files: %v", err)
	}

	for _, match := range matches {
		zap.S().Infof("Untarring %s", match)
		err = untar.Extract(match, cp.baseDir)
		if err != nil {
			return fmt.Errorf("error untarring containerd tar file: %v", err)
		}
	}
	// now reload the dbus daemon and start the service

	// Reload dbus daemon to load new services
	zap.S().Infof("Reloading dbus")
	err = cp.conn.Reload()
	if err != nil {
		return fmt.Errorf("error reloading dbus: %v", err)
	}
	return nil
}

func (cp *ContainerdConfigPhase) Stop(context.Context, config.Config) error {
	return nil
}

func (cp *ContainerdConfigPhase) Status(context.Context, config.Config) error {
	return nil
}
