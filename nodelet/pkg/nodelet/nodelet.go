package nodelet

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"strconv"
	"time"

	"github.com/ghodss/yaml"
	"go.uber.org/zap"
	"github.com/platform9/nodelet/nodelet/pkg/pf9kube"

	"github.com/platform9/nodelet/nodelet/pkg/phases"
	"github.com/platform9/nodelet/nodelet/pkg/utils/config"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	"github.com/platform9/nodelet/nodelet/pkg/utils/extensionfile"
	"github.com/platform9/nodelet/nodelet/pkg/utils/fileio"
	"github.com/platform9/nodelet/nodelet/pkg/utils/sunpikeutils"

	sunpikev1alpha1 "github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike/v1alpha1"
)

// Following 2 lines are needed to simplify unit tests
var getFileIO = fileio.New
var loadRolePhases = phases.InitAndLoadRolePhases

type Nodelet struct {
	phases       []phases.PhaseInterface
	sunpike      *sunpikeutils.Wrapper
	config       *config.Config
	log          *zap.SugaredLogger
	currentState *extensionfile.ExtensionData
}

func (n *Nodelet) Run(ctx context.Context) error {
	n.log.Info("Starting nodelet...")
	err := pf9kube.Extract()
	if err != nil {
		n.log.Errorf("Failed to extract pf9-kube: %v", err)
		return fmt.Errorf("failed to extract pf9-kube: %v", err)
	}
	// Do an initial persist + config check here to ensure that the Host makes
	// itself known to Sunpike, even if things go wrong in the reconciling itself.
	err = n.persistStatusAndUpdateConfigIfChanged(ctx)
	if err != nil {
		n.log.Errorf("Failed to perform the initial status update: %v", err)
	}

	// Now that we've sent one `converging` state for the node - we can safely
	// reset the node state
	if n.config.ClusterID == "" {
		n.currentState.NodeState = constants.OkState
		n.currentState.StartAttempts = 0
	}

	// Configure the kube.env symlink to point to either the sunpike or resmgr kube.env.
	err = n.setKubeEnvSymlink()
	if err != nil {
		n.log.Errorf("Failed to create kube.env symlink: %v", err)
	}

	// Set the initial timer to 0 to immediately trigger the first reconciliation.
	poller := time.After(0)
	pollInterval := time.Duration(n.config.LoopInterval) * time.Second

	n.log.Infof("Starting nodelet reconciliation loop")
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-poller:
			err := n.Reconcile(ctx)
			if err != nil {
				n.log.Errorf("Failed to reconcile host: %v", err)
			}

			if n.config.DisableLoop {
				n.log.Warn("Looping is disabled; not entering the reconciliation loop.")
				return nil
			}

			n.log.Debugf("Loop iteration completed done. Sleeping for %v before next iteration of Reconcile",
				pollInterval)
			// Not using a time.Ticker because the old behaviour of an exact
			// delay between iterations should be preserved
			poller = time.After(pollInterval)
		}
	}
}

func CreateNodeletFromConfig(ctx context.Context, cfg *config.Config) (*Nodelet, error) {
	log := zap.S()
	phases, err := loadRolePhases(ctx, *cfg)
	if err != nil {
		// Phases could not be loaded. Cannot recover from this.
		return nil, fmt.Errorf("could not load phases: %v", err)
	}

	state, err := constructInitialCurrentState(cfg, phases)
	if err != nil {
		return nil, err
	}

	log.Info("Fetch the current config from the filesystem and try to infer the current HostSpec to send to Sunpike.")
	var host *sunpikev1alpha1.Host
	kubeEnvCfg, err := fetchKubeEnvMapFromConfigFile(cfg.SunpikeConfigPath)
	if err != nil {
		log.Warnf("Failed to fetch nodelet config from file: %v", err)
	} else {
		// Getting the host is on a best effort basis; some fields might not be
		// decoded correctly because we lack the type information in the kube.env
		// representation (there, everything is a string).
		host, err = kubeEnvCfg.ToHost()
		if err != nil {
			log.Warnf("Could not convert initial nodelet config to Host object: %v", err)
			host = &sunpikev1alpha1.Host{}
		}
	}

	// Configure Sunpike client
	sunpike, err := sunpikeutils.InitOrGetSunpikeClient(phases, *cfg, host.Spec)
	if err != nil {
		log.Errorf("Error creating sunpike client for sunpike server at '%s': %v. Continuing without Sunpike.", cfg.TransportURL, err)
	}

	return &Nodelet{
		phases:       phases,
		currentState: state,
		config:       cfg,
		log:          log,
		sunpike:      sunpike,
	}, nil
}

func getStatusFailureToleration(cfg config.Config) time.Duration {
	thresholdInt := cfg.PF9StatusThresholdSeconds
	duration, err := time.ParseDuration(fmt.Sprintf("%ds", thresholdInt))
	if err != nil {
		zap.S().Warnf("Value: %d cannot be converted to time in seconds. Falling back to threshold of 0s", thresholdInt)
		return time.Duration(0)
	}
	return duration
}

func (n *Nodelet) Reconcile(ctx context.Context) error {
	expectedState := n.config.KubeServiceState
	n.log.Infof("Reconciling to expected service state: %s", expectedState)

	// NOTE: Be careful when using ctx in this function. This function deals with creation of k8s
	// cluster. Pass ctx to underlying functions when it is OK for the operation to be killed.
	// For operations that should not be killed pass a new context.Background() instead of ctx
	n.Status(context.Background())

	switch expectedState {
	case constants.ServiceStateTrue:
		n.handleServiceStartState(ctx)
	case constants.ServiceStateFalse:
		n.handleServiceStopState(ctx)
	default:
		n.log.Infof("Unknown service state %s. Not reconciling...", expectedState)
	}

	if n.config.DisableConfigUpdate {
		n.tryToPersistStatus(ctx)
		return nil
	} else {
		err := n.persistStatusAndUpdateConfigIfChanged(ctx)
		if err != nil {
			return err
		}
	}

	n.log.Info("Reconcile completed.")
	return nil
}

func (n *Nodelet) handleServiceStartState(ctx context.Context) {
	failedCheck := n.currentState.FailedStatusCheck
	if n.config.DisableScripts {
		n.log.Warnf("Running scripts is disabled; not running start scripts.")
		return
	}
	n.currentState.Operation = constants.StartOp
	expectedRole := n.config.ClusterRole
	expectedID := n.config.ClusterID
	if n.currentState.KubeRunning &&
		(n.currentState.ClusterID == expectedID &&
			n.currentState.ClusterRole == expectedRole ||
			(n.currentState.ClusterRole == constants.RoleNone && n.currentState.ClusterID == "")) {
		n.log.Infof("pf9-kube is already running...")
		n.log.Debugf("Resetting start attempt counter")
		n.currentState.StartAttempts = 0
		n.currentState.ServiceState = constants.ServiceStateTrue
		n.currentState.StartFailStep = -1
		n.currentState.CompletedPhases = n.currentState.AllPhases
	} else {
		n.currentState.StartAttempts++
		if n.currentState.StartAttempts%n.config.FullRetryCount == 0 {
			// If failed to start service 10 times perform a complete stop and start.
			n.log.Warnf("Performing complete restart of the service. Retry attempt: %d",
				n.currentState.StartAttempts)
			failedCheck = 0
		}
		n.Stop(ctx, failedCheck, false)
		n.currentState.ClusterID = expectedID
		n.currentState.ClusterRole = expectedRole
		// To prevent this operation from being killed pass a new context.Background() instead of ctx
		failedStep, err := n.Start(context.Background(), failedCheck)
		if err != nil {
			failedPhase := n.phases[failedStep]
			n.currentState.ServiceState = constants.ServiceStateFalse
			n.currentState.StartFailStep = failedStep
			n.log.Warnf("Failed to start kube service at step: %s", failedPhase.GetPhaseName())
		} else {
			n.currentState.StartAttempts = 0
			n.currentState.ServiceState = constants.ServiceStateTrue
			n.currentState.LastFailedPhase = ""
			n.currentState.StartFailStep = -1
		}
	}
	n.tryToPersistStatus(ctx)
}

func (n *Nodelet) handleServiceStopState(ctx context.Context) {
	n.currentState.Operation = constants.StopOp
	if n.config.DisableScripts {
		n.log.Warnf("Running scripts is disabled; not running stop scripts.")
		return
	}

	if !n.currentState.KubeRunning {
		n.log.Infof("pf9-kube is already stopped...")
	} else {
		n.log.Infof("pf9-kube is running or partially stopped. Running stop now...")
		n.Stop(ctx, 0, false)
		n.currentState.StartFailStep = -1
	}
}

func (n *Nodelet) Status(ctx context.Context) {
	var failedStatusCheck int = -1
	n.currentState.CurrentStatusCheckTime = time.Now().Unix()
	n.currentState.Operation = constants.StatusOp
	n.currentState.KubeRunning = false

	// special case for incomplete start
	if n.currentState.StartFailStep != -1 {
		n.currentState.CurrentStatusCheck = ""
		n.tryToPersistStatus(ctx)
		n.currentState.FailedStatusCheck = n.currentState.StartFailStep
		return
	}

	for i, phase := range n.phases {
		if n.config.DisableScripts {
			n.log.Warnf("Running scripts is disabled; not running status scripts.")
			break
		}

		phasename := phase.GetPhaseName()
		n.log.Infof("Running status check: %s", phasename)
		n.currentState.CurrentStatusCheck = phasename
		err := phase.Status(ctx, *n.config)
		if err != nil {
			failedStatusCheck = i
			n.currentState.LastFailedCheck = phasename
			n.currentState.LastFailedCheckTime = time.Now().Unix()
			break
		}
	}
	n.currentState.CurrentStatusCheck = ""
	n.tryToPersistStatus(ctx)
	if failedStatusCheck == -1 {
		// All status checks executed as expected.
		n.currentState.KubeRunning = true
		failedStatusCheck = 0
		n.currentState.LastSuccessfulStatus = time.Now()
	} else if n.currentState.ServiceState == constants.ServiceStateTrue {
		// PMK-2434
		delta := time.Since(n.currentState.LastSuccessfulStatus)
		threshold := getStatusFailureToleration(*n.config)
		if delta < threshold {
			n.currentState.KubeRunning = true
			failedStatusCheck = 0
			n.log.Warnf("The time between now and pf9-kube status returning an exit code of 0 (%s) is shorter than the configured threshold of (%s). Returning exit code of 0", delta.String(), threshold)
		} else if threshold != time.Duration(0) {
			// if threshold is actually configured then print the warning about returning actual exit code
			n.log.Warnf("The time between now and pf9-kube status returning an exit code of 0 (%s) is longer than the configured threshold of (%s). Returning actual exit code", delta.String(), threshold)
		}
	} else {
		// status check failed i.e. returned non-zero exit code and service is not expected to be running.
		// non-zero exit code is expected in this case. So treating it as successful status check for failed status ignore threshold calculations.
		failedStatusCheck = 0
		n.currentState.LastSuccessfulStatus = time.Now()
	}
	n.currentState.FailedStatusCheck = failedStatusCheck
}

// Start attempts to start the Kubernetes service on the host, running through the phases, starting from startPhaseIndex.
//
// If successful, the lastPhase will be the len(phases), the error will be nil.
// If failed, the lastPhase will be the phase at which the start failed, and error will be non-nil.
func (n *Nodelet) Start(ctx context.Context, startPhaseIndex int) (lastPhase int, err error) {
	n.currentState.CompletedPhases = n.currentState.AllPhases[0:startPhaseIndex]
	n.log.Infof("Running start chain from script: %+v",
		n.phases[startPhaseIndex])
	for startPhaseIndex < len(n.phases) {
		phase := n.phases[startPhaseIndex]
		phasename := phase.GetPhaseName()
		n.currentState.CurrentPhase = phasename
		n.tryToPersistStatus(ctx)
		err = phase.Start(ctx, *n.config)
		if err != nil {
			n.currentState.LastFailedPhase = phasename
			n.tryToPersistStatus(ctx)
			break
		}
		n.currentState.CompletedPhases = append(
			n.currentState.CompletedPhases, phasename)
		n.tryToPersistStatus(ctx)
		startPhaseIndex++
	}
	n.currentState.CurrentPhase = ""
	n.tryToPersistStatus(ctx)
	return startPhaseIndex, err
}

// SkipGenCertsPhase sets the required flags to make Stop operation skip the gen_certs phase by satisfying FailedStatusCheck != idx condition
func (n *Nodelet) SkipGenCertsPhase() {
	n.currentState.Operation = constants.StartOp
	n.currentState.FailedStatusCheck = -1
}

// Stop attempts to stop the Kubernetes service on the host, running through the phases in reverse, starting from stopPhaseIndex.
func (n *Nodelet) Stop(ctx context.Context, stopPhaseIndex int, force bool) error {
	var err error
	n.log.Infof("Running stop chain till script: %+v",
		n.phases[stopPhaseIndex])
	for idx := len(n.phases) - 1; idx >= stopPhaseIndex; idx-- {
		phase := n.phases[idx]
		if n.currentState.Operation == constants.StartOp && phase.GetHostPhase().Order == int32(constants.GenCertsPhaseOrder) {
			// Restarting kube stack (stop invoked but the operation is set to start)
			// and currently processing cert generation phase.
			// Cert generation is not be skipped if it failed status check.
			if n.currentState.FailedStatusCheck != idx {
				return nil
			}
		}
		err = phase.Stop(ctx, *n.config)
		if err != nil {
			if force {
				n.log.Warnf("failed to stop phase %d, continuing the stop chain as force is set", idx)
			} else {
				break
			}
		}
	}
	n.currentState.ServiceState = constants.ServiceStateFalse
	// Only a subset of phases can be considered as completed.
	n.currentState.CompletedPhases = n.currentState.AllPhases[0:stopPhaseIndex]
	if stopPhaseIndex == 0 {
		// service was completely stopped
		n.currentState.StartAttempts = 0
		n.currentState.CompletedPhases = []string{}
		n.currentState.ClusterID = ""
		n.currentState.ClusterRole = ""
		n.currentState.CurrentPhase = ""
		n.currentState.LastFailedPhase = ""
	}
	n.tryToPersistStatus(ctx)
	return err
}

// IsK8sRunning return the current state of k8s stack as recorded in nodelet
func (n *Nodelet) IsK8sRunning() bool {
	return n.currentState.KubeRunning
}

// NumPhases returns the number of phases needed to configure the node as a k8s node
func (n *Nodelet) NumPhases() int {
	return len(n.phases)
}

// StartSinglePhase executes the "start" function of the provided phase	only
func (n *Nodelet) StartSinglePhase(ctx context.Context, idx int) error {
	phase := n.phases[idx]
	phase.Start(ctx, *n.config)
	if phase.GetHostPhase().Status == constants.FailedState {
		return fmt.Errorf("failed to start phase %s", phase.GetPhaseName())
	}
	return nil
}

// StopSinglePhase executes the "stop" function of the provided phase only
func (n *Nodelet) StopSinglePhase(ctx context.Context, idx int) error {
	phase := n.phases[idx]
	phase.Stop(ctx, *n.config)
	if phase.GetHostPhase().Status != constants.StoppedState {
		return fmt.Errorf("failed to cleanly stop phase: %s", phase.GetPhaseName())
	}
	return nil
}

// tryToPersistStatus persists on a best-effort basis; any errors will not be propagated
func (n *Nodelet) tryToPersistStatus(ctx context.Context) {
	if n.config.DisableExtFile {
		n.log.Warnf("State persisting to file is disabled; not writing to extension file.")
	} else {
		n.currentState.Write()
	}

	if n.config.DisableSunpike {
		n.log.Warnf("Sunpike communication is disabled; not sending an update to sunpike-conductor.")
	} else {
		n.log.Infof("Submitting status update to Sunpike: %s", n.config.TransportURL)
		_, err := n.sunpike.Update(ctx, *n.currentState)
		if err != nil {
			n.log.Warnf(err.Error())
		}
	}
}

func (n *Nodelet) persistStatusAndUpdateConfigIfChanged(ctx context.Context) error {
	if n.config.DisableExtFile {
		n.log.Warnf("State persisting to file is disabled; not writing to extension file.")
	} else {
		n.currentState.Write()
	}

	if n.config.DisableSunpike {
		n.log.Warnf("Sunpike communication is disabled; not sending an update to sunpike-conductor.")
		// Generate kube.env from config_sunpike.yaml.
		// TODO (pacharya): make the config path customizable
		kubeEnvCfg, err := fetchKubeEnvMapFromConfigFile(n.config.SunpikeConfigPath)
		if err != nil {
			return fmt.Errorf("failed to fetch nodelet config from file: %w", err)
		}
		err = os.Remove(n.config.ResmgrKubeEnvPath)
		if err != nil {
			n.log.Debugf("Could not delete old config %s: %v", n.config.ResmgrKubeEnvPath, err)
		}
		err = n.writeEnvMapToKubeEnvFile(kubeEnvCfg, n.config.ResmgrKubeEnvPath)
		if err != nil {
			return fmt.Errorf("failed to write to updated config to kube.env: %w", err)
		}
		return nil
	}

	n.log.Infof("Submitting status update to Sunpike and checking it for new config: %s", n.config.TransportURL)
	updatedHost, err := n.sunpike.Update(ctx, *n.currentState)
	if err != nil {
		return err
	}

	return n.handleConfigUpdate(updatedHost)
}

func (n *Nodelet) handleConfigUpdate(updatedHost *sunpikev1alpha1.Host) error {
	if reflect.DeepEqual(updatedHost.Spec, sunpikev1alpha1.HostSpec{}) {
		n.log.Info("Ignoring received HostSpec from Sunpike because it is empty")
		return nil
	}
	n.log.Info("Handling received HostSpec from Sunpike.")

	// Convert HostSpec to Nodelet Config to compare it with the existing one.
	kubeEnvMap := config.ConvertHostToKubeEnvMap(updatedHost)
	updatedCfg, err := kubeEnvMap.ToConfig()
	if err != nil {
		return fmt.Errorf("error converting updated config to Nodelet config format: %v", err)
	}

	// Collect the necessary config.
	kubeEnvCfg, err := fetchKubeEnvMapFromConfigFile(n.config.SunpikeConfigPath)
	if err != nil {
		n.log.Warnf("Failed to fetch nodelet config from file: %v", err)
	}
	_, sunpikeKubeEnvErr := os.Stat(n.config.SunpikeKubeEnvPath)

	// If there is no change in the configs and the kube_sunpike.env still
	// exists, no need to update the config files. We use the kubeEnvMap
	// representations because we can convert from both Hosts and the config files.
	n.log.Debugf("Comparing existing config with Sunpike-received config")
	n.log.Debugf("Received config: %+v", kubeEnvMap)
	n.log.Debugf("Current config: %+v", kubeEnvCfg)
	if sunpikeKubeEnvErr == nil && reflect.DeepEqual(kubeEnvMap, kubeEnvCfg) {
		n.log.Info("Received config from Sunpike has not changed compared to the current config.")
		return nil
	}

	n.log.Infof("Config update detected! Writing new config to kube.env and nodelet config: %+v", updatedCfg)

	// Try to write updated config to kube.env
	err = n.writeEnvMapToKubeEnvFile(kubeEnvMap, n.config.SunpikeKubeEnvPath)
	if err != nil {
		n.log.Warnf("Failed to write to updated config to kube.env: %v", err)
	}

	// Try to write updated config to nodelet/config.yaml
	err = n.writeEnvMapToNodeletConfigFile(kubeEnvMap)
	if err != nil {
		n.log.Warnf("Failed to write to updated config to nodelet config file: %v", err)
	}

	// The config has changed, so Nodelet should restart
	if n.config.DisableExitOnUpdate {
		n.log.Warn("Exiting on Sunpike config updates is disabled; not picking up new configuration until Nodelet is restarted!")
	} else {
		n.log.Warn("Exiting Nodelet to trigger a restart, to pick up the new config.")
		os.Exit(0)
	}

	return nil
}

func (n *Nodelet) writeEnvMapToKubeEnvFile(kubeEnvMap config.KubeEnvMap, kubeEnvPath string) error {
	fd, err := os.OpenFile(kubeEnvPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer fd.Close()
	fd.WriteString(fmt.Sprintf("# %s\n", constants.GeneratedFileHeader))
	return kubeEnvMap.ToKubeEnv(fd)
}

func (n *Nodelet) writeEnvMapToNodeletConfigFile(kubeEnvMap config.KubeEnvMap) error {
	fd, err := os.OpenFile(n.config.SunpikeConfigPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer fd.Close()
	fd.WriteString(fmt.Sprintf("# %s\n", constants.GeneratedFileHeader))
	return kubeEnvMap.ToYAML(fd)
}

func (n *Nodelet) setKubeEnvSymlink() error {
	// Set the kube.env symlink to the appropriate kube_*.env
	err := os.Remove(n.config.KubeEnvPath)
	if err != nil {
		n.log.Debugf("Could not delete symlink %s: %v", n.config.KubeEnvPath, err)
	}
	var dst string
	if _, err := os.Stat(n.config.SunpikeKubeEnvPath); n.config.DisableSunpike || err != nil {
		dst = n.config.ResmgrKubeEnvPath
	} else {
		dst = n.config.SunpikeKubeEnvPath
	}
	err = os.Symlink(dst, n.config.KubeEnvPath)
	if err != nil {
		return fmt.Errorf("failed to create kube.env symlink: %w", err)
	}
	n.log.Infof("Set %s symlink to point to %s.", n.config.KubeEnvPath, dst)
	return nil
}

// ListPhases returns a list of strings with details of all phases to be displayed as a table
func (n *Nodelet) ListPhases() [][]string {
	phasesList := [][]string{}
	for i := 0; i < len(n.phases); i++ {
		phase := n.phases[i]
		// Increment the index by 1 to keep it consistent across UI and human readable
		row := []string{strconv.Itoa(i + 1), phase.GetPhaseName()}
		phasesList = append(phasesList, row)
	}
	return phasesList
}

// PhasesStatus returns a list of strings with details of phases with their current status to be displayed as a table
func (n *Nodelet) PhasesStatus() [][]string {
	phasesList := [][]string{}
	statusString := ""
	for i := 0; i < len(n.phases); i++ {
		phase := n.phases[i]
		hostPhase := phase.GetHostPhase()
		statusString = hostPhase.Status
		// Increment the index by 1 to keep it consistent across UI and human readable
		row := []string{strconv.Itoa(i + 1), phase.GetPhaseName(), statusString}
		phasesList = append(phasesList, row)
	}
	return phasesList
}

func fetchKubeEnvMapFromConfigFile(configPath string) (config.KubeEnvMap, error) {
	kubeEnv := config.KubeEnvMap{}
	_, err := os.Stat(configPath)
	if err != nil {
		return kubeEnv, nil
	}

	contents, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("error reading config: %v", err)
	}

	err = yaml.Unmarshal(contents, &kubeEnv)
	if err != nil {
		return nil, fmt.Errorf("error parsing config to kubeEnvMap: %v", err)
	}
	return kubeEnv, nil
}

func constructInitialCurrentState(cfg *config.Config, priorityPhases []phases.PhaseInterface) (*extensionfile.ExtensionData, error) {
	state := extensionfile.New(getFileIO(), cfg.ExtensionOutputFile, cfg)
	if cfg.DisableExtFile {
		zap.S().Warnf("State persisting to file is disabled; not reading from extension file.")
	} else {
		state.Load()
	}

	var allPhaseNames []string
	var allStatusCheckNames []string
	for i := 0; i < len(priorityPhases); i++ {
		phase := priorityPhases[i]
		phasename := phase.GetPhaseName()
		allPhaseNames = append(allPhaseNames, phasename)
		allStatusCheckNames = append(allStatusCheckNames, phasename)
	}
	state.CompletedPhases = []string{}
	state.StartFailStep = -1
	state.AllPhases = allPhaseNames
	state.AllStatusChecks = allStatusCheckNames
	state.Operation = constants.StopOp
	state.ServiceState = constants.ServiceStateFalse
	state.Cfg = cfg
	return &state, nil
}
