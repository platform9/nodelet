package containerruntime

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/oci"
	rspecs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/pkg/errors"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	"go.uber.org/zap"
)

type ContainerUtility struct {
	Service string
	Socket  string
	Client  *containerd.Client
	log     *zap.SugaredLogger
}

type RunOpts struct {
	Env      []string
	EnvFiles []string
	Volumes  []string
	Network  string
}

const (
	TimeOut = "10s"
	Host    = "host"
)

func NewContainerUtil() (ContainerUtils, error) {

	containerdclient, err := NewContainerdClient()
	if err != nil {
		return nil, err
	}
	return &ContainerUtility{
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

func (c *ContainerUtility) CloseClient() {
	err := c.Client.Close()
	if err != nil {
		c.log.Infof("could not close containerd connection: %v", err)
	}
}

func (c *ContainerUtility) EnsureFreshContainerRunning(ctx context.Context, namespace string, containerName string, containerImage string, runOpts RunOpts, cmdArgs []string) error {

	ctx = namespaces.WithNamespace(ctx, namespace)
	err := c.EnsureContainerDestroyed(ctx, containerName, TimeOut)
	if err != nil {
		return err
	}
	container, err := c.CreateContainer(ctx, containerName, containerImage, runOpts, cmdArgs)
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

// EnsureContainerDestroyed takes containers Name
func (c *ContainerUtility) EnsureContainerDestroyed(ctx context.Context, containerName string, timeoutStr string) error {

	container, err := c.GetContainerWithGivenName(ctx, containerName)
	if err != nil {
		return err
	}
	if container == nil {
		c.log.Infof("container not present: %s", containerName)
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

// EnsureContainersDestroyed takes containers list
func (c *ContainerUtility) EnsureContainersDestroyed(ctx context.Context, containers []containerd.Container, timeoutStr string) error {
	var err error
	for _, container := range containers {
		zap.S().Infof("stopping container:%s", container.ID())
		err = c.StopContainer(ctx, container, timeoutStr)
		if err != nil {
			zap.S().Errorf("couldn't stop the container: %s :%s", container.ID(), err)
			zap.S().Warnf("skipping container: %s", container.ID())
			continue
		}
		zap.S().Infof("container:%s stopped", container.ID())
		zap.S().Infof("removing container:%s ", container.ID())
		err = c.RemoveContainer(ctx, container, true)
		if err != nil {
			zap.S().Infof("couldn't remove the container: %s :%s", container.ID(), err)
			zap.S().Warnf("skipping container: %s", container.ID())
			continue
		}
		zap.S().Infof("container :%s destroyed", container.ID())
	}
	return nil
}

func (c *ContainerUtility) EnsureContainerStoppedOrNonExistent(ctx context.Context, containerName string) error {

	c.log.Infof("Ensuring container %s is stopped or non-existent\n", containerName)

	container, err := c.GetContainerWithGivenName(ctx, containerName)
	if err != nil {
		return err
	}
	if container == nil {
		c.log.Infof("container %s does not exist\n", containerName)
		return nil
	}

	err = c.StopContainer(ctx, container, TimeOut)
	if err != nil {
		return errors.Wrapf(err, "couldn't stop the container: %s", container.ID())
	}
	return nil
}

func (c *ContainerUtility) GetContainersInNamespace(ctx context.Context, namespace string) ([]containerd.Container, error) {

	containers, err := c.Client.Containers(ctx)
	if err != nil {
		return nil, err
	}
	return containers, nil
}

func (c *ContainerUtility) DestroyContainersInNamespace(ctx context.Context, namespace string) error {

	containers, err := c.GetContainersInNamespace(ctx, namespace)
	if err != nil {
		return errors.Wrapf(err, "error getting containers in namespace: %s ", namespace)
	}

	err = c.EnsureContainersDestroyed(ctx, containers, TimeOut)
	if err != nil {
		return errors.Wrapf(err, "could not destroy containers in namespace: %s", namespace)
	}
	return nil
}

func (c *ContainerUtility) DestroyContainersInNamespacesList(ctx context.Context, namespacelist []string) error {

	for _, namespace := range namespacelist {
		zap.S().Infof("Destroying containers in namespace: %s", namespace)
		ctx = namespaces.WithNamespace(ctx, namespace)
		err := c.DestroyContainersInNamespace(ctx, namespace)
		return err
	}
	return nil
}

func (c *ContainerUtility) CreateContainer(ctx context.Context, containerName string, containerImage string, runOpts RunOpts, cmdArgs []string) (containerd.Container, error) {

	image, err := c.Client.GetImage(ctx, containerImage)
	if err != nil {
		c.log.Infof("couldn't get %s image from client, so pulling the image\n", containerImage)
		image, err = c.Client.Pull(ctx, containerImage, containerd.WithPullUnpack)
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't pull image:%s", containerImage)
		}
		c.log.Infof("image pulled: %s\n", image.Name())
	}
	var (
		opts  []oci.SpecOpts
		cOpts []containerd.NewContainerOpts
	)

	// TODO: is unpacking required?
	if err := image.Unpack(ctx, constants.DefaultSnapShotter); err != nil {
		return nil, fmt.Errorf("error unpacking image: %w", err)
	}

	if runOpts.Network == Host {
		opts = append(opts, oci.WithHostNamespace(rspecs.NetworkNamespace), oci.WithHostHostsFile, oci.WithHostResolvconf)
	}
	// TODO: net for cni, none and invalid

	opts = append(opts, oci.WithImageConfigArgs(image, cmdArgs))
	// check if this required //opts = append(opts, oci.WithProcessArgs(processArgs...)) and how to use

	env := runOpts.Env
	if len(env) > 0 {
		opts = append(opts, oci.WithEnv(env))
	}

	envFiles := runOpts.EnvFiles
	if len(envFiles) > 0 {
		env, err := parseEnvVars(envFiles)
		if err != nil {
			return nil, err
		}
		opts = append(opts, oci.WithEnv(env))
	}

	mounts := []rspecs.Mount{}
	for _, v := range runOpts.Volumes {
		split := strings.Split(v, ":")
		src := split[0]
		dst := split[1]
		// when we provide mode option like --volume ${CERTS_DIR}/authn_webhook/:/certs:ro" here `ro` is mode
		options := []string{}
		if len(split) == 3 {
			options = append(options, split[2])
		}
		options = append(options, "rbind")
		mount := rspecs.Mount{
			Type:        "none",
			Source:      src,
			Destination: dst,
			Options:     options,
		}
		mounts = append(mounts, mount)
	}
	opts = append(opts, oci.WithMounts(mounts))

	cOpts = append(cOpts, containerd.WithImage(image))
	cOpts = append(cOpts, containerd.WithNewSnapshot(containerName, image))

	var s rspecs.Spec
	spec := containerd.WithSpec(&s, opts...)
	cOpts = append(cOpts, spec)

	// create a container
	container, err := c.Client.NewContainer(ctx, containerName, cOpts...)
	if err != nil {
		return nil, err
	}

	return container, nil
}

func (c *ContainerUtility) RemoveContainer(ctx context.Context, container containerd.Container, force bool) error {

	id := container.ID()
	task, err := container.Task(ctx, cio.Load)
	if err != nil {

		if errdefs.IsNotFound(err) {
			zap.S().Infof("task not found so deleting directly with snapshot cleanup")
			if container.Delete(ctx, containerd.WithSnapshotCleanup) != nil {
				zap.S().Infof("failed to delete with snapshot")
				if err = container.Delete(ctx); errdefs.IsNotFound(err) {
					zap.S().Infof("container not found on store so skipping delete")
					return nil
				}
				zap.S().Infof("container delete failed:%v", err)
				return err
			}
			zap.S().Infof("container deleted with snapshot successfully: %v", id)
			return nil
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

func (c *ContainerUtility) StopContainer(ctx context.Context, container containerd.Container, timeoutStr string) error {

	timeout, err := time.ParseDuration(timeoutStr)
	if err != nil {
		return err
	}
	task, err := container.Task(ctx, cio.Load)
	if err != nil {
		if errdefs.IsNotFound(err) {
			zap.S().Infof("task not found in container:%s", container.ID())
			return nil
		}
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

func (c *ContainerUtility) GetContainerWithGivenName(ctx context.Context, containerName string) (containerd.Container, error) {
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

func parseEnvVars(paths []string) ([]string, error) {
	vars := make([]string, 0)
	for _, path := range paths {
		f, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("failed to open env file %s: %w", path, err)
		}
		defer f.Close()

		sc := bufio.NewScanner(f)
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			// skip comment lines
			if strings.HasPrefix(line, "#") {
				continue
			}
			vars = append(vars, line)
		}
		if err = sc.Err(); err != nil {
			return nil, err
		}
	}
	return vars, nil
}
