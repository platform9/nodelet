package containerruntime

import (
	"context"
	"fmt"
	"syscall"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/oci"
	"github.com/pkg/errors"
	"github.com/platform9/nodelet/nodelet/pkg/utils/config"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	"go.uber.org/zap"
)

type Containerd struct {
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
	return &Containerd{
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
		return nil, errors.Wrap(err, "failed to create containerd client")
	}
	return containerdclient, nil
}

func (c *Containerd) EnsureFreshContainerRunning(ctx context.Context, cfg config.Config, containerName string, containerImage string) error {
	err := c.EnsureContainerDestroyed(ctx, cfg, containerImage)
	if err != nil {
		return err
	}
	img, err := c.Client.GetImage(ctx, containerImage)
	if err != nil {
		return errors.Wrapf(err, "couldn't get %s iamge from client\n", containerImage)
	}
	container, err := c.Client.NewContainer(
		ctx,
		containerName,
		containerd.WithNewSnapshot(containerName, img),
		containerd.WithImageName(containerImage),
		containerd.WithNewSpec(oci.WithImageConfig(img)),
	)
	if err != nil {
		return errors.Wrapf(err, "couldn't create container %s\n", containerName)
	}

	// create a task from the container
	task, err := container.NewTask(ctx, cio.NewCreator())
	if err != nil {
		return errors.Wrapf(err, "couldn't create task from container %s\n", containerName)
	}

	exitStatusC, err := task.Wait(ctx)
	if err != nil {
		return err
	}
	err = task.Start(ctx)
	if err != nil {
		return errors.Wrapf(err, "couldn't start task from container %s\n", containerName)
	}

	status := <-exitStatusC
	code, _, err := status.Result()
	if err != nil {
		return err
	}
	if code != 0 {
		return fmt.Errorf("task creation process exited with status code: %d", code)
	}

	return nil
}

func (c *Containerd) EnsureContainerDestroyed(ctx context.Context, cfg config.Config, containerImage string) error {

	containers, err := c.Client.Containers(ctx)
	if err != nil {
		return errors.Wrap(err, "couldn't get containers from containerd client\n")
	}

	for _, container := range containers {
		img, err := container.Image(ctx)
		if err != nil {
			return err
		}
		if img.Name() == containerImage {

			defer container.Delete(ctx, containerd.WithSnapshotCleanup)

			task, err := container.Task(ctx, cio.Load)
			if err != nil {
				return errors.Wrapf(err, "couldn't get task from container %s\n", containerImage)
			}
			defer task.Delete(ctx)

			exitStatusC, err := task.Wait(ctx)
			if err != nil {
				return err
			}

			// kill the process and get the exit status
			if err := task.Kill(ctx, syscall.SIGTERM); err != nil {
				return err
			}

			status := <-exitStatusC
			code, _, err := status.Result()
			if err != nil {
				return err
			}
			if code != 0 {
				return fmt.Errorf("task deletion exited with status code: %d", code)
			}
			return nil
		}
	}
	c.log.Infof("container not present: %s\n", containerImage)
	return nil
}

func (c *Containerd) EnsureContainerStoppedOrNonExistent(ctx context.Context, cfg config.Config, containerImage string) error {

	c.log.Info("Ensuring container %s is stopped or non-existent", containerImage)
	containers, err := c.Client.Containers(ctx)
	if err != nil {
		return errors.Wrap(err, "couldn't get containers from containerd client\n")
	}
	for _, container := range containers {
		img, err := container.Image(ctx)
		if err != nil {
			return err
		}
		if img.Name() == containerImage {
			task, err := container.Task(ctx, cio.Load)
			if err != nil {
				return errors.Wrapf(err, "couldn't get task from container %s\n", containerImage)
			}

			err = task.Pause(ctx)
			if err != nil {
				return err
			}
			return nil
		}
	}
	c.log.Infof("container %s not present\n", containerImage)

	return nil
}
