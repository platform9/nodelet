package sunpikeutils

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/erwinvaneyk/goversion"
	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/platform9/nodelet/pkg/phases"
	"github.com/platform9/nodelet/pkg/utils/config"
	"github.com/platform9/nodelet/pkg/utils/constants"
	"github.com/platform9/nodelet/pkg/utils/extensionfile"
	sunpikev1alpha1 "github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike/v1alpha1"
	"github.com/platform9/pf9-qbert/sunpike/conductor/pkg/api"
)

var doOnce sync.Once
var singleton *Wrapper
var initError error

// Wrapper struct encapsulates api.Host object to simplify sunpike communication
type Wrapper struct {
	Host           *sunpikev1alpha1.Host
	conn           api.ConductorClient
	connectTimeout time.Duration
	config         config.Config
}

// GetOrderForPhaseName is a convenience function to convert phase name to phase order
// This function is needed because pre-sunpike nodelet was entirely using phase names for UI simplicity.
// The new data model however uses phase order since it is a unique identifier
func (c *Wrapper) GetOrderForPhaseName(name string) int32 {
	if name == "" {
		// Empty string is used to denote no current phase or no current phase check in progress.
		return -1
	}
	for _, phase := range c.Host.Status.Phases {
		if phase.Name == name {
			return phase.Order
		}
	}
	return -1
}

// InitOrGetSunpikeClient returns an instance of the sunpike.Wrapper
//
// cfg contains the configuration needed to setup the Sunpike client, such as
// transport URL and the connection timeout. hostSpec on the other hand is
// optional and is what this Sunpike client will send to Nodelet as the Spec
// as part of the status update.
//
// The idea is to also incorporate the settings of cfg into the HostSpec. After
// merging those we can remove the need for the cfg argument.
// TODO merge cfg and hostSpec at some point
func InitOrGetSunpikeClient(phases []phases.PhaseInterface, cfg config.Config, hostSpec sunpikev1alpha1.HostSpec) (*Wrapper, error) {
	doOnce.Do(func() {
		var err error
		singleton = &Wrapper{}
		singleton.connectTimeout = time.Duration(cfg.ConnectTimeout) * time.Second
		singleton.config = cfg
		singleton.Host = &sunpikev1alpha1.Host{}
		singleton.Host.Spec = hostSpec
		singleton.Host.ObjectMeta = createHostMetadata(cfg.HostID)
		singleton.Host.Status = sunpikev1alpha1.HostStatus{}
		singleton.Host.Status.Phases, singleton.Host.Status.AllStatusChecks = convertOldPhaseToSunpikePhase(phases)
		singleton.Host.Status.ClusterRole = cfg.ClusterRole
		singleton.Host.Status.ClusterID = cfg.ClusterID
		singleton.Host.Status.StartAttempts = 0
		singleton.Host.Status.ServiceState = false
		singleton.Host.Status.LastFailedCheck = -1
		singleton.Host.Status.LastFailedPhase = -1
		singleton.Host.Status.PhaseCompleted = -1
		singleton.Host.Status.Hostname = getHostName()
		singleton.Host.Status.Nodelet = sunpikev1alpha1.NodeletStatus{
			Version: goversion.Get().Version,
		}
		if cfg.DisableSunpike {
			zap.S().Infof("Disabling sunpike communication as per config setting for DISABLE_SUNPIKE option")
			return
		}
		err = singleton.genOrGetSunpikeConn()
		if err != nil {
			initError = fmt.Errorf("could not initialize the sunpike client: %v", err)
			return
		}
	})
	return singleton, initError
}

// Update is a convenience function to convert extension data into api.Host object and report it to sunpike
func (c *Wrapper) Update(ctx context.Context, extnData extensionfile.ExtensionData) (*sunpikev1alpha1.Host, error) {
	// Having this condition here keeps rest of code simple.
	if c.conn == nil {
		// try to create a connection once more
		err := c.genOrGetSunpikeConn()
		if err != nil {
			return nil, fmt.Errorf("skipping update to sunpike as client could not be initialized: %w", err)
		}
	}

	c.populateHostObj(extnData)
	ctx, cancel := context.WithTimeout(ctx, c.connectTimeout)
	defer cancel()
	callOptions := c.getGRPCOptions()
	resp, err := c.conn.UpdateHostStatus(ctx, &api.UpdateHostStatusRequest{Status: c.Host}, callOptions...)
	if err != nil {
		return nil, fmt.Errorf("error sending status update to sunpike: %v", err)
	}
	return resp.Host, nil
}

func (c *Wrapper) populatePhaseObj(order int32, op, status string) {
	var phase *sunpikev1alpha1.HostPhase
	for _, p := range c.Host.Status.Phases {
		if p.Order == order {
			phase = &p
			break
		}
	}
	if phase != nil {
		phase.Operation = op
		phase.Status = status
		phase.StartedAt = sunpikev1alpha1.NewTime(time.Now())
	}
}

func (c *Wrapper) populateHostObj(extnData extensionfile.ExtensionData) {
	spHost := c.Host
	currentPhaseOrder := c.GetOrderForPhaseName(extnData.CurrentPhase)
	failedPhaseOrder := c.GetOrderForPhaseName(extnData.LastFailedPhase)
	var latestCompletedPhaseOrder int32 = -1
	spHost.Status.StartAttempts = int32(extnData.StartAttempts)
	if len(extnData.CompletedPhases) > 0 {
		latestCompletedPhaseOrder = c.GetOrderForPhaseName(extnData.CompletedPhases[len(extnData.CompletedPhases)-1])
	}
	spHost.Status.LastFailedCheckTime = extnData.LastFailedCheckTime
	spHost.Status.CurrentStatusCheckTime = extnData.CurrentStatusCheckTime
	spHost.Status.ServiceState = strings.ToLower(extnData.ServiceState) == constants.ServiceStateTrue
	spHost.Status.CurrentStatusCheck = c.GetOrderForPhaseName(extnData.CurrentStatusCheck)
	spHost.Status.LastFailedCheck = c.GetOrderForPhaseName(extnData.LastFailedCheck)
	switch extnData.NodeState {
	case constants.OkState:
		spHost.Status.HostState = sunpikev1alpha1.NodeStateOk
	case constants.ConvergingState:
		spHost.Status.HostState = sunpikev1alpha1.NodeStateConverging
	case constants.RetryingState:
		spHost.Status.HostState = sunpikev1alpha1.NodeStateRetrying
	case constants.ErrorState:
		spHost.Status.HostState = sunpikev1alpha1.NodeStateFailed
	default:
		spHost.Status.HostState = sunpikev1alpha1.NodeStateFailed
	}
	if currentPhaseOrder != -1 {
		c.populatePhaseObj(currentPhaseOrder, extnData.Operation, constants.ExecutingState)
	}
	if failedPhaseOrder != -1 {
		c.populatePhaseObj(failedPhaseOrder, extnData.Operation, constants.FailedState)
	}
	if latestCompletedPhaseOrder != -1 {
		for _, p := range spHost.Status.Phases {
			if p.Order <= latestCompletedPhaseOrder {
				c.populatePhaseObj(p.Order, extnData.Operation, constants.RunningState)
			} else if p.Status != constants.NotStartedState {
				c.populatePhaseObj(p.Order, extnData.Operation, constants.StoppedState)
			}
		}
	} else if !spHost.Status.ServiceState {
		for _, p := range spHost.Status.Phases {
			c.populatePhaseObj(p.Order, extnData.Operation, constants.StoppedState)
		}
	}
}

func (c *Wrapper) genOrGetSunpikeConn() error {
	if c.conn != nil {
		return nil
	}
	var err error
	c.conn, err = createSunpikeClient(c.config)
	if err != nil {
		return err
	}
	return nil
}

func (c *Wrapper) getGRPCOptions() []grpc.CallOption {
	/*
	* Configures max retries to be cfg.GRPCRetryCount with deadline of each attempt will be now + cfg.GRPCRetryTimeout
	* Retries will only be attempted on "Aborted", "Unavailable", "Cancelled" and "DeadlineExceeded" status codes.
	* "WithPerRetryTimeout" handles the "Cancelled" and "DeadlineExceeded" codes.
	 */
	grpcOpts := []grpc.CallOption{
		grpc_retry.WithMax(c.config.GRPCRetryMax),
		grpc_retry.WithPerRetryTimeout(time.Duration(c.config.GRPCRetryTimeoutSeconds) * time.Second),
		grpc_retry.WithCodes(codes.Aborted, codes.Unavailable),
	}
	return grpcOpts
}

func convertOldPhaseToSunpikePhase(phases []phases.PhaseInterface) ([]sunpikev1alpha1.HostPhase, []int32) {
	spPhases := make([]sunpikev1alpha1.HostPhase, len(phases))
	var allChecks []int32
	for i, phase := range phases {
		spPhases[i] = phase.GetHostPhase()
		order := int32(spPhases[i].Order)
		allChecks = append(allChecks, order)
	}
	return spPhases, allChecks
}

func getHostName() string {
	hostname, err := os.Hostname()
	if err != nil {
		zap.S().Warnf("Could not fetch hostname: %v", err)
		return ""
	}
	return hostname
}

func createHostMetadata(hostID string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name: hostID,
		Labels: map[string]string{
			// We duplicate hostname status field in the labels to allow for
			// easier querying.
			"hostname": getHostName(),
		},
	}
}

func createSunpikeClient(cfg config.Config) (api.ConductorClient, error) {
	timeout := time.Duration(cfg.ConnectTimeout) * time.Second
	/*
	* WithInsecure - No auth; We rely on comms <-> haproxy tunnel for mutual TLS auth.
	* WithBlock - Synchronous gRPC connection creation; so that we can fail faster
	* WithTimeout - We should not wait forever for blocking connection to be completed. Default is 20s.
	 */
	conn, err := grpc.Dial(cfg.TransportURL, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(timeout))
	if err != nil {
		return nil, err
	}
	return api.NewConductorClient(conn), nil
}
