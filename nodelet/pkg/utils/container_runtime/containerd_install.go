package containerruntime

import (
	"context"
	"embed"
	"os"
	"path"
	"path/filepath"
	"regexp"

	"github.com/coreos/go-systemd/dbus"
	"github.com/pkg/errors"
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
	log     *zap.SugaredLogger
}

func NewContainerd() InstallRuntime {

	return &ContainerdInstall{
		embedFs: nil,
		cmd:     command.New(),
		file:    fileio.New(),
		log:     zap.S(),
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
		//check if installed version is same as required version
		if constants.ContainerdVersion == installedVersion {
			containerdInstalled = true
		}
	}

	if !containerdInstalled {

		ci.log.Infof("Extracting containerd zip to %s", constants.ContainerdBaseDir)
		err = ci.embedFs.Extract(constants.ContainerdBaseDir)
		if err != nil {
			return errors.Wrap(err, "error extracting containerd zip: to baseDir")
		}

		ci.log.Infof("Untarring containerd zip to %s", constants.UsrLocalDir)

		matches, err := filepath.Glob(path.Join(constants.ContainerdBaseDir, "containerd*.tar.gz"))
		if err != nil {
			return errors.Wrap(err, "error finding containerd tar files")
		}

		for _, match := range matches {
			ci.log.Infof("Untarring %s", match)
			err = untar.Extract(match, constants.UsrLocalDir)
			if err != nil {
				errors.Wrap(err, "error untarring containerd tar file")
			}
		}

		ci.log.Infof("Generating containerd.service unit file")
		err = ci.GenerateContainerdUnit()
		if err != nil {
			return errors.Wrap(err, "error generating containerd.service unit file")
		}

		ci.log.Infof("Generating config.toml")
		err = ci.GenerateContainerdConfig()
		if err != nil {
			return errors.Wrap(err, "error generating config.toml config file")
		}

		ci.log.Infof("Setting containerd sysctl params")
		err = ci.SetContainerdSysctlParams(ctx)
		if err != nil {
			return errors.Wrap(err, "error setting containerd sysctl params")
		}

		ci.log.Infof("Loading kernel modules")
		modules := []string{"overlay", "br_netfilter"}
		err = ci.LoadKernelModules(ctx, modules)
		if err != nil {
			return errors.Wrap(err, "error loading kernel modules required for containerd")
		}

		ci.log.Infof("Installing runc required for containerd")
		err = ci.EnsureRuncInstalled()
		if err != nil {
			return errors.Wrap(err, "error installing runc")
		}

		ci.log.Infof("Installing cni plugins required for containerd")
		err = ci.EnsureCNIPluginsInstalled()
		if err != nil {
			return errors.Wrap(err, "error installing cni-plugins")
		}

		conn, err := dbus.NewSystemConnection()
		if err != nil {
			return errors.Wrap(err, "error connecting to dbus")
		}
		defer conn.Close()

		// Reload dbus daemon to load new containerd service
		ci.log.Infof("Reloading dbus")
		err = conn.Reload()
		if err != nil {
			return errors.Wrap(err, "error reloading dbus")
		}

		// systemctl enable --now containerd
		ci.log.Infof("Enabling containerd")
		unitfiles := []string{constants.ContainerdUnitFile}

		/*

			TODO: crosscheck 2nd and 3rd arg of below function
			2nd arg: runtime: controls whether the unit shall be enabled for runtime only (true, /run), or persistently (false, /etc)
			3rd arg: force: controls whether symlinks pointing to other units shall be replaced if necessary

		*/

		_, changes, err := conn.EnableUnitFiles(unitfiles, false, true)
		if err != nil {
			return errors.Wrap(err, "error enabling containerd")
		}

		if len(changes) > 0 {
			ci.log.Infof("%sed %s to %s", changes[0].Type, changes[0].Filename, changes[0].Destination)
		}

	}
	return nil
}

func (ci *ContainerdInstall) EnsureRuncInstalled() error {

	ci.embedFs = &embedutil.EmbedFS{
		Fs:   runcBin,
		Root: "src/runc_bin",
	}

	ci.log.Infof("Extracting Runc to %s", constants.UsrLocalSbinDir)
	err = ci.embedFs.Extract(constants.UsrLocalSbinDir)
	if err != nil {
		return errors.Wrapf(err, "error extracting containerd zip: to baseDir %s", constants.UsrLocalSbinDir)
	}

	ci.log.Infof("Giving execute permmission to runc")
	err = os.Chmod(constants.RuncBin, 0775)
	if err != nil {
		return errors.Wrapf(err, "error giving exec perm to  %s", constants.RuncBin)
	}

	return nil
}

func (ci *ContainerdInstall) EnsureCNIPluginsInstalled() error {

	ci.embedFs = &embedutil.EmbedFS{
		Fs:   cniPlugins,
		Root: "src/cni_plugins",
	}

	ci.log.Infof("Extracting cni-plugins zip to %s", constants.ContainerdBaseDir)
	err = ci.embedFs.Extract(constants.ContainerdBaseDir)
	if err != nil {
		return errors.Wrapf(err, "error extracting cni-plugins zip: to baseDir %s", constants.ContainerdBaseDir)
	}

	ci.log.Infof("Untarring cni-plugins zip to %s", constants.CniDir)

	matches, err := filepath.Glob(path.Join(constants.ContainerdBaseDir, "cni-plugins*.tgz"))
	if err != nil {
		return errors.Wrapf(err, "error finding cni-plugins tar files")
	}

	for _, match := range matches {
		ci.log.Infof("Untarring %s", match)
		err = untar.Extract(match, constants.CniDir)
		if err != nil {
			return errors.Wrap(err, "error untarring cni-plugins tar file")
		}
	}

	return nil
}

func (ci *ContainerdInstall) LoadKernelModules(ctx context.Context, modules []string) error {

	err := os.Mkdir(constants.EtcModulesDir, 0770)
	if err != nil {
		return errors.Wrap(err, "error creating etc modules directory")
	}

	for _, module := range modules {
		exitCode, err := ci.cmd.RunCommand(ctx, nil, 0, "", "modprobe", module)
		if err != nil || exitCode != 0 {
			ci.log.Warnf("modprobe command exited with exitcode: %d :%v", exitCode, err)
			//TODO: need to return error here or only warn log is fine? same to check in other places too.
			return nil
		}
	}

	ci.file.WriteToFile(constants.EtcModulesContainerdConfFile, modules, false)
	if err != nil {
		ci.log.Warnf("failed to write kernel modules to file: %s :%v", constants.EtcModulesContainerdConfFile, err)
		return nil
	}

	return nil
}

func (ci *ContainerdInstall) SetContainerdSysctlParams(ctx context.Context) error {

	data := []string{
		"net.bridge.bridge-nf-call-iptables  = 1",
		"net.ipv4.ip_forward  = 1",
		"net.bridge.bridge-nf-call-ip6tables = 1"}

	err = ci.file.WriteToFile(constants.SysctlPf9CriConfFile, data, true)
	if err != nil {
		return err
	}

	exitCode, err := ci.cmd.RunCommand(ctx, nil, 0, "", "sysctl", "--system")
	if err != nil || exitCode != 0 {
		ci.log.Warnf("sysctl --system command exited with code: %d", exitCode)
		return nil
	}

	return nil
}

func (ci *ContainerdInstall) GenerateContainerdUnit() error {

	err := os.MkdirAll(constants.UsrLocalLibSytemdDir, 0770)
	if err != nil {
		return errors.Wrap(err, "error creating local lib systemd directory")
	}

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

	err = ci.file.WriteToFile(constants.ContainerdUnitFile, data, false)
	if err != nil {
		return errors.Wrapf(err, "failed to write containerd unit file:%s", constants.ContainerdUnitFile)
	}

	return nil

}

func (ci *ContainerdInstall) GenerateContainerdConfig() error {

	err := os.Mkdir(constants.EtcContainerdDir, 0770)
	if err != nil {
		return errors.Wrapf(err, "failed to write containerd config file:%s", constants.ContainerdConfigFile)
	}

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

	err = ci.file.WriteToFile(constants.ContainerdConfigFile, data, false)
	if err != nil {
		return errors.Wrapf(err, "failed to write containerd config file:%s", constants.ContainerdConfigFile)
	}

	err = ci.file.WriteToFile(constants.ContainerdConfigFile, "\n", true)
	if err != nil {
		return errors.Wrapf(err, "failed to write containerd config file:%s", constants.ContainerdConfigFile)
	}

	if constants.ContainerdCgroup == "systemd" {

		appendata :=
			`[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc.options]
            	SystemdCgroup = true`

		ci.file.WriteToFile(constants.ContainerdConfigFile, appendata, true)
		if err != nil {
			return errors.Wrapf(err, "failed to write containerd config file:%s", constants.ContainerdConfigFile)
		}

	} else {

		appendata :=
			`[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc.options]
            	SystemdCgroup = false`

		ci.file.WriteToFile(constants.ContainerdConfigFile, appendata, true)
		if err != nil {
			return errors.Wrapf(err, "failed to write containerd config file:%s", constants.ContainerdConfigFile)
		}

	}

	return nil
}
