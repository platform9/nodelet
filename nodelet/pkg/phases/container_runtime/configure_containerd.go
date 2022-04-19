package containerruntime

import (
	"context"
	"embed"
	"fmt"
	"os"

	"path"
	"path/filepath"

	"github.com/coreos/go-systemd/dbus"
	"github.com/platform9/nodelet/nodelet/pkg/untar"

	"github.com/platform9/nodelet/nodelet/pkg/embedutil"
	"github.com/platform9/nodelet/nodelet/pkg/utils/command"
	"github.com/platform9/nodelet/nodelet/pkg/utils/config"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	"github.com/platform9/nodelet/nodelet/pkg/utils/fileio"
	sunpikev1alpha1 "github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike/v1alpha1"
	"go.uber.org/zap"
)

//go:embed containerd/containerd.tar.gz
var containerdZip embed.FS
var (
	cmdLine = command.New()
	f       = fileio.New()
)

type ContainerdConfigPhase struct {
	hostPhase          *sunpikev1alpha1.HostPhase
	embedFs            *embedutil.EmbedFS
	conn               *dbus.Conn
	containerdRunPhase *ContainerdRunPhase
}

// Extract containerd zip to the specified directory
func NewContainerdConfigPhase() *ContainerdConfigPhase {

	conn, err := dbus.NewSystemConnection()
	if err != nil {
		zap.S().Errorf("error connecting to dbus: %v", err)
	}

	embedFs := embedutil.EmbedFS{
		Fs:   containerdZip,
		Root: "containerd",
	}
	containerdRunPhase := newContainerdRunPhaseInternal(conn)
	if err != nil {
		zap.S().Errorf("error creating containerd run phase: %v", err)
	}
	runtimeConfigPhase := &ContainerdConfigPhase{
		hostPhase: &sunpikev1alpha1.HostPhase{
			Name:  "Configure Container Runtime",
			Order: int32(constants.ConfigureRuntimePhaseOrder),
		},
		embedFs:            &embedFs,
		conn:               conn,
		containerdRunPhase: containerdRunPhase,
	}
	return runtimeConfigPhase
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
func (cp *ContainerdConfigPhase) Start(ctx context.Context, cfg config.Config) error {

	// first make sure if the service exists it is stopped
	err := cp.containerdRunPhase.Stop(context.Background(), config.Config{})
	if err != nil {
		zap.S().Infof("Error while stopping containerd: %v", err)
		//return fmt.Errorf("error stopping containerd: %v", err)
	}

	zap.S().Infof("Extracting containerd zip to %s", constants.PhaseBaseDir)
	err = cp.embedFs.Extract(constants.PhaseBaseDir)
	if err != nil {
		zap.S().Infof("error extracting containerd zip: to baseDir %s,  %v", constants.PhaseBaseDir, err)
		return fmt.Errorf("error extracting containerd zip: to baseDir %s,  %v", constants.PhaseBaseDir, err)
	}

	// now extract the tar files
	zap.S().Infof("Untarring containerd zip to %s", constants.UsrLocalDir)

	matches, err := filepath.Glob(path.Join(constants.PhaseBaseDir, "containerd*.tar.gz"))
	if err != nil {
		zap.S().Infof("error finding containerd tar files: %v", err)
		return fmt.Errorf("error finding containerd tar files: %v", err)
	}

	for _, match := range matches {
		zap.S().Infof("Untarring %s", match)
		err = untar.Extract(match, constants.UsrLocalDir)
		if err != nil {
			zap.S().Infof("error untarring containerd tar file: %v", err)
			return fmt.Errorf("error untarring containerd tar file: %v", err)
		}
	}
	zap.S().Infof("generating containerd.service unit file")
	err = containerdUnit()
	if err != nil {
		zap.S().Infof("error generating containerd.service unit file: %v", err)
		return err
	}

	zap.S().Infof("loading kernel modules")
	modules := []string{"overlay", "br_netfilter"}
	err = loadKernelModules(ctx, modules)
	if err != nil {
		zap.S().Infof("error loading kernel modules: %v", err)
		return err
	}
	zap.S().Infof("setting containerd sysctl params")
	err = setContainerdSysctlParams(ctx)
	if err != nil {
		zap.S().Infof("error setting containerd sysctl params: %v", err)
		return err
	}
	zap.S().Infof("generating config.toml")
	err = containerdConfig()
	if err != nil {
		zap.S().Infof("error generating config.toml: %v", err)
		return err
	}
	//TODO CHECK IF CONTAINERD IS ACTIVE AND STOP SERVICE
	err = cp.containerdRunPhase.Stop(context.Background(), config.Config{})
	if err != nil {
		zap.S().Infof("Error while stopping containerd: %v", err)
		//return fmt.Errorf("error stopping containerd: %v", err)
	}
	// now reload the dbus daemon and start the service

	// Reload dbus daemon to load new services
	zap.S().Infof("Reloading dbus")
	err = cp.conn.Reload()
	if err != nil {
		zap.S().Infof("error reloading dbus: %v", err)
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

func loadKernelModules(ctx context.Context, modules []string) error {

	err := os.MkdirAll("/etc/modules-load.d/", 0770)
	if err != nil {
		return err
	}

	for _, module := range modules {
		exitCode, err := cmdLine.RunCommand(ctx, nil, 0, "", "modprobe", module)
		if err != nil || exitCode != 0 {
			zap.S().Infof("command exited with exitcode:%v and err:%v", exitCode, err)
			return err
		}
	}
	file := "/etc/modules-load.d/" + "containerd.conf"
	f.WriteToFile(file, modules, false)
	if err != nil {
		return err
	}
	return nil
}
func setContainerdSysctlParams(ctx context.Context) error {
	err := os.MkdirAll("/etc/sysctl.d/", 0770)
	if err != nil {
		return err
	}
	file := "/etc/sysctl.d/" + "pf9-kubernetes-cri.conf"
	data := []string{
		"net.bridge.bridge-nf-call-iptables  = 1",
		"net.ipv4.ip_forward  = 1",
		"net.bridge.bridge-nf-call-ip6tables = 1"}
	err = f.WriteToFile(file, data, true)
	if err != nil {
		return err
	}
	exitCode, err := cmdLine.RunCommand(ctx, nil, 0, "", "sysctl", "--system")
	if err != nil || exitCode != 0 {
		zap.S().Infof("command exited with exitcode:%v and err:%v", exitCode, err)
		return err
	}
	return nil
}

func containerdUnit() error {
	err := os.MkdirAll("/usr/local/lib/systemd/system", 0770)
	if err != nil {
		return err
	}
	file := "/usr/local/lib/systemd/system/containerd.service"
	data :=
		`[Unit]
	Description=containerd container runtime
	Documentation=https://containerd.io
	After=network.target local-fs.target
	
	[Service]
	ExecStartPre=-/sbin/modprobe overlay
	ExecStart=/usr/local/bin/containerd
	
	Type=notify
	Delegate=yes
	KillMode=process
	Restart=always
	RestartSec=5
	# Having non-zero Limit*s causes performance problems due to accounting overhead
	# in the kernel. We recommend using cgroups to do container-local accounting.
	LimitNPROC=infinity
	LimitCORE=infinity
	LimitNOFILE=infinity
	# Comment TasksMax if your systemd version does not supports it.
	# Only systemd 226 and above support this version.
	TasksMax=infinity
	OOMScoreAdjust=-999
	
	[Install]
	WantedBy=multi-user.target`
	err = f.WriteToFile(file, data, false)
	if err != nil {
		return err
	}
	return nil
}

func containerdConfig() error {
	err := os.MkdirAll("/etc/containerd", 0770)
	if err != nil {
		return err
	}
	file := "/etc/containerd" + "config.toml"
	data :=
		`version = 2
		root = "/var/lib/containerd"
		state = "/run/containerd"
		plugin_dir = ""
		disabled_plugins = []
		required_plugins = []
		oom_score = 0
		[plugins]
		  [plugins."io.containerd.grpc.v1.cri"]
			[plugins."io.containerd.grpc.v1.cri".registry]
			  [plugins."io.containerd.grpc.v1.cri".registry.mirrors]
				[plugins."io.containerd.grpc.v1.cri".registry.mirrors."platform9.io"]
				  endpoint = ["https://dockermirror.platform9.io"]
				[plugins."io.containerd.grpc.v1.cri".registry.mirrors."docker.io"]
				  endpoint = ["https://dockermirror.platform9.io", "https://registry-1.docker.io"]
		  [plugins."io.containerd.grpc.v1.cri".containerd]
			snapshotter = "overlayfs"
		  [plugins."io.containerd.grpc.v1.cri".containerd.runtimes]
			[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc]
			  runtime_type = "io.containerd.runc.v2"`

	err = f.WriteToFile(file, data, false)
	if err != nil {
		return err
	}
	if constants.ContainerdCgroup == "systemd" {
		appendata :=
			`[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc.options]
            	SystemdCgroup = true`
		f.WriteToFile(file, appendata, true)
		if err != nil {
			return err
		}
	} else {
		appendata :=
			`[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc.options]
            	SystemdCgroup = false`
		f.WriteToFile(file, appendata, true)
		if err != nil {
			return err
		}
	}
	return nil
}
