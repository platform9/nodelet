package containerruntime

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"
	"github.com/platform9/nodelet/nodelet/pkg/utils/config"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	"go.uber.org/zap"
)

type DockerImpl struct {
	Service string
	Cli     string
	Socket  string
	Client  *client.Client
	log     *zap.SugaredLogger
}

var (
	err          error
	dockerClient *client.Client
)

func NewDocker() (Runtime, error) {
	dockerClient, err = newDockerClient()
	if err != nil {
		return nil, err
	}
	return &DockerImpl{
		Service: "docker",
		Cli:     "/usr/bin/docker",
		Socket:  constants.DockerSocket,
		Client:  dockerClient,
		log:     zap.S(),
	}, nil
}

func newDockerClient() (*client.Client, error) {
	var err error
	opts := client.WithHost(constants.DockerSocket)
	dclient, err := client.NewClientWithOpts(opts)
	if err != nil {
		zap.S().Errorf("Could not create docker client: %s", err)
		return nil, err
	}
	return dclient, nil
}

func (d *DockerImpl) EnsureFreshContainerRunning(ctx context.Context, cfg config.Config, containerName string, containerImage string) error {

	err = d.EnsureContainerDestroyed(ctx, cfg, containerName)
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

func (d *DockerImpl) EnsureContainerDestroyed(ctx context.Context, cfg config.Config, containerName string) error {
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

func (d *DockerImpl) EnsureContainerStoppedOrNonExistent(ctx context.Context, cfg config.Config, containerName string) error {
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
