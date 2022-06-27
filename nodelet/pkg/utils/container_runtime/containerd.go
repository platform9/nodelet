package containerruntime

import (
	"context"
	"fmt"
	"syscall"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/oci"
	"github.com/pkg/errors"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	"go.uber.org/zap"

	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/platforms"
	refdocker "github.com/containerd/containerd/reference/docker"
	"github.com/containerd/containerd/remotes"
	"github.com/containerd/imgcrypt"
	"github.com/containerd/imgcrypt/images/encryption"
	"github.com/containerd/nerdctl/pkg/errutil"
	"github.com/containerd/nerdctl/pkg/imgutil/dockerconfigresolver"
	"github.com/containerd/nerdctl/pkg/imgutil/pull"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/opencontainers/runtime-spec/specs-go"
)

type Containerd struct {
	Service string
	Socket  string
	Client  *containerd.Client
	log     *zap.SugaredLogger
}

var (
	containerdclient *containerd.Client
)

const (
	NameLabelForContainer = "nerdctl/name"
	IDLength              = 64
)

func NewContainerd() (Runtime, error) {

	containerdclient, err = NewContainerdClient()
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

	containerdclient, err = containerd.New(constants.ContainerdSocket)
	if err != nil {
		zap.S().Info("failed to create containerd client")
		return nil, errors.Wrap(err, "failed to create containerd client")
	}
	return containerdclient, nil
}

func (c *Containerd) EnsureFreshContainerRunning(ctx context.Context, containerName string, containerImage string) error {
	err := c.EnsureContainerDestroyed(ctx, containerName, "10s")
	if err != nil {
		return err
	}
	cont, err := CreateContainer(c.Client, ctx, containerImage, containerName)
	if err != nil {
		return err
	}
	task, err := cont.NewTask(ctx, cio.NewCreator())
	if err != nil {
		return errors.Wrapf(err, "couldn't create task from container %s\n", containerName)
	}
	statusC, err := task.Wait(ctx)
	if err != nil {
		return err
	}
	err = task.Start(ctx)
	if err != nil {
		return errors.Wrapf(err, "couldn't start task from container %s\n", containerName)
	}
	status := <-statusC
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

	containers, err := GetContainersWithGivenName(c.Client, ctx, containerName)
	if err != nil {
		return err
	}

	for _, container := range containers {
		err = StopContainer(c.Client, ctx, container, timeoutStr)
		if err != nil {
			return err
		}
		err = RemoveContainer(c.Client, ctx, container, true)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Containerd) EnsureContainerStoppedOrNonExistent(ctx context.Context, containerName string) error {

	fmt.Printf("Ensuring container %s is stopped or non-existent", containerName)

	containers, err := GetContainersWithGivenName(c.Client, ctx, containerName)
	if err != nil {
		return err
	}
	if len(containers) == 0 {
		fmt.Printf("container %s does not exist", containerName)
		return nil
	}
	if len(containers) > 1 {
		fmt.Printf("multiple containers with same name present")
	}
	for _, container := range containers {
		err = StopContainer(c.Client, ctx, container, "10s")
		if err != nil {
			return err
		}
	}

	return nil
}

func GetContainersWithGivenName(client *containerd.Client, ctx context.Context, containerName string) ([]containerd.Container, error) {

	filters := []string{
		fmt.Sprintf("labels.%s==%s", "nerdctl/name", containerName),
	}
	containers, err := client.Containers(ctx, filters...)
	if err != nil {
		return nil, err
	}
	return containers, nil
}

func StopContainer(client *containerd.Client, ctx context.Context, container containerd.Container, timeoutStr string) error {

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

	// NOTE: ctx is main context so that it's ok to use for task.Wait().
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
				fmt.Printf("Cannot unpause container %s: %s", container.ID(), err)
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
			fmt.Printf("Cannot unpause container %s: %s", container.ID(), err)
		}
	}
	return waitContainerStop(ctx, exitCh, container.ID())
}

func waitContainerStop(ctx context.Context, exitCh <-chan containerd.ExitStatus, id string) error {
	select {
	case <-ctx.Done():
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("wait container %v: %w", id, err)
		}
		return nil
	case status := <-exitCh:
		return status.Error()
	}
}

func RemoveContainer(client *containerd.Client, ctx context.Context, container containerd.Container, force bool) (retErr error) {
	id := container.ID()
	defer func() {
		if errdefs.IsNotFound(retErr) {
			retErr = nil
		} else {
			fmt.Printf("failed to remove container %q", id)
		}
	}()

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
			return fmt.Errorf("failed to delete task %v: %w", id, err)
		}
	case containerd.Paused:
		if !force {
			err := fmt.Errorf("you cannot remove a %v container %v. Unpause the container before attempting removal or force remove", status.Status, id)
			return err
		}
		_, err := task.Delete(ctx, containerd.WithProcessKill)
		if err != nil && !errdefs.IsNotFound(err) {
			return fmt.Errorf("failed to delete task %v: %w", id, err)
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
			return fmt.Errorf("failed to delete task %v", id)
		}
	}
	var delOpts []containerd.DeleteOpts
	if _, err := container.Image(ctx); err == nil {
		delOpts = append(delOpts, containerd.WithSnapshotCleanup)
	}

	if err := container.Delete(ctx, delOpts...); err != nil {
		return err
	}
	return nil
}

func CreateContainer(client *containerd.Client, ctx context.Context, imgref string, name string) (containerd.Container, error) {
	containerdImage, err := Pull(client, ctx, imgref)
	var (
		opts  []oci.SpecOpts
		cOpts []containerd.NewContainerOpts
		id    = GenerateID()
		s     specs.Spec
	)

	opts = append(opts,
		oci.WithDefaultSpec(),
	)
	opts = append(opts, oci.WithImageConfig(containerdImage))
	//imageRef = ensuredImage.Ref
	//name = referenceutil.SuggestContainerName(imageRef, id)
	cOpts = append(cOpts, containerd.WithNewSnapshot(id, containerdImage))

	spec := containerd.WithSpec(&s, opts...)

	cOpts = append(cOpts, spec)

	m := make(map[string]string)
	if name != "" {
		m[NameLabelForContainer] = name
	}
	cOpts = append(cOpts, containerd.WithAdditionalContainerLabels(m))

	container, err := client.NewContainer(ctx, id, cOpts...)
	if err != nil {
		return nil, err
	}
	return container, nil
}

func Pull(client *containerd.Client, ctx context.Context, rawRef string) (containerd.Image, error) {
	ocispecPlatforms := []ocispec.Platform{platforms.DefaultSpec()}

	named, err := refdocker.ParseDockerRef(rawRef)
	if err != nil {
		return nil, err
	}
	ref := named.String()
	refDomain := refdocker.Domain(named)

	var dOpts []dockerconfigresolver.Opt
	dOpts = append(dOpts, dockerconfigresolver.WithSkipVerifyCerts(true))
	resolver, err := dockerconfigresolver.New(ctx, refDomain, dOpts...)
	if err != nil {
		return nil, err
	}

	img, err := PullImage(client, ctx, constants.DefaultSnapShotter, resolver, ref, ocispecPlatforms)
	if err != nil {
		// In some circumstance (e.g. people just use 80 port to support pure http), the error will contain message like "dial tcp <port>: connection refused".
		if !errutil.IsErrHTTPResponseToHTTPSClient(err) && !errutil.IsErrConnectionRefused(err) {
			return nil, err
		}
		fmt.Printf("server %q does not seem to support HTTPS, falling back to plain HTTP", refDomain)
		dOpts = append(dOpts, dockerconfigresolver.WithPlainHTTP(true))
		resolver, err = dockerconfigresolver.New(ctx, refDomain, dOpts...)
		if err != nil {
			return nil, err
		}
		return PullImage(client, ctx, constants.DefaultSnapShotter, resolver, ref, ocispecPlatforms)

	}
	return img, nil
}

func PullImage(client *containerd.Client, ctx context.Context, snapshotter string, resolver remotes.Resolver, ref string, ocispecPlatforms []ocispec.Platform) (containerd.Image, error) {
	ctx, done, err := client.WithLease(ctx)
	if err != nil {
		return nil, err
	}
	defer done(ctx)

	var containerdImage containerd.Image
	config := &pull.Config{
		Resolver:   resolver,
		RemoteOpts: []containerd.RemoteOpt{},
		Platforms:  ocispecPlatforms, // empty for all-platforms
	}

	fmt.Printf("The image will be unpacked for platform %q, snapshotter %q.", ocispecPlatforms[0], snapshotter)
	imgcryptPayload := imgcrypt.Payload{}
	imgcryptUnpackOpt := encryption.WithUnpackConfigApplyOpts(encryption.WithDecryptedUnpack(&imgcryptPayload))
	config.RemoteOpts = append(config.RemoteOpts,
		containerd.WithPullUnpack,
		containerd.WithUnpackOpts([]containerd.UnpackOpt{imgcryptUnpackOpt}))
	config.RemoteOpts = append(
		config.RemoteOpts,
		containerd.WithPullSnapshotter(snapshotter))
	containerdImage, err = pull.Pull(ctx, client, ref, config)
	if err != nil {
		return nil, err
	}
	return containerdImage, nil

}

func GenerateID() string {
	bytesLength := IDLength / 2
	b := make([]byte, bytesLength)
	n, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	if n != bytesLength {
		panic(fmt.Errorf("expected %d bytes, got %d bytes", bytesLength, n))
	}
	return hex.EncodeToString(b)
}
