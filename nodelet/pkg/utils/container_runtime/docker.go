package containerruntime

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
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
}

var (
	err          error
	dockerClient *client.Client
)

func NewDocker() Runtime {
	opts := client.WithHost(constants.DockerSocket)
	dockerClient, err = client.NewClientWithOpts(opts)
	if err != nil {
		zap.S().Errorf("Could not create docker client: %s", err)
	}
	return &DockerImpl{
		Service: "docker",
		Cli:     "/usr/bin/docker",
		Socket:  constants.DockerSocket,
		Client:  dockerClient,
	}
}

func (r *DockerImpl) NewDockerClient() {
	var err error
	opts := client.WithHost(r.Socket)
	r.Client, err = client.NewClientWithOpts(opts)
	if err != nil {
		zap.S().Errorf("Could not create docker client: %s", err)
	}
}

func (r *DockerImpl) EnsureFreshContainerRunning(ctx context.Context, cfg config.Config, containerName string, containerImage string, Ip string, port string) error {
	//TODO: find way to add below tags
	//-dest ${EXTERNAL_DNS_NAME}
	//"-d --net host"
	err = r.EnsureContainerDestroyed(ctx, cfg, containerName)
	if err != nil {
		return errors.Wrapf(err, "couldn't remove container: %s", containerName)
	}

	hostBinding := nat.PortBinding{
		HostIP:   Ip,
		HostPort: port,
	}
	containerPort, err := nat.NewPort("tcp", "80")
	if err != nil {
		return errors.Wrap(err, "Unable to get the port")
	}

	portBinding := nat.PortMap{containerPort: []nat.PortBinding{hostBinding}}
	cont, err := r.Client.ContainerCreate(
		ctx,
		&container.Config{
			Image: containerImage,
		},
		&container.HostConfig{
			PortBindings: portBinding,
		}, nil, nil,
		containerName,
	)
	if err != nil {
		return errors.Wrapf(err, "could not create container: %s", containerName)
	}

	err = r.Client.ContainerStart(context.Background(), cont.ID, types.ContainerStartOptions{})
	if err != nil {
		return errors.Wrapf(err, "could not start container: %s", containerName)
	}
	zap.S().Infof("Container %s is started", cont.ID)
	return nil

}

func (r *DockerImpl) EnsureContainerDestroyed(ctx context.Context, cfg config.Config, containerName string) error {
	zap.S().Infof("Ensuring container %s is destroyed", containerName)

	_, err := r.Client.ContainerInspect(ctx, containerName)

	if err != nil {
		//err is not nil means container with given name doesnt exist
		//TODO: specifically check if err is ObjectNotFoundError{container: containerName}
		return nil
	}

	if err := r.Client.ContainerStop(ctx, containerName, nil); err != nil {
		zap.S().Errorf("unable to stop container: %s", err)
		return err
	}
	removeOptions := types.ContainerRemoveOptions{
		RemoveVolumes: true, // is this needed
		Force:         true,
	}
	if err := r.Client.ContainerRemove(ctx, containerName, removeOptions); err != nil {
		zap.S().Errorf("unable to remove container: %s", err)
		return err
	}

	return nil
}

func (r *DockerImpl) EnsureContainerStoppedOrNonExistent(ctx context.Context, cfg config.Config, containerName string) error {
	zap.S().Infof("Ensuring container %s is stopped or non-existent", containerName)
	cont, err := r.Client.ContainerInspect(ctx, containerName)
	if err != nil {
		zap.S().Infof("container %s does not exist", containerName)
		return nil
	}
	runState := cont.State.Running
	if runState {
		zap.S().Infof("stopping container %s", containerName)
		if err := r.Client.ContainerStop(ctx, containerName, nil); err != nil {
			zap.S().Errorf("unable to stop container: %s", err)
			return err
		}
	} else {
		zap.S().Infof("container %s is already stopped", containerName)
	}
	return nil
}
