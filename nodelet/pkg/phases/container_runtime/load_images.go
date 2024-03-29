package containerruntime

import (
	"context"
	"fmt"
	"os"

	"github.com/platform9/nodelet/nodelet/pkg/utils/config"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	runtime "github.com/platform9/nodelet/nodelet/pkg/utils/container_runtime"
	"github.com/platform9/nodelet/nodelet/pkg/utils/fileio"
	"github.com/platform9/nodelet/nodelet/pkg/utils/phaseutils"
	sunpikev1alpha1 "github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike/v1alpha1"
	"go.uber.org/zap"
)

type LoadImagePhase struct {
	HostPhase  *sunpikev1alpha1.HostPhase
	log        *zap.SugaredLogger
	imageUtils runtime.ImageUtils
	fileUtils  fileio.FileInterface
}

func NewLoadImagePhase() *LoadImagePhase {
	log := zap.S()
	return &LoadImagePhase{
		HostPhase: &sunpikev1alpha1.HostPhase{
			Name:  "Load user images to container runtime",
			Order: int32(constants.LoadImagePhaseOrder),
		},
		log:        log,
		imageUtils: runtime.NewImageUtil(),
		fileUtils:  fileio.New(),
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

	if _, err := os.Stat(cfg.UserImagesDir); os.IsNotExist(err) {
		l.log.Warnf("User images Directory:%s is not present", cfg.UserImagesDir)
		phaseutils.SetHostStatus(l.HostPhase, constants.RunningState, "")
		return nil
	}

	check, err := l.fileUtils.VerifyChecksum(cfg.UserImagesDir)
	if err != nil {
		l.log.Error(err.Error())
		phaseutils.SetHostStatus(l.HostPhase, constants.FailedState, err.Error())
		return err
	}
	if !check {
		err := l.imageUtils.LoadImagesFromDir(ctx, cfg.UserImagesDir, constants.K8sNamespace)
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

	if _, err := os.Stat(constants.UserImagesDir); os.IsNotExist(err) {

		if err := os.MkdirAll(constants.UserImagesDir, os.ModePerm); err != nil {
			l.log.Error(err.Error())
			phaseutils.SetHostStatus(l.HostPhase, constants.FailedState, err.Error())
			return err
		}
	}

	if _, err := os.Stat(cfg.UserImagesDir); os.IsNotExist(err) {
		l.log.Warnf("User images Directory:%s is not present, so couldn't load images", cfg.UserImagesDir)
		phaseutils.SetHostStatus(l.HostPhase, constants.RunningState, "")
		return nil
	}

	checksumFile := fmt.Sprintf("%s/checksum/sha256sums.txt", cfg.UserImagesDir)
	_, err := os.Stat(checksumFile)
	if err == nil {
		l.log.Infof("Previous image bundle checksum exists: %s\n", checksumFile)
		match, err := l.fileUtils.VerifyChecksum(cfg.UserImagesDir)
		if err != nil {
			l.log.Error(err.Error())
			phaseutils.SetHostStatus(l.HostPhase, constants.FailedState, err.Error())
			return err
		}
		if match {
			l.log.Infof("Checksum matches, not reloading image bundle")
			phaseutils.SetHostStatus(l.HostPhase, constants.RunningState, "")
			return nil
		}
	} else if os.IsNotExist(err) {
		l.log.Infof("Checksum file does not exist, loading image bundle")
		err := l.fileUtils.GenerateChecksum(cfg.UserImagesDir)
		if err != nil {
			l.log.Error(err.Error())
			phaseutils.SetHostStatus(l.HostPhase, constants.FailedState, err.Error())
			return err
		}
	} else {
		l.log.Errorf("Error opening checksum file %s: %s", checksumFile, err)
		phaseutils.SetHostStatus(l.HostPhase, constants.FailedState, err.Error())
		return err
	}

	err = l.imageUtils.LoadImagesFromDir(ctx, cfg.UserImagesDir, constants.K8sNamespace)
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
