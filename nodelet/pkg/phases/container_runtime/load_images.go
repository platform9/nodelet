package containerruntime

import (
	"context"
	"os"

	"github.com/platform9/nodelet/nodelet/pkg/utils/config"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	sunpikev1alpha1 "github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike/v1alpha1"
	"go.uber.org/zap"

	"github.com/platform9/nodelet/nodelet/pkg/utils/phaseutils"
)

type LoadImagePhase struct {
	HostPhase *sunpikev1alpha1.HostPhase
	log       *zap.SugaredLogger
}

func NewLoadImagePhase() *LoadImagePhase {
	log := zap.S()
	return &LoadImagePhase{
		HostPhase: &sunpikev1alpha1.HostPhase{
			Name:  "Load user images to container runtime",
			Order: int32(constants.LoadImagePhaseOrder),
		},
		log: log,
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

	check, err := phaseutils.VerifyChecksum(constants.UserImagesDir)
	if err != nil {
		return err
	}
	if !check {
		err := phaseutils.LoadImagesFromDir(ctx, constants.UserImagesDir)
		if err != nil {
			return err
		}
	}
	return nil
}
func (d *LoadImagePhase) Start(ctx context.Context, cfg config.Config) error {

	if _, err := os.Stat(constants.ChecksumDir); os.IsNotExist(err) {
		err := phaseutils.GenerateChecksum(constants.UserImagesDir)
		if err != nil {
			return err
		}
	}
	err := phaseutils.LoadImagesFromDir(ctx, constants.UserImagesDir)
	if err != nil {
		return err
	}
	return nil
}
func (d *LoadImagePhase) Stop(ctx context.Context, cfg config.Config) error {
	return nil
}
