package containerruntime

import (
	"context"
	"syscall"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/oci"
	"github.com/platform9/nodelet/nodelet/pkg/utils/config"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	"go.uber.org/zap"
)

type ContainerdImpl struct {
	Service string
	Cli     string
	Crictl  string
	Socket  string
	Client  *containerd.Client
	log     *zap.SugaredLogger
}

var (
	containerdclient *containerd.Client
)

func NewContainerd() (Runtime, error) {

	containerdclient, err = newContainerdClient()
	if err != nil {
		return nil, err
	}
	return &ContainerdImpl{
		Service: "containerd",
		Cli:     "/opt/pf9/pf9-kube/bin/nerdctl",
		Crictl:  "/opt/pf9/pf9-kube/bin/crictl",
		Socket:  constants.ContainerdSocket,
		Client:  containerdclient,
		log:     zap.S(),
	}, nil
}

func newContainerdClient() (*containerd.Client, error) {

	containerdclient, err = containerd.New(constants.ContainerdSocket)
	if err != nil {
		zap.S().Info("failed to create containerd client")
		return nil, err
	}
	return containerdclient, nil
}

func (c *ContainerdImpl) EnsureFreshContainerRunning(ctx context.Context, cfg config.Config, containerName string, containerImage string) error {
	err := c.EnsureContainerDestroyed(ctx, cfg, containerName)
	if err != nil {
		return err
	}
	img, err := c.Client.GetImage(ctx, containerImage)
	if err != nil {
		return err
	}
	container, err := c.Client.NewContainer(
		ctx,
		containerName,
		containerd.WithNewSpec(oci.WithImageConfig(img)),
		containerd.WithImageName(containerImage),
	)
	if err != nil {
		return err
	}

	// create a task from the container
	task, err := container.NewTask(ctx, cio.NewCreator(cio.WithStdio))
	if err != nil {
		return err
	}

	err = task.Start(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (c *ContainerdImpl) EnsureContainerDestroyed(ctx context.Context, cfg config.Config, containerName string) error {

	containers, err := c.Client.Containers(ctx)
	if err != nil {
		return err
	}

	for _, container := range containers {
		if container.ID() == containerName {
			err = container.Delete(ctx, containerd.WithSnapshotCleanup)
			if err != nil {
				return err
			}
			break
		}
	}
	return nil
}

func (c *ContainerdImpl) EnsureContainerStoppedOrNonExistent(ctx context.Context, cfg config.Config, containerName string) error {

	c.log.Infof("Ensuring container %s is stopped or non-existent", containerName)
	containers, err := c.Client.Containers(ctx)
	if err != nil {
		return err
	}
	containerPresent := false
	for _, container := range containers {
		if container.ID() == containerName {
			task, err := container.Task(ctx, cio.NewAttach())
			if err != nil {
				return err
			}
			task.Kill(ctx, syscall.SIGTERM)
			if err != nil {
				return err
			}
			containerPresent = true
			return nil
		}
	}
	if !containerPresent {
		c.log.Info("container %s not present", containerName)
	}
	return nil
}
