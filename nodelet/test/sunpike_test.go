package test

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"reflect"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/spf13/pflag"
	"google.golang.org/grpc"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	nodeletcmd "github.com/platform9/nodelet/pkg/nodelet/cmd"
	"github.com/platform9/nodelet/pkg/utils/config"
	"github.com/platform9/nodelet/pkg/utils/constants"
	sunpikev1alpha1 "github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike/v1alpha1"
	"github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/hosts"
	"github.com/platform9/pf9-qbert/sunpike/conductor/pkg/api"
)

var (
	// Set the image tag to be used to a non-default 'e2e' to avoid
	// accidentally testing with an outdated yet existing `latest` image.
	sunpikeConductorImage = getEnvOrDefault("TEST_SUNPIKE_CONDUCTOR_IMAGE", "sunpike-conductor:e2e")
	nodeletdPath          = getEnvOrDefault("TEST_NODELETD_PATH", "nodeletd")

	containerNamePrefix           = "nodelet_sunpike_test_"
	sunpikeConductorContainerName = containerNamePrefix + "conductor"
	defaultConfigPath             = "testdata/sunpike_test_config.yaml"
	tmpTestDateDir                = path.Join(os.TempDir(), "nodelet", "sunpike_test")
)

// TestNodeletSunpike tests the communication between Nodelet and the Sunpike components.
func TestNodeletSunpike(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "nodelet-sunpike Suite")
}

var _ = Describe("With memory-backed conductor", func() {
	ctx, cancel := context.WithCancel(context.Background())
	var conductorClient api.Client
	var originalConductorState []*sunpikev1alpha1.Host

	BeforeSuite(func() {
		By("Verifying that prerequisites are satisfied")
		ensureCommandExists("docker")
		ensureCommandExists(nodeletdPath)
		ensureDockerImageExists(ctx, sunpikeConductorImage)

		By("Checking if nodeletd is working.")
		cmd := exec.Command(nodeletdPath, "version")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		Expect(err).To(BeNil())

		By("Cleaning up any potential left-over test resources from a previous test")
		_ = stopAndRemoveDockerContainer(ctx, sunpikeConductorContainerName)
		_ = os.RemoveAll(tmpTestDateDir)

		By("Deploying sunpike-conductor in Docker")
		startSunpikeConductorInDocker(ctx, sunpikeConductorContainerName, sunpikeConductorImage)

		By("Configuring a conductor client")
		conductorClient = getConductorClient()

		By("Awaiting the setup of the conductor")
		Eventually(func() error {
			_, err := conductorClient.ListHosts(ctx, &api.ListHostsRequest{})
			return err
		}, time.Minute, 2*time.Second).Should(BeNil())

		By("Creating temporary test data directory: " + tmpTestDateDir)
		err = os.MkdirAll(tmpTestDateDir, 0755)
		Expect(err).To(BeNil())
		fmt.Printf("Test directory: %s\n", tmpTestDateDir)
	})

	AfterSuite(func() {
		cancel()
		if os.Getenv("KEEP_TEST_RESOURCES") == "" {
			By("Cleaning up all Docker containers used in the test")
			Expect(stopAndRemoveDockerContainer(context.Background(), sunpikeConductorContainerName)).To(BeNil())
			By("Removing temporary test data directory: " + tmpTestDateDir)
			Expect(os.RemoveAll(tmpTestDateDir)).To(BeNil())
			By("To retain the test resources; set KEEP_TEST_RESOURCES to a non-empty value.")
		} else {
			By("Not cleaning up test resources because KEEP_TEST_RESOURCES was set to a non-empty value")
		}
	})

	BeforeEach(func() {
		// We do not have an API to delete hosts in conductor, so instead we
		// are just loading the current state of the conductor to use in the
		// tests.
		resp, err := conductorClient.ListDetailedHosts(ctx, &api.ListDetailedHostsRequest{})
		Expect(err).To(BeNil())
		originalConductorState = resp.Hosts
	})

	Context("sending status update to Sunpike", func() {

		It("should fail for an invalid host (missing hostid)", func() {
			// Note: we rely on the ordering of the tests in this context
			err := execNodeletd(ctx, defaultTestRootOptions())
			Expect(err).To(BeNil())

			// Check at sunpike-conductor if the Host was updated as expected
			resp, err := conductorClient.ListDetailedHosts(ctx, &api.ListDetailedHostsRequest{})
			Expect(err).To(BeNil())
			Expect(resp.Hosts).To(Equal(originalConductorState))
		})

		It("should succeed for a new Host", func() {
			HostID := hosts.GenerateID()
			opts := defaultTestRootOptions()
			opts.NodeletConfig.HostID = HostID
			err := execNodeletd(ctx, opts)
			Expect(err).To(BeNil())

			// Check at sunpike-conductor if the Host was updated as expected
			resp, err := conductorClient.ListDetailedHosts(ctx, &api.ListDetailedHostsRequest{})
			Expect(err).To(BeNil())
			Expect(len(resp.Hosts)).To(Equal(len(originalConductorState) + 1))
			var storedHost *sunpikev1alpha1.Host
			for _, storedHost = range resp.Hosts {
				if storedHost.Name == HostID {
					break
				}
			}
			if storedHost == nil {
				Fail(fmt.Sprintf("expected host '%s' not found", storedHost))
				return
			}
			Expect(storedHost.Name).To(Equal(HostID))

			// Make sure that the spec submitted by Nodelet is not persisted.
			Expect(storedHost.Spec).To(BeZero())

			// Read the used config file to understand which values we are expecting.
			cfg, err := config.GetConfigFromFile(defaultConfigPath)
			Expect(err).To(BeNil())

			// Check a few fields in the status which we know the expected value of.
			Expect(storedHost.Status).ToNot(BeZero())
			Expect(storedHost.Status.ClusterRole).To(Equal(cfg.ClusterRole))
			Expect(storedHost.Status.ClusterID).To(Equal(cfg.ClusterID))
			Expect(storedHost.Status.Phases).ToNot(BeZero())
		})

		It("should succeed for an existing Host", func() {
			// Create a new Host
			HostID := hosts.GenerateID()
			opts := defaultTestRootOptions()
			opts.NodeletConfig.HostID = HostID
			opts.NodeletConfig.ClusterRole = constants.RoleWorker
			err := execNodeletd(ctx, opts)
			Expect(err).To(BeNil())

			createdHostResp, err := conductorClient.GetHostStatus(ctx, &api.GetHostStatusRequest{HostUUID: HostID})
			Expect(err).To(BeNil())
			Expect(createdHostResp).ToNot(BeNil())
			Expect(createdHostResp.Host.Status.ClusterRole).To(Equal(opts.NodeletConfig.ClusterRole))

			// Update the host
			UpdatedRole := constants.RoleNone
			updatedOpts := defaultTestRootOptions()
			updatedOpts.NodeletConfig.HostID = HostID
			updatedOpts.NodeletConfig.ClusterRole = UpdatedRole
			err = execNodeletd(ctx, updatedOpts)
			Expect(err).To(BeNil())

			// Check at sunpike-conductor if the existing Host was updated as expected
			resp, err := conductorClient.ListDetailedHosts(ctx, &api.ListDetailedHostsRequest{})
			Expect(err).To(BeNil())
			Expect(len(resp.Hosts)).To(Equal(len(originalConductorState) + 1))
			fetchHostResp, err := conductorClient.GetHostStatus(ctx, &api.GetHostStatusRequest{HostUUID: HostID})
			Expect(err).To(BeNil())
			storedHost := fetchHostResp.Host
			Expect(storedHost).ToNot(BeNil())

			// Make sure that the spec submitted by Nodelet is not persisted.
			Expect(storedHost.Spec).To(BeZero())

			// Read the used config file to understand which values we are expecting.
			cfg, err := config.GetConfigFromFile(defaultConfigPath)
			Expect(err).To(BeNil())

			// Check a few fields in the status which we know the expected value of.
			Expect(storedHost.Status).ToNot(BeZero())
			Expect(storedHost.Status.ClusterRole).To(Equal(UpdatedRole))
			Expect(storedHost.Status.ClusterID).To(Equal(cfg.ClusterID))
			Expect(storedHost.Status.Phases).ToNot(BeZero())
		})
	})

	Context("receiving configuration from Sunpike", func() {

		It("should restart once if config was updated", func() {
			// Setup test dir
			testDir := path.Join(tmpTestDateDir, "config_restart_test")
			err := os.MkdirAll(testDir, 0755)
			Expect(err).To(BeNil())

			// Ensure that a kube.env exists
			sunpikeKubeEnvPath := path.Join(testDir, "kube_sunpike.env")
			Expect(err).To(BeNil())

			// Prepare the initial config in a path where it can be overwritten.
			nodeletConfigPath := path.Join(testDir, "config.yaml")
			err = exec.Command("cp", defaultConfigPath, nodeletConfigPath).Run()
			Expect(err).To(BeNil())

			// Read the used config file to understand which values we are expecting.
			initialHostCfg, err := config.GetConfigFromFile(nodeletConfigPath)
			Expect(err).To(BeNil())

			// Create a Host in Conductor with a new HostSpec/config beforehand
			host := &sunpikev1alpha1.Host{
				ObjectMeta: metav1.ObjectMeta{
					Name: hosts.GenerateID(),
				},
				Spec: sunpikev1alpha1.HostSpec{
					PF9Cfg: sunpikev1alpha1.PF9Opts{
						ClusterRole: constants.RoleMaster,
					},
					ExtraOpts: "--updated-opts",
				},
				Status: sunpikev1alpha1.HostStatus{
					HostState: sunpikev1alpha1.NodeStateUnknown,
				},
			}
			_, err = conductorClient.CreateHost(ctx, &api.CreateHostRequest{
				Host: host,
			})
			Expect(err).To(BeNil())

			// Run nodelet with the same host ID but with another (old) config.
			opts := defaultTestRootOptions()
			opts.NodeletConfig.HostID = host.Name
			opts.ConfigFileOrDirPath = nodeletConfigPath
			opts.NodeletConfig.SunpikeKubeEnvPath = sunpikeKubeEnvPath
			opts.NodeletConfig.SunpikeConfigPath = nodeletConfigPath
			opts.NodeletConfig.DisableLoop = false         // Not disabling loop, because we want to verify that nodelet exits on a config update.
			opts.NodeletConfig.DisableConfigUpdate = false // By default this is disabled.

			// Nodelet should restart on the config update, so it should not be
			// canceled on the context deadline.
			timedCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			err = execNodeletd(timedCtx, opts)
			Expect(err).To(BeNil())
			Expect(timedCtx.Err()).To(BeNil())
			cancel()

			// Nodelet should have picked up the config present in conductor,
			// written it to kube.env and nodelet/config.yaml, and restarted.
			kubeEnvContents, err := ioutil.ReadFile(sunpikeKubeEnvPath)
			Expect(err).To(BeNil())
			expectedKubeEnvContentsBuf := bytes.NewBuffer(nil)
			err = config.ConvertHostToKubeEnvMap(host).ToKubeEnv(expectedKubeEnvContentsBuf)
			Expect(err).To(BeNil())
			// kube.env is not guaranteed to be ordered, so match on collection.
			Expect(strings.Split(string(kubeEnvContents), "\n")).To(ContainElements(strings.Split(expectedKubeEnvContentsBuf.String(), "\n")))

			// Checking if the config.yaml was written correctly.
			// It is a bit of a hacky comparison because simply using GetConfigFromFile
			// will also augment the config with defaults
			updatedCfgContents, err := ioutil.ReadFile(nodeletConfigPath)
			Expect(err).To(BeNil())
			buf := bytes.NewBuffer(nil)
			err = config.ConvertHostToKubeEnvMap(host).ToYAML(buf)
			Expect(strings.Split(string(updatedCfgContents), "\n")).To(ContainElements(strings.Split(buf.String(), "\n")))

			// At the Conductor side, the spec should not have changed, but the
			// status should have been updated.
			fetchedHostResp, err := conductorClient.GetHostStatus(ctx, &api.GetHostStatusRequest{
				HostUUID: host.Name,
			})
			Expect(err).To(BeNil())
			Expect(fetchedHostResp.Host.ObjectMeta).To(Equal(host.ObjectMeta))
			Expect(fetchedHostResp.Host.Spec).To(Equal(host.Spec))
			// Verify that Nodelet did not yet use the received host config in its status update.
			Expect(fetchedHostResp.Host.Status.ClusterRole).To(Equal(initialHostCfg.ClusterRole))

			// Run Nodelet again, which should now pick-up the new config.
			// Nodelet should now not restart anymore, because it picked up and
			// persisted the update in the previous run, so it should be
			// canceled on the context deadline.
			//
			// Note: the timeout should be enough to give nodelet a chance to
			// do a full reconciliation iteration.
			timedCtx, cancel = context.WithTimeout(ctx, 5*time.Second)
			err = execNodeletd(timedCtx, opts)
			Expect(err).ToNot(BeNil())
			Expect(timedCtx.Err()).ToNot(BeNil())
			cancel()

			// Verify that kube.env was not updated after the second run.
			bs, err := ioutil.ReadFile(sunpikeKubeEnvPath)
			Expect(err).To(BeNil())
			Expect(string(bs)).To(Equal(string(kubeEnvContents)))

			// Verify that nodelet/config.yaml was not updated after the second run.
			bs, err = ioutil.ReadFile(nodeletConfigPath)
			Expect(err).To(BeNil())
			Expect(string(bs)).To(Equal(string(updatedCfgContents)))
		})

		It("should not restart if Nodelet receives a config it is already using", func() {
			// Setup test dir
			testDir := path.Join(tmpTestDateDir, "existing_config_no_restart_test")
			err := os.MkdirAll(testDir, 0755)
			Expect(err).To(BeNil())

			// Generate the Host
			host := &sunpikev1alpha1.Host{
				ObjectMeta: metav1.ObjectMeta{
					Name: hosts.GenerateID(),
				},
				Spec: sunpikev1alpha1.HostSpec{
					PF9Cfg: sunpikev1alpha1.PF9Opts{
						ClusterRole: constants.RoleMaster,
					},
					ExtraOpts: "foobar",
				},
				Status: sunpikev1alpha1.HostStatus{
					HostState: sunpikev1alpha1.NodeStateUnknown,
				},
			}
			kubeEnvMap := config.ConvertHostToKubeEnvMap(host)

			// Store the Host-derived kube.env in the kube.env
			kubeEnvPath := path.Join(testDir, "kube.env")
			kubeEnvBuf := bytes.NewBuffer(nil)
			err = kubeEnvMap.ToKubeEnv(kubeEnvBuf)
			Expect(err).To(BeNil())
			err = ioutil.WriteFile(kubeEnvPath, kubeEnvBuf.Bytes(), 0644)
			Expect(err).To(BeNil())

			// Store the Host-derived config in the nodelet/config.yaml
			nodeletConfigPath := path.Join(testDir, "config.yaml")
			nodeletConfigBuf := bytes.NewBuffer(nil)
			err = kubeEnvMap.ToYAML(nodeletConfigBuf)
			Expect(err).To(BeNil())
			err = ioutil.WriteFile(nodeletConfigPath, nodeletConfigBuf.Bytes(), 0644)
			Expect(err).To(BeNil())

			// Create the Host in Conductor
			_, err = conductorClient.CreateHost(ctx, &api.CreateHostRequest{
				Host: host,
			})
			Expect(err).To(BeNil())

			// Configure nodelet
			opts := defaultTestRootOptions()
			opts.NodeletConfig.HostID = hosts.GenerateID()
			opts.ConfigFileOrDirPath = nodeletConfigPath
			opts.NodeletConfig.KubeEnvPath = kubeEnvPath
			opts.NodeletConfig.DisableLoop = false         // Not disabling loop, because we want to verify that nodelet exits on a config update.
			opts.NodeletConfig.DisableConfigUpdate = false // By default this is disabled.

			// Nodelet should not restart on the config update, so it should be
			// canceled on the context deadline.
			// Note: the timeout should be enough to give nodelet a chance to
			// do a full reconciliation iteration.
			timedCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			err = execNodeletd(timedCtx, opts)
			Expect(err).ToNot(BeNil())
			Expect(timedCtx.Err()).ToNot(BeNil())
			cancel()

			// Verify that kube.env was not updated
			bs, err := ioutil.ReadFile(kubeEnvPath)
			Expect(err).To(BeNil())
			Expect(strings.Split(string(bs), "\n")).To(Equal(strings.Split(kubeEnvBuf.String(), "\n")))

			// Verify that nodelet/config.yaml was not updated
			bs, err = ioutil.ReadFile(nodeletConfigPath)
			Expect(err).To(BeNil())
			Expect(strings.Split(string(bs), "\n")).To(Equal(strings.Split(nodeletConfigBuf.String(), "\n")))
		})

		It("should not restart if Nodelet receives an empty config", func() {
			// Setup test dir
			testDir := path.Join(tmpTestDateDir, "empty_config_no_restart_test")
			err := os.MkdirAll(testDir, 0755)
			Expect(err).To(BeNil())

			// Ensure that a kube.env exists
			kubeEnvPath := path.Join(testDir, "kube_sunpike.env")
			err = exec.Command("touch", kubeEnvPath).Run()
			Expect(err).To(BeNil())

			// Prepare the initial config in a path where it can be overwritten.
			nodeletConfigPath := path.Join(testDir, "config.yaml")
			err = exec.Command("cp", defaultConfigPath, nodeletConfigPath).Run()
			Expect(err).To(BeNil())

			// Configure nodelet
			opts := defaultTestRootOptions()
			opts.NodeletConfig.HostID = hosts.GenerateID()
			opts.ConfigFileOrDirPath = nodeletConfigPath
			opts.NodeletConfig.SunpikeKubeEnvPath = nodeletConfigPath
			opts.NodeletConfig.SunpikeConfigPath = nodeletConfigPath
			opts.NodeletConfig.DisableLoop = false         // Not disabling loop, because we want to verify that nodelet exits on a config update.
			opts.NodeletConfig.DisableConfigUpdate = false // By default this is disabled.

			// Nodelet should not restart on the config update, so it should be
			// canceled on the context deadline.
			// Note: the timeout should be enough to give nodelet a chance to
			// do a full reconciliation iteration.
			timedCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			err = execNodeletd(timedCtx, opts)
			Expect(err).ToNot(BeNil())
			Expect(timedCtx.Err()).ToNot(BeNil())
			cancel()

			// Verify that kube.env was not updated
			bs, err := ioutil.ReadFile(kubeEnvPath)
			Expect(err).To(BeNil())
			Expect(bs).To(HaveLen(0))

			// Verify that nodelet/config.yaml was not updated
			expectedContents, err := ioutil.ReadFile(defaultConfigPath)
			Expect(err).To(BeNil())
			actualContents, err := ioutil.ReadFile(nodeletConfigPath)
			Expect(err).To(BeNil())
			Expect(string(actualContents)).To(Equal(string(expectedContents)))
		})
	})
})

func defaultTestRootOptions() *nodeletcmd.RootOptions {
	return &nodeletcmd.RootOptions{
		Debug:               true,
		ConfigFileOrDirPath: defaultConfigPath,
		NodeletConfig: config.Config{
			PhaseScriptsDir:     "../../root/opt/pf9/pf9-kube",
			DisableScripts:      true,
			DisableExtFile:      true,
			DisableLoop:         true,
			DisableConfigUpdate: true,
		},
	}
}

func execNodeletd(ctx context.Context, opts *nodeletcmd.RootOptions) error {
	// Convert RootOptions to flags
	var flags []string
	opts.Flags().VisitAll(func(flag *pflag.Flag) {
		val := reflect.Indirect(reflect.ValueOf(flag.Value))
		if !val.IsZero() {
			flags = append(flags, fmt.Sprintf("--%s=%s", flag.Name, flag.Value.String()))
		}
	})

	cmd := command(ctx, nodeletdPath, flags...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func startSunpikeConductorInDocker(ctx context.Context, containerName string, image string) {
	cmd := command(ctx, "docker", "run",
		"-d",
		"--name", containerName,
		"-p=9111:9111",
		image,
		"serve",
		"--addr=0.0.0.0:9111",
		"--storage=inmemory",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	_ = cmd.Run()
}

func stopAndRemoveDockerContainer(ctx context.Context, containerID string) error {
	out, err := command(ctx, "docker", "stop", containerID).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to stop Docker container: %s", string(out))
	}

	out, err = command(ctx, "docker", "rm", containerID).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to remove Docker container: %s", string(out))
	}
	return nil
}

func ensureCommandExists(name string) {
	_, err := command(context.Background(), "which", name).CombinedOutput()
	if err != nil {
		Fail(fmt.Sprintf("command '%s' is required for test suite", name))
	}
}

func getConductorClient() api.Client {
	conn, err := grpc.Dial("localhost:9111", grpc.WithInsecure())
	Expect(err).To(BeNil())
	return api.NewClient(conn)
}

// getHostIP gets the local IP of the current host. (e.g., 192.168.1.157)
//
// Bash command is adapted from https://stackoverflow.com/questions/13322485/how-to-get-the-primary-ip-address-of-the-local-machine-on-linux-and-os-x
// It should work on both MacOS and Linux.
func getHostIP(ctx context.Context) string {
	out, err := command(ctx, "bash", "-c", "ifconfig | grep -Eo 'inet (addr:)?([0-9]*\\.){3}[0-9]*' | grep -Eo '([0-9]*\\.){3}[0-9]*' | grep -v '127.0.0.1' | head -n 1").CombinedOutput()
	if err != nil {
		panic(err)
	}

	return strings.TrimSpace(string(out))
}

func command(ctx context.Context, cmd string, args ...string) *exec.Cmd {
	logf("Running external command: %s %s", cmd, strings.Join(args, " "))
	return exec.CommandContext(ctx, cmd, args...)
}

func logf(s string, args ...interface{}) {
	fmt.Fprintf(GinkgoWriter, "  "+s+"\n", args...)
}

func getEnvOrDefault(envKey string, defaultVal string) string {
	envVal := os.Getenv(envKey)
	if envVal == "" {
		return defaultVal
	}
	return envVal
}

func ensureDockerImageExists(ctx context.Context, image string) {
	err := command(ctx, "docker", "image", "inspect", image).Run()
	Expect(err).To(BeNil(), fmt.Sprintf("Expected Docker image not found: %s", image))
}
