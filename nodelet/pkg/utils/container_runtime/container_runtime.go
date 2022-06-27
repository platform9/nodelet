package containerruntime

import (
	"context"
)

type Runtime interface {
	EnsureFreshContainerRunning(context.Context, string, string) error
	EnsureContainerDestroyed(context.Context, string, string) error
	EnsureContainerStoppedOrNonExistent(context.Context, string) error
}

type ImageUtils interface {
	LoadImagesFromDir(context.Context, string, string) error
	LoadImagesFromFile(context.Context, string) error
}
