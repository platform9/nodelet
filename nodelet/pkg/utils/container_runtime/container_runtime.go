package containerruntime

import (
	"context"

	"github.com/platform9/nodelet/nodelet/pkg/utils/config"
)

type Runtime interface {
	EnsureFreshContainerRunning(context.Context, config.Config, string, string) error
	EnsureContainerDestroyed(context.Context, config.Config, string) error
	EnsureContainerStoppedOrNonExistent(context.Context, config.Config, string) error
}

type ImageUtils interface {
	LoadImagesFromDir(context.Context, string, string) error
	LoadImagesFromFile(context.Context, string) error
}
