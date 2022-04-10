package containerruntime

import (
	"context"
	"fmt"
	"os"

	"github.com/platform9/nodelet/nodelet/pkg/utils/config"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	sunpikev1alpha1 "github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike/v1alpha1"
	"go.uber.org/zap"

	containerutils "github.com/platform9/nodelet/nodelet/pkg/utils/container_runtime"
	"github.com/platform9/nodelet/nodelet/pkg/utils/fileio"
	"github.com/platform9/nodelet/nodelet/pkg/utils/phaseutils"
)

type LoadImagePhase struct {
	HostPhase   *sunpikev1alpha1.HostPhase
	log         *zap.SugaredLogger
	runtime     containerutils.Runtime
	fileUtility fileio.FileInterface
}

func NewLoadImagePhase() *LoadImagePhase {
	log := zap.S()
	return &LoadImagePhase{
		HostPhase: &sunpikev1alpha1.HostPhase{
			Name:  "Load user images to container runtime",
			Order: int32(constants.LoadImagePhaseOrder),
		},
		log:         log,
		runtime:     containerutils.New(),
		fileUtility: fileio.New(),
	}
}

func (l *LoadImagePhase) GetHostPhase() sunpikev1alpha1.HostPhase {
	return *l.HostPhase
}

func (l *LoadImagePhase) GetPhaseName() string {
	return l.HostPhase.Name
}

func (l *LoadImagePhase) GetOrder() int {
	return int(l.HostPhase.Order)
}

func (l *LoadImagePhase) Status(ctx context.Context, cfg config.Config) error {

	l.log.Infof("Running Status of phase: %s", l.HostPhase.Name)

	check, err := l.fileUtility.VerifyChecksum(cfg.UserImagesDir)
	if err != nil {
		l.log.Error(err.Error())
		phaseutils.SetHostStatus(l.HostPhase, constants.FailedState, err.Error())
		return err
	}
	if !check {
		err := l.runtime.LoadImagesFromDir(ctx, cfg.UserImagesDir, constants.K8sNamespace)
		if err != nil {
			l.log.Error(err.Error())
			phaseutils.SetHostStatus(l.HostPhase, constants.FailedState, err.Error())
			return err
		}
	}
	phaseutils.SetHostStatus(l.HostPhase, constants.RunningState, "")
	return nil
}
func (l *LoadImagePhase) Start(ctx context.Context, cfg config.Config) error {

	l.log.Infof("Running Start of phase: %s", l.HostPhase.Name)

	checksumFile := fmt.Sprintf("%s/checksum/sha256sums.txt", cfg.UserImagesDir)
	if _, err := os.Stat(checksumFile); os.IsNotExist(err) {
		err := l.fileUtility.GenerateChecksum(cfg.UserImagesDir)
		if err != nil {
			l.log.Error(err.Error())
			phaseutils.SetHostStatus(l.HostPhase, constants.FailedState, err.Error())
			return err
		}
	}
	err := l.runtime.LoadImagesFromDir(ctx, cfg.UserImagesDir, constants.K8sNamespace)
	if err != nil {
		l.log.Error(err.Error())
		phaseutils.SetHostStatus(l.HostPhase, constants.FailedState, err.Error())
		return err
	}
	phaseutils.SetHostStatus(l.HostPhase, constants.RunningState, "")
	return nil
}
func (l *LoadImagePhase) Stop(ctx context.Context, cfg config.Config) error {

	l.log.Infof("Running Stop of phase: %s", l.HostPhase.Name)

	phaseutils.SetHostStatus(l.HostPhase, constants.StoppedState, "")
	return nil
}
