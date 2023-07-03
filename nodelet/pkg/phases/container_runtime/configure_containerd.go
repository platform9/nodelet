package containerruntime

import (
	"context"
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"github.com/platform9/nodelet/nodelet/pkg/utils/command"
	"github.com/platform9/nodelet/nodelet/pkg/utils/config"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	"github.com/platform9/nodelet/nodelet/pkg/utils/fileio"
	"github.com/platform9/nodelet/nodelet/pkg/utils/phaseutils"
	sunpikev1alpha1 "github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike/v1alpha1"
	"go.uber.org/zap"
)

type ContainerdConfigPhase struct {
	hostPhase *sunpikev1alpha1.HostPhase
	log       *zap.SugaredLogger
	cmd       command.CLI
}

func NewContainerdConfigPhase() *ContainerdConfigPhase {

	containerdConfigPhase := &ContainerdConfigPhase{
		hostPhase: &sunpikev1alpha1.HostPhase{
			Name:  "Configure Container Runtime",
			Order: int32(constants.ConfigureRuntimePhaseOrder),
		},
		log: zap.S(),
		cmd: command.New(),
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

// TODO: check PF9_MANAGED_DOCKER var
func (cp *ContainerdConfigPhase) Start(ctx context.Context, cfg config.Config) error {

	cp.log.Infof("Running Start of phase: %s", cp.hostPhase.Name)

	exitCode, output, err := cp.cmd.RunCommandWithStdOut(ctx, nil, 0, "", constants.ContainerdBinPath, "--version")
	if err != nil || exitCode != 0 {
		cp.log.Errorf("containerd not installed command containerd --version exited with code:%v", exitCode)
		phaseutils.SetHostStatus(cp.hostPhase, constants.FailedState, fmt.Sprintf("containerd not installed command containerd --version exited with code:%v", exitCode))
		return errors.Wrapf(err, "containerd not installed command containerd --version exited with code:%v", exitCode)
	}

	if output != nil {
		r := regexp.MustCompile(`v*\d.\d.\d`)
		installedVersion := r.FindString(output[0])
		cp.log.Infof("installed containerd version:%v", installedVersion)
	}

	file := fileio.New()
	b, err := ioutil.ReadFile(constants.ContainerdConfigFile)
	if err != nil {
		phaseutils.SetHostStatus(cp.hostPhase, constants.FailedState, fmt.Sprintf("couldn't read containerd config file:%s :%v", constants.ContainerdConfigFile, err))
		return errors.Wrapf(err, "couldn't read containerd config file:%s", constants.ContainerdConfigFile)
	}
	fileContent := string(b)

	// TODO: check if we can use this:
	// https://github.com/containerd/containerd/blob/84ec0796f82e5b3cf875942f43b612862eb3cf15/services/server/config/config.go#L188

	// checking containerd Cgroup configured
	if !strings.Contains(fileContent, constants.CgroupSystemd) {
		appendata := "\n\t[plugins.\"io.containerd.grpc.v1.cri\".containerd.runtimes.runc.options]\n\t\tSystemdCgroup = false"
		if constants.ContainerdCgroup == "systemd" {
			appendata = "\n\t[plugins.\"io.containerd.grpc.v1.cri\".containerd.runtimes.runc.options]\n\t\tSystemdCgroup = true"
		}
		err = file.WriteToFile(constants.ContainerdConfigFile, appendata, true)
		if err != nil {
			phaseutils.SetHostStatus(cp.hostPhase, constants.FailedState, fmt.Sprintf("couldn't write to containerd config file:%s :%v", constants.ContainerdConfigFile, err))
			return errors.Wrapf(err, "couldn't write to containerd config file:%s", constants.ContainerdConfigFile)
		}
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
