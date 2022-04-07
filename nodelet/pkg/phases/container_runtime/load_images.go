package containerruntime

import (
	"context"
	"os"

	"github.com/platform9/nodelet/nodelet/pkg/utils/config"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	sunpikev1alpha1 "github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike/v1alpha1"
	"go.uber.org/zap"

	containerutils "github.com/platform9/nodelet/nodelet/pkg/utils/container_runtime_utils"
	"github.com/platform9/nodelet/nodelet/pkg/utils/phaseutils"
)

type LoadImagePhase struct {
	HostPhase   *sunpikev1alpha1.HostPhase
	log         *zap.SugaredLogger
	runtimeUtil containerutils.Runtime
}

func NewLoadImagePhase() *LoadImagePhase {
	log := zap.S()
	return &LoadImagePhase{
		HostPhase: &sunpikev1alpha1.HostPhase{
			Name:  "Load user images to container runtime",
			Order: int32(constants.LoadImagePhaseOrder),
		},
		log:         log,
		runtimeUtil: containerutils.New(),
	}
}

func (d *LoadImagePhase) GetHostPhase() sunpikev1alpha1.HostPhase {
	return *d.HostPhase
}

func (d *LoadImagePhase) GetPhaseName() string {
	return d.HostPhase.Name
}

func (d *LoadImagePhase) GetOrder() int {
	return int(d.HostPhase.Order)
}

func (d *LoadImagePhase) Status(ctx context.Context, cfg config.Config) error {

	check, err := d.runtimeUtil.VerifyChecksum(constants.UserImagesDir)
	if err != nil {
		d.log.Error(err.Error())
		phaseutils.SetHostStatus(d.HostPhase, constants.FailedState, err.Error())
		return err
	}
	if !check {
		err := d.runtimeUtil.LoadImagesFromDir(ctx, constants.UserImagesDir, "k8s.io")
		if err != nil {
			d.log.Error(err.Error())
			phaseutils.SetHostStatus(d.HostPhase, constants.FailedState, err.Error())
			return err
		}
	}
	phaseutils.SetHostStatus(d.HostPhase, constants.RunningState, "")
	return nil
}
func (d *LoadImagePhase) Start(ctx context.Context, cfg config.Config) error {

	if _, err := os.Stat(constants.ChecksumFile); os.IsNotExist(err) {
		err := d.runtimeUtil.GenerateChecksum(constants.UserImagesDir)
		if err != nil {
			d.log.Error(err.Error())
			phaseutils.SetHostStatus(d.HostPhase, constants.FailedState, err.Error())
			return err
		}
	}
	err := d.runtimeUtil.LoadImagesFromDir(ctx, constants.UserImagesDir, "k8s.io")
	if err != nil {
		d.log.Error(err.Error())
		phaseutils.SetHostStatus(d.HostPhase, constants.FailedState, err.Error())
		return err
	}
	phaseutils.SetHostStatus(d.HostPhase, constants.RunningState, "")
	return nil
}
func (d *LoadImagePhase) Stop(ctx context.Context, cfg config.Config) error {
	return nil
}
