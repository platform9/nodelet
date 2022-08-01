package containerruntime

import (
	"context"
	"embed"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"

	"github.com/coreos/go-systemd/dbus"
	"github.com/platform9/nodelet/nodelet/pkg/utils/command"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	"github.com/platform9/nodelet/nodelet/pkg/utils/embedutil"
	"github.com/platform9/nodelet/nodelet/pkg/utils/fileio"
	"github.com/platform9/nodelet/nodelet/pkg/utils/untar"
	"go.uber.org/zap"
)

var err error

//go:embed src/containerd/containerd.tar.gz
var containerdZip embed.FS

//go:embed src/runc_bin/runc
var runcBin embed.FS

//go:embed src/cni_plugins/cni-plugins.tgz
var cniPlugins embed.FS

type ContainerdInstall struct {
	embedFs *embedutil.EmbedFS
	cmd     command.CLI
	file    fileio.FileInterface
}

func NewContainerd() InstallRuntime {

	return &ContainerdInstall{
		embedFs: nil,
		cmd:     command.New(),
		file:    fileio.New(),
	}
}

func (ci *ContainerdInstall) EnsureContainerdInstalled(ctx context.Context) error {

	ci.embedFs = &embedutil.EmbedFS{
		Fs:   containerdZip,
		Root: "src/containerd",
	}
	var installedVersion string
	containerdInstalled := false

	exitCode, output, err := ci.cmd.RunCommandWithStdOut(ctx, nil, 0, "", "containerd", "--version")

	if err == nil && output != nil && exitCode == 0 {
		r := regexp.MustCompile(`v*\d.\d.\d`)
		installedVersion = r.FindString(output[0])
		if constants.ContainerdVersion == installedVersion {
			containerdInstalled = true
		}
	}

	if !containerdInstalled {

		zap.S().Infof("Extracting containerd zip to %s", constants.ContainerdBaseDir)
		err = ci.embedFs.Extract(constants.ContainerdBaseDir)
		if err != nil {
			zap.S().Infof("error extracting containerd zip: to baseDir %s,  %v", constants.ContainerdBaseDir, err)
			return fmt.Errorf("error extracting containerd zip: to baseDir %s,  %v", constants.ContainerdBaseDir, err)
		}

		// now extract the tar files
		zap.S().Infof("Untarring containerd zip to %s", constants.UsrLocalDir)

		matches, err := filepath.Glob(path.Join(constants.ContainerdBaseDir, "containerd*.tar.gz"))
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
		err = ci.GenerateContainerdUnit()
		if err != nil {
			zap.S().Infof("error generating containerd.service unit file: %v", err)
			return err
		}

		zap.S().Infof("generating config.toml")
		err = ci.GenerateContainerdConfig()
		if err != nil {
			zap.S().Infof("error generating config.toml: %v", err)
			return err
		}

		zap.S().Infof("setting containerd sysctl params")
		err = ci.SetContainerdSysctlParams(ctx)
		if err != nil {
			zap.S().Infof("error setting containerd sysctl params: %v", err)
			return err
		}

		zap.S().Infof("loading kernel modules")
		modules := []string{"overlay", "br_netfilter"}
		err = ci.LoadKernelModules(ctx, modules)
		if err != nil {
			zap.S().Infof("error loading kernel modules: %v", err)
			return err
		}

		zap.S().Infof("installing runc required for containerd")
		err = ci.EnsureRuncInstalled()
		if err != nil {
			zap.S().Infof("Error installing runc: %v", err)
			return err
		}

		zap.S().Infof("installing cni plugins required for containerd")
		err = ci.EnsureCNIPluginsInstalled()
		if err != nil {
			zap.S().Infof("Error installing cni-plugins: %v", err)
			return err
		}

		conn, err := dbus.NewSystemConnection()
		if err != nil {
			zap.S().Errorf("error connecting to dbus: %v", err)
		}
		defer conn.Close()

		// Reload dbus daemon to load new containerd service
		zap.S().Infof("Reloading dbus")
		err = conn.Reload()
		if err != nil {
			zap.S().Infof("error reloading dbus: %v", err)
			return fmt.Errorf("error reloading dbus: %v", err)
		}

		// systemctl enable --now containerd
		zap.S().Infof("Enabling containerd")
		unitfiles := []string{constants.ContainerdUnitFile}

		/*

			TODO: crosscheck 2nd and 3rd arg of below function
			2nd arg: runtime: controls whether the unit shall be enabled for runtime only (true, /run), or persistently (false, /etc)
			3rd arg: force: controls whether symlinks pointing to other units shall be replaced if necessary

		*/

		enablement, changes, err := conn.EnableUnitFiles(unitfiles, false, false)
		if err != nil {
			zap.S().Infof("error enabling containerd: %v", err)
			return fmt.Errorf("error enabling containerd: %v", err)
		}

		//TODO: check how to use above enablement boolean.
		//enablement : The boolean signals whether the unit files contained any enablement information (i.e. an [Install]) section

		zap.S().Info("just printing this to avoid var unused error. will remove it %v", enablement)

		if len(changes) > 0 {
			zap.S().Infof("%sed %s to %s", changes[0].Type, changes[0].Filename, changes[0].Destination)
		}

		zap.S().Infof("Restarting containerd")
		jobId, err := conn.RestartUnit("containerd.service", "replace", nil)
		if err != nil {
			zap.S().Infof("error reloading dbus: %v", err)
			return err
		}
		zap.S().Infof("containerd started with job id: %v", jobId)

	}
	return nil
}

func (ci *ContainerdInstall) EnsureRuncInstalled() error {

	ci.embedFs = &embedutil.EmbedFS{
		Fs:   runcBin,
		Root: "src/runc_bin",
	}

	zap.S().Infof("Extracting Runc to %s", constants.UsrLocalSbinDir)
	err = ci.embedFs.Extract(constants.UsrLocalSbinDir)
	if err != nil {
		zap.S().Infof("error extracting containerd zip: to baseDir %s,  %v", constants.UsrLocalSbinDir, err)
		return fmt.Errorf("error extracting containerd zip: to baseDir %s,  %v", constants.UsrLocalSbinDir, err)
	}

	zap.S().Infof("giving execute permmission to runc")
	err = os.Chmod(constants.RuncBin, 0755)
	if err != nil {
		zap.S().Infof("error giving exec perm to  %s,  %v", constants.RuncBin, err)
		return fmt.Errorf("error giving exec perm to  %s,  %v", constants.RuncBin, err)
	}

	return nil
}

func (ci *ContainerdInstall) EnsureCNIPluginsInstalled() error {

	ci.embedFs = &embedutil.EmbedFS{
		Fs:   cniPlugins,
		Root: "src/cni_plugins",
	}

	zap.S().Infof("Extracting cni-plugins zip to %s", constants.ContainerdBaseDir)
	err = ci.embedFs.Extract(constants.ContainerdBaseDir)
	if err != nil {
		zap.S().Infof("error extracting cni-plugins zip: to baseDir %s,  %v", constants.ContainerdBaseDir, err)
		return fmt.Errorf("error extracting cni-plugins zip: to baseDir %s,  %v", constants.ContainerdBaseDir, err)
	}

	// now extract the tar files
	zap.S().Infof("Untarring cni-plugins zip to %s", constants.CniDir)

	matches, err := filepath.Glob(path.Join(constants.ContainerdBaseDir, "cni-plugins*.tgz"))
	if err != nil {
		zap.S().Infof("error finding cni-plugins tar files: %v", err)
		return fmt.Errorf("error finding cni-plugins tar files: %v", err)
	}

	for _, match := range matches {
		zap.S().Infof("Untarring %s", match)
		err = untar.Extract(match, constants.CniDir)
		if err != nil {
			zap.S().Infof("error untarring cni-plugins tar file: %v", err)
			return fmt.Errorf("error untarring cni-plugins tar file: %v", err)
		}
	}

	return nil
}

func (ci *ContainerdInstall) LoadKernelModules(ctx context.Context, modules []string) error {

	err := os.MkdirAll("/etc/modules-load.d/", 0770)
	if err != nil {
		return err
	}

	for _, module := range modules {
		exitCode, err := ci.cmd.RunCommand(ctx, nil, 0, "", "modprobe", module)
		if err != nil || exitCode != 0 {
			zap.S().Infof("command exited with exitcode:%v and err:%v", exitCode, err)
			return err
		}
	}
	file := "/etc/modules-load.d/" + "containerd.conf"
	ci.file.WriteToFile(file, modules, false)
	if err != nil {
		return err
	}
	return nil
}

func (ci *ContainerdInstall) SetContainerdSysctlParams(ctx context.Context) error {
	err := os.MkdirAll("/etc/sysctl.d/", 0770)
	if err != nil {
		return err
	}
	file := "/etc/sysctl.d/" + "pf9-kubernetes-cri.conf"
	data := []string{
		"net.bridge.bridge-nf-call-iptables  = 1",
		"net.ipv4.ip_forward  = 1",
		"net.bridge.bridge-nf-call-ip6tables = 1"}
	err = ci.file.WriteToFile(file, data, true)
	if err != nil {
		return err
	}
	exitCode, err := ci.cmd.RunCommand(ctx, nil, 0, "", "sysctl", "--system")
	if err != nil || exitCode != 0 {
		zap.S().Infof("command exited with exitcode:%v and err:%v", exitCode, err)
		return err
	}
	return nil
}

func (ci *ContainerdInstall) GenerateContainerdUnit() error {

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

	err = ci.file.WriteToFile(file, data, false)
	if err != nil {
		return err
	}
	return nil
}

func (ci *ContainerdInstall) GenerateContainerdConfig() error {
	err := os.MkdirAll("/etc/containerd", 0770)
	if err != nil {
		return err
	}
	file := "/etc/containerd/" + "config.toml"
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

	err = ci.file.WriteToFile(file, data, false)
	if err != nil {
		return err
	}
	if constants.ContainerdCgroup == "systemd" {
		appendata :=
			`[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc.options]
            	SystemdCgroup = true`
		ci.file.WriteToFile(file, appendata, true)
		if err != nil {
			return err
		}
	} else {
		appendata :=
			`[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc.options]
            	SystemdCgroup = false`
		ci.file.WriteToFile(file, appendata, true)
		if err != nil {
			return err
		}
	}
	return nil
}
