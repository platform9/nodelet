package containerruntime

import (
	"context"

	"github.com/containerd/containerd"
)

type ContainerUtils interface {
	EnsureFreshContainerRunning(ctx context.Context, namespace string, containerName string, containerImage string, runOpts RunOpts, cmdArgs []string) error
	EnsureContainerDestroyed(context.Context, string, string) error
	EnsureContainersDestroyed(ctx context.Context, containers []containerd.Container, timeoutStr string) error
	EnsureContainerStoppedOrNonExistent(context.Context, string) error
	GetContainersInNamespace(ctx context.Context, namespace string) ([]containerd.Container, error)
	GetContainerWithGivenName(ctx context.Context, containerName string) (containerd.Container, error)
	DestroyContainersInNamespace(ctx context.Context, namespace string) error
	DestroyContainersInNamespacesList(ctx context.Context, namespaces []string) error
	CreateContainer(ctx context.Context, containerName string, containerImage string, runOpts RunOpts, cmdArgs []string) (containerd.Container, error)
	RemoveContainer(ctx context.Context, container containerd.Container, force bool) error
	StopContainer(ctx context.Context, container containerd.Container, timeoutStr string) error
	CloseClient()
}

type ImageUtils interface {
	LoadImagesFromDir(context.Context, string, string) error
	LoadImagesFromFile(context.Context, string) error
}

// type InstallRuntime interface {
// 	EnsureContainerdInstalled(ctx context.Context) error
// 	EnsureRuncInstalled() error
// 	EnsureCNIPluginsInstalled() error
// 	LoadKernelModules(ctx context.Context, modules []string) error
// 	SetContainerdSysctlParams(ctx context.Context) error
// 	GenerateContainerdUnit() error
// 	GenerateContainerdConfig() error
// }
