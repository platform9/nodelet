package containerruntime

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	"go.uber.org/zap"
)

type Docker struct {
	Service string
	Socket  string
	Client  *client.Client
	log     *zap.SugaredLogger
}

var (
	err          error
	dockerClient *client.Client
)

func NewDocker() (Runtime, error) {
	dockerClient, err = NewDockerClient()
	if err != nil {
		return nil, err
	}
	return &Docker{
		Service: "docker",
		Socket:  constants.DockerSocket,
		Client:  dockerClient,
		log:     zap.S(),
	}, nil
}

func NewDockerClient() (*client.Client, error) {
	var err error
	opts := client.WithHost(constants.DockerSocket)
	dclient, err := client.NewClientWithOpts(opts)
	if err != nil {
		zap.S().Errorf("Could not create docker client: %s", err)
		return nil, errors.Wrap(err, "couldn't create docker client")
	}
	return dclient, nil
}

func (d *Docker) EnsureFreshContainerRunning(ctx context.Context, containerName string, containerImage string) error {

	err = d.EnsureContainerDestroyed(ctx, containerName, "10s")
	if err != nil {
		return errors.Wrapf(err, "couldn't remove container: %s", containerName)
	}

	cont, err := d.Client.ContainerCreate(
		ctx,
		&container.Config{
			Image: containerImage,
		},
		nil, nil, nil,
		containerName,
	)
	if err != nil {
		return errors.Wrapf(err, "could not create container: %s", containerName)
	}

	err = d.Client.ContainerStart(context.Background(), cont.ID, types.ContainerStartOptions{})
	if err != nil {
		return errors.Wrapf(err, "could not start container: %s", containerName)
	}
	d.log.Infof("Container %s is started", cont.ID)
	return nil

}

func (d *Docker) EnsureContainerDestroyed(ctx context.Context, containerName string, timeout string) error {
	d.log.Infof("Ensuring container %s is destroyed", containerName)

	_, err := d.Client.ContainerInspect(ctx, containerName)

	if err != nil {
		//err is not nil means container with given name doesnt exist
		//TODO: specifically check if err is ObjectNotFoundError{container: containerName}
		return nil
	}

	if err := d.Client.ContainerStop(ctx, containerName, nil); err != nil {
		d.log.Errorf("unable to stop container: %s", err)
		return err
	}
	removeOptions := types.ContainerRemoveOptions{
		RemoveVolumes: true, // is this needed
		Force:         true,
	}
	if err := d.Client.ContainerRemove(ctx, containerName, removeOptions); err != nil {
		d.log.Errorf("unable to remove container: %s", err)
		return err
	}

	return nil
}

func (d *Docker) EnsureContainerStoppedOrNonExistent(ctx context.Context, containerName string) error {
	d.log.Infof("Ensuring container %s is stopped or non-existent", containerName)
	cont, err := d.Client.ContainerInspect(ctx, containerName)
	if err != nil {
		d.log.Infof("container %s does not exist", containerName)
		return nil
	}
	runState := cont.State.Running
	if runState {
		d.log.Infof("stopping container %s", containerName)
		if err := d.Client.ContainerStop(ctx, containerName, nil); err != nil {
			d.log.Errorf("unable to stop container: %s", err)
			return err
		}
	} else {
		d.log.Infof("container %s is already stopped", containerName)
	}
	return nil
}
