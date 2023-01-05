package containerruntime

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/mount"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/oci"
	"github.com/containerd/containerd/pkg/userns"
	"github.com/opencontainers/image-spec/identity"
	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/pkg/errors"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	"go.uber.org/zap"
	"golang.org/x/sys/unix"
)

type ContainerUtility struct {
	Service string
	Socket  string
	Client  *containerd.Client
	log     *zap.SugaredLogger
}

type RunOpts struct {
	Env        []string
	EnvFiles   []string
	Volumes    []string
	Network    string
	Privileged bool
}

const (
	TimeOut = "10s"
	Host    = "host"
)
const (
	IDLength = 64
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

func (c *ContainerUtility) CloseClientConnection() {
	if c.Client != nil {
		err := c.Client.Close()
		if err != nil {
			c.log.Warnf("couldn't close containerd client connection: %v", err)
			return
		}
		c.log.Info("closed containerd client connection")
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
	c.log.Infof("container: %v created", container.ID())
	//create a task from the container
	task, err := container.NewTask(ctx, cio.NewCreator(cio.WithStdio))
	if err != nil {
		return errors.Wrapf(err, "failed to create new task:%s", containerName)
	}

	// make sure we wait before calling start
	exitStatusC, err := task.Wait(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to wait for new task:%v in %s", task.ID(), containerName)
	}

	if err := task.Start(ctx); err != nil {
		return errors.Wrapf(err, "failed to start new task:%v in %s", task.ID(), containerName)
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

	var (
		opts  []oci.SpecOpts
		cOpts []containerd.NewContainerOpts
	)

	image, err := c.EnsureImage(ctx, containerImage)
	if err != nil {
		return nil, err
	}

	opts = GetNetworkopts(runOpts, opts)
	opts = GetCmdargsOpts(image, cmdArgs, opts)
	//opts = SetPlatformOptions(opts) //is it required?
	if runOpts.Privileged {
		opts = GetPrivilegedOpts(runOpts, opts)
	}

	opts, err = GetEnvOpts(runOpts, opts)
	if err != nil {
		return nil, err
	}

	// err = c.MountandUnmount(ctx, image) // dont know whats it doing. just added for cheking its working
	// if err != nil {
	// 	c.log.Errorf("error in mountandunmount:%v", err)
	// 	return nil, err
	// }

	opts, err = GetVolumeOPts(runOpts, opts)
	if err != nil {
		c.log.Errorf("error getting volumeopts:%v", err)
		return nil, err
	}

	cOpts = append(cOpts, containerd.WithImage(image))
	//cOpts = append(cOpts, containerd.WithNewSnapshot(containerName+"-snapshot", image))

	var specs specs.Spec
	spec := containerd.WithSpec(&specs, opts...)
	cOpts = append(cOpts, spec)

	// add lables for container name
	labels := map[string]string{
		"pf9.io/containerName": containerName,
	}
	cOpts = append(cOpts, containerd.WithAdditionalContainerLabels(labels))
	// generate id
	id := GenerateContID()
	// create a container
	container, err := c.Client.NewContainer(ctx, id, cOpts...)
	if err != nil {
		c.log.Errorf("error creating container:%v", err)
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
	task, err := container.Task(ctx, cio.Load) // can we use this cio for taking logs?
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
	// TODO: we can use of filters to list containers
	// namelable := "pf9.io/containerName"
	// filters := []string{
	// 	fmt.Sprintf("labels.%s==%s", namelable, containerName),
	// }
	// containers, err := c.Client.Containers(ctx, filters...)
	containers, err := c.Client.Containers(ctx)
	if err != nil {
		return nil, err
	}
	if len(containers) < 1 {
		c.log.Infof("container not found: %s\n", containerName)
		return nil, nil
	}
	for _, container := range containers {
		lables, err := container.Labels(ctx)
		if err != nil {
			return nil, err
		}
		if lables["pf9.io/containerName"] == containerName {
			return container, nil
		}
	}
	c.log.Infof("container not found: %s\n", containerName)
	return nil, nil
}

func (c *ContainerUtility) IsContainerRunning(ctx context.Context, containerName string) (bool, error) {
	cont, err := c.GetContainerWithGivenName(ctx, containerName)
	if err != nil {
		return false, err
	}
	task, err := cont.Task(ctx, cio.Load)
	if err != nil {
		if errdefs.IsNotFound(err) {
			zap.S().Infof("task not found in container:%s", cont.ID())
			return false, nil
		}
		return false, err
	}
	status, err := task.Status(ctx)
	if err != nil {
		return false, err
	}
	if status.Status == containerd.Running {
		return true, nil
	} // else {
	//get container logs //is necessary?
	//}
	return false, nil
}

// just checks if container exist. it doesnt check for the task is present or not.
func (c *ContainerUtility) IsContainerExist(ctx context.Context, containerName string) (bool, error) {
	cont, err := c.GetContainerWithGivenName(ctx, containerName)
	if err != nil {
		return false, err
	}
	if cont != nil {
		return true, nil
	}
	return false, nil
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

// getUnprivilegedMountFlags is from https://github.com/moby/moby/blob/v20.10.5/daemon/oci_linux.go#L420-L450
//
// Get the set of mount flags that are set on the mount that contains the given
// path and are locked by CL_UNPRIVILEGED. This is necessary to ensure that
// bind-mounting "with options" will not fail with user namespaces, due to
// kernel restrictions that require user namespace mounts to preserve
// CL_UNPRIVILEGED locked flags.
func getUnprivilegedMountFlags(path string) ([]string, error) {
	var statfs unix.Statfs_t
	if err := unix.Statfs(path, &statfs); err != nil {
		return nil, err
	}

	// The set of keys come from https://github.com/torvalds/linux/blob/v4.13/fs/namespace.c#L1034-L1048.
	unprivilegedFlags := map[uint64]string{
		unix.MS_RDONLY:     "ro",
		unix.MS_NODEV:      "nodev",
		unix.MS_NOEXEC:     "noexec",
		unix.MS_NOSUID:     "nosuid",
		unix.MS_NOATIME:    "noatime",
		unix.MS_RELATIME:   "relatime",
		unix.MS_NODIRATIME: "nodiratime",
	}

	var flags []string
	for mask, flag := range unprivilegedFlags {
		if uint64(statfs.Flags)&mask == mask {
			flags = append(flags, flag)
		}
	}

	return flags, nil
}

// checks and removes duplicate values in string slice
func DedupeStrSlice(in []string) []string {
	m := make(map[string]struct{})
	var res []string
	for _, s := range in {
		if _, ok := m[s]; !ok {
			res = append(res, s)
			m[s] = struct{}{}
		}
	}
	return res
}

func (c *ContainerUtility) EnsureImage(ctx context.Context, containerImage string) (containerd.Image, error) {
	image, err := c.Client.GetImage(ctx, containerImage)
	if err != nil {
		c.log.Infof("couldn't get %s image from client, so pulling the image\n", containerImage)
		image, err = c.Client.Pull(ctx, containerImage, containerd.WithPullUnpack) // withpull unpack required?
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't pull image:%s", containerImage)
		}
		c.log.Infof("image pulled: %s\n", image.Name())
	}

	return image, nil
}

func (c *ContainerUtility) MountandUnmount(ctx context.Context, image containerd.Image) error {
	diffIDs, err := image.RootFS(ctx)
	if err != nil {
		return errors.Wrapf(err, "could not get rootfs")
	}
	chainID := identity.ChainID(diffIDs).String()

	s := c.Client.SnapshotService(constants.DefaultSnapShotter)
	tempDir, err := os.MkdirTemp("", "initialC")
	if err != nil {
		return errors.Wrapf(err, "could not mkdirtemp")
	}
	// We use Remove here instead of RemoveAll.
	// The RemoveAll will delete the temp dir and all children it contains.
	// When the Unmount fails, RemoveAll will incorrectly delete data from the mounted dir
	defer os.Remove(tempDir)

	var mounts []mount.Mount
	mounts, err = s.View(ctx, tempDir, chainID)
	if err != nil {
		return errors.Wrapf(err, "could not view snapshotter")
	}

	unmounter := func(mountPath string) {
		if uerr := mount.Unmount(mountPath, 0); uerr != nil {
			zap.S().Debugf("Failed to unmount snapshot %q", tempDir)
			if err == nil {
				err = uerr
			}
		}
	}

	defer unmounter(tempDir)
	if err := mount.All(mounts, tempDir); err != nil {
		if err := s.Remove(ctx, tempDir); err != nil && !errdefs.IsNotFound(err) {
			return errors.Wrapf(err, "could not remove tempdir")
		}
		return errors.Wrapf(err, "could not get get mount.all")
	}
	return nil
}

func GetNetworkopts(runOpts RunOpts, opts []oci.SpecOpts) []oci.SpecOpts {

	if runOpts.Network == Host {
		opts = append(opts, oci.WithHostNamespace(specs.NetworkNamespace), oci.WithHostHostsFile, oci.WithHostResolvconf)
	}
	return opts
	// TODO: --net for cni, none and invalid
}

func SetPlatformOptions(opts []oci.SpecOpts) []oci.SpecOpts {
	opts = append(opts,
		oci.WithDefaultUnixDevices,
		oci.WithoutRunMount, // unmount default tmpfs on "/run": https://github.com/containerd/nerdctl/issues/157)
	)
	//is this needed/right/wrong ?
	opts = append(opts,
		oci.WithMounts([]specs.Mount{
			{Type: "cgroup", Source: "cgroup", Destination: "/sys/fs/cgroup", Options: []string{"ro", "nosuid", "noexec", "nodev"}},
		}))
	return opts
}

func GetEnvOpts(runOpts RunOpts, opts []oci.SpecOpts) ([]oci.SpecOpts, error) {
	env := runOpts.Env
	if len(env) > 0 {
		opts = append(opts, oci.WithEnv(env))
	}

	envFiles := runOpts.EnvFiles
	if len(envFiles) > 0 {
		env, err := parseEnvVars(envFiles)
		if err != nil {
			zap.S().Errorf("error parsing env vars: %v", err)
			return opts, err
		}
		opts = append(opts, oci.WithEnv(env))
	}
	return opts, nil
}

func GetVolumeOPts(runOpts RunOpts, opts []oci.SpecOpts) ([]oci.SpecOpts, error) {
	mounts := []specs.Mount{}
	for _, v := range runOpts.Volumes {
		split := strings.Split(v, ":")
		src := split[0]
		dst := split[1]
		// flag like --volume ${CERTS_DIR}/authn_webhook/:/certs:ro" contains here `ro` which is mode/option = split[2]
		options := []string{}
		if len(split) == 3 {
			options = append(options, split[2])
		}
		options = append(options, "rbind") // dont know why appending rbind.
		if userns.RunningInUserNS() {
			unpriv, err := getUnprivilegedMountFlags(src)
			if err != nil {
				return opts, errors.Wrapf(err, "error getting unprirvileged mount flags")
			}
			options = DedupeStrSlice(append(options, unpriv...))
		}

		mount := specs.Mount{
			Type:        "none", // dont know why its none
			Source:      src,
			Destination: dst,
			Options:     options,
		}
		mounts = append(mounts, mount)
	}

	opts = append(opts, oci.WithMounts(mounts))

	return opts, nil
}

func GetCmdargsOpts(image containerd.Image, cmdArgs []string, opts []oci.SpecOpts) []oci.SpecOpts {
	opts = append(opts, oci.WithImageConfigArgs(image, cmdArgs))
	// check if this required //opts = append(opts, oci.WithProcessArgs(processArgs...)) and how to use
	return opts
}

func GetPrivilegedOpts(runOpts RunOpts, opts []oci.SpecOpts) []oci.SpecOpts {

	if runOpts.Privileged {
		privilegedOpts := []oci.SpecOpts{
			oci.WithPrivileged,
			oci.WithAllDevicesAllowed,
			oci.WithHostDevices,
			oci.WithNewPrivileges,
		}
		opts = append(opts, privilegedOpts...)
	}

	return opts
}

func GenerateContID() string {
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
