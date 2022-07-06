package containerruntime

import (
	"context"
	"fmt"
	"syscall"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/oci"
	"github.com/pkg/errors"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	"go.uber.org/zap"
)

type Containerd struct {
	Service string
	Socket  string
	Client  *containerd.Client
	log     *zap.SugaredLogger
}

const timeOut = "10s"

func NewContainerd() (ContainerUtils, error) {

	containerdclient, err := NewContainerdClient()
	if err != nil {
		return nil, err
	}
	return &Containerd{
		Service: constants.RuntimeContainerd,
		Socket:  constants.ContainerdSocket,
		Client:  containerdclient,
		log:     zap.S(),
	}, nil
}

func NewContainerdClient() (*containerd.Client, error) {

	containerdclient, err := containerd.New(constants.ContainerdSocket)
	if err != nil {
		zap.S().Info("failed to create containerd client")
		return nil, errors.Wrap(err, "failed to create containerd client")
	}
	return containerdclient, nil
}

func (c *Containerd) EnsureFreshContainerRunning(ctx context.Context, containerName string, containerImage string) error {

	err := c.EnsureContainerDestroyed(ctx, containerName, timeOut)
	if err != nil {
		return err
	}
	container, err := c.CreateContainer(ctx, containerName, containerImage)
	if err != nil {
		return errors.Wrapf(err, "failed to create container:%s", containerName)
	}
	// create a task from the container
	task, err := container.NewTask(ctx, cio.NewCreator())
	if err != nil {
		return err
	}

	// make sure we wait before calling start
	exitStatusC, err := task.Wait(ctx)
	if err != nil {
		return err
	}

	if err := task.Start(ctx); err != nil {
		return err
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

func (c *Containerd) EnsureContainerDestroyed(ctx context.Context, containerName string, timeoutStr string) error {

	container, err := c.GetContainerWithGivenName(ctx, containerName)
	if err != nil {
		return err
	}
	if container == nil {
		c.log.Infof("container not present: %s.\n", containerName)
		return nil
	}
	err = c.StopContainer(ctx, container, timeoutStr)
	if err != nil {
		return errors.Wrapf(err, "couldn't stop the container: %s", container.ID())
	}
	err = c.RemoveContainer(ctx, container, true)
	if err != nil {
		return errors.Wrapf(err, "couldn't remove the container: %s", container.ID())
	}

	return nil
}

func (c *Containerd) EnsureContainerStoppedOrNonExistent(ctx context.Context, containerName string) error {

	c.log.Infof("Ensuring container %s is stopped or non-existent\n", containerName)

	container, err := c.GetContainerWithGivenName(ctx, containerName)
	if err != nil {
		return err
	}
	if container == nil {
		c.log.Infof("container %s does not exist\n", containerName)
		return nil
	}

	err = c.StopContainer(ctx, container, "10s")
	if err != nil {
		return errors.Wrapf(err, "couldn't stop the container: %s", container.ID())
	}

	return nil
}

func (c *Containerd) CreateContainer(ctx context.Context, containerName string, containerImage string) (containerd.Container, error) {
	image, err := c.Client.GetImage(ctx, containerImage)
	if err != nil {
		c.log.Infof("couldn't get %s image from client, so pulling the image\n", containerImage)
		image, err = c.Client.Pull(ctx, containerImage, containerd.WithPullUnpack)
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't pull image:%s", containerImage)
		}
		c.log.Infof("image pulled: %s\n", image.Name())
	}
	// create a container
	container, err := c.Client.NewContainer(
		ctx,
		containerName,
		containerd.WithImage(image),
		containerd.WithNewSnapshot(containerName, image),
		containerd.WithNewSpec(oci.WithImageConfig(image)),
	)
	if err != nil {
		return nil, err
	}
	return container, nil
}
func (c *Containerd) RemoveContainer(ctx context.Context, container containerd.Container, force bool) error {

	id := container.ID()
	task, err := container.Task(ctx, cio.Load)
	if err != nil {
		if errdefs.IsNotFound(err) {
			if container.Delete(ctx, containerd.WithSnapshotCleanup) != nil {
				return container.Delete(ctx)
			}
		}
		return err
	}

	status, err := task.Status(ctx)
	if err != nil {
		if errdefs.IsNotFound(err) {
			return nil
		}
		return err
	}

	switch status.Status {
	case containerd.Created, containerd.Stopped:
		if _, err := task.Delete(ctx); err != nil && !errdefs.IsNotFound(err) {
			return errors.Wrapf(err, "failed to delete task %v", id)
		}
	case containerd.Paused:
		if !force {
			err := fmt.Errorf("you cannot remove a %v container %v. Unpause the container before attempting removal or force remove", status.Status, id)
			return err
		}
		_, err := task.Delete(ctx, containerd.WithProcessKill)
		if err != nil && !errdefs.IsNotFound(err) {
			return errors.Wrapf(err, "failed to delete task %v", id)
		}
	// default is the case, when status.Status = containerd.Running
	default:
		if !force {
			err := fmt.Errorf("you cannot remove a %v container %v. Stop the container before attempting removal or force remove", status.Status, id)
			return err
		}
		if err := task.Kill(ctx, syscall.SIGKILL); err != nil {
			return fmt.Errorf("failed to send SIGKILL")
		}
		es, err := task.Wait(ctx)
		if err == nil {
			<-es
		}
		_, err = task.Delete(ctx, containerd.WithProcessKill)
		if err != nil && !errdefs.IsNotFound(err) {
			return errors.Wrapf(err, "failed to delete task %v", id)
		}
	}
	var delOpts []containerd.DeleteOpts
	if _, err := container.Image(ctx); err == nil {
		delOpts = append(delOpts, containerd.WithSnapshotCleanup)
	}
	err = container.Delete(ctx, delOpts...)
	if err != nil {
		return err
	}
	return nil
}

func (c *Containerd) StopContainer(ctx context.Context, container containerd.Container, timeoutStr string) error {

	timeout, err := time.ParseDuration(timeoutStr)
	if err != nil {
		return err
	}
	task, err := container.Task(ctx, cio.Load)
	if err != nil {
		return err
	}

	status, err := task.Status(ctx)
	if err != nil {
		return err
	}

	paused := false

	switch status.Status {
	case containerd.Created, containerd.Stopped:
		return nil
	case containerd.Paused, containerd.Pausing:
		paused = true
	default:
	}

	exitCh, err := task.Wait(ctx)
	if err != nil {
		return err
	}

	if timeout > 0 {

		if err := task.Kill(ctx, syscall.SIGTERM); err != nil {
			return err
		}

		// signal will be sent once resume is finished
		if paused {
			if err := task.Resume(ctx); err != nil {
				c.log.Infof("Cannot unpause container %s: %s\n", container.ID(), err)
			} else {
				// no need to do it again when send sigkill signal
				paused = false
			}
		}

		sigtermCtx, sigtermCtxCancel := context.WithTimeout(ctx, timeout)
		defer sigtermCtxCancel()

		err = waitContainerStop(sigtermCtx, exitCh, container.ID())
		if err == nil {
			return nil
		}

		if ctx.Err() != nil {
			return ctx.Err()
		}
	}

	if err := task.Kill(ctx, syscall.SIGKILL); err != nil {
		return err
	}

	// signal will be sent once resume is finished
	if paused {
		if err := task.Resume(ctx); err != nil {
			c.log.Infof("Cannot unpause container %s: %s\n", container.ID(), err)
		}
	}
	return waitContainerStop(ctx, exitCh, container.ID())
}

func waitContainerStop(ctx context.Context, exitCh <-chan containerd.ExitStatus, id string) error {
	select {
	case <-ctx.Done():
		if err := ctx.Err(); err != nil {
			return errors.Wrapf(err, "wait container %v", id)
		}
		return nil
	case status := <-exitCh:
		return status.Error()
	}
}

func (c *Containerd) GetContainerWithGivenName(ctx context.Context, containerName string) (containerd.Container, error) {
	// TODO: investigate use of filters to below function from containerd
	containers, err := c.Client.Containers(ctx)
	if err != nil {
		return nil, err
	}
	for _, container := range containers {
		if container.ID() == containerName {
			return container, nil
		}
	}
	c.log.Infof("container not found: %s\n", containerName)
	return nil, nil
}
