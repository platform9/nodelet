package extensionfile

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/platform9/nodelet/pkg/utils/config"
	"github.com/platform9/nodelet/pkg/utils/constants"
	"github.com/platform9/nodelet/pkg/utils/fileio"
	"go.uber.org/zap"
)

// ExtensionFile interface contains the methods for reading and writing ExtensionData struct to extension data file.
type ExtensionFile interface {
	Write()
	Load() error
}

// ExtensionData : Struct for capturing data written to and read from extension data file.
type ExtensionData struct {
	// stick to _ notation as this data structure will be used by
	// hostagent extensions and all other extensions follow _ instead of camel casing.
	ServiceState           string   `json:"pf9_kube_service_state"`
	NodeState              string   `json:"pf9_kube_node_state"`
	StartAttempts          int      `json:"pf9_kube_start_attempt"`
	ClusterID              string   `json:"pf9_cluster_id"`
	ClusterRole            string   `json:"pf9_cluster_role"`
	AllStatusChecks        []string `json:"all_status_checks"`
	AllPhases              []string `json:"all_tasks"`
	CompletedPhases        []string `json:"completed_tasks"`
	CurrentStatusCheck     string   `json:"current_status_check"`
	CurrentPhase           string   `json:"current_task"`
	LastFailedCheck        string   `json:"last_failed_status_check"`
	LastFailedCheckTime    int64    `json:"last_failed_status_time"`
	LastFailedPhase        string   `json:"last_failed_task"`
	CurrentStatusCheckTime int64    `json:"status_check_timestamp"`
	file                   fileio.FileInterface
	path                   string
	log                    *zap.SugaredLogger
	Cfg                    *config.Config `json:"-"`

	// Operation is the current operation that nodelet is executing. Options: start|stop|status.
	Operation string `json:"-"`

	// KubeRunning indicates if the host is currently a node in the Kubernetes cluster.
	KubeRunning bool `json:"-"`

	// FailedStatusCheck contains the index of the first status check in the chain that failed.
	// If none failed, this should be -1.
	FailedStatusCheck int `json:"-"`

	// LastSuccessfulStatus contains the last the known time that the status check was successful.
	LastSuccessfulStatus time.Time `json:"-"`

	// StartFailStep contains the index of the start phase in the chain that failed.
	// If none failed, this should be -1.
	StartFailStep int `json:"-"`
}

// New returns new instance of ExtensionData.
func New(f fileio.FileInterface, path string, cfg *config.Config) ExtensionData {
	return ExtensionData{
		NodeState:     constants.ConvergingState,
		StartAttempts: 1,
		file:          f,
		path:          path,
		log:           zap.S(),
		Cfg:           cfg,
	}
}

func (data *ExtensionData) Write() {
	data.populateExtraFields()
	fileData, _ := json.MarshalIndent(*data, "", " ")
	err := data.file.WriteToFile(data.path, fileData, false)
	if err != nil {
		data.log.Errorf("Failed to write to %s. The error was %s", data.path, err)
	}
}

// convertExtnDataToJSON converts the old extension data file format to the new
// JSON format. This should be needed only once when upgrading to Platform9 release 4.3
// It is possible to use reflect library to auto-magically loop over json tags and
// also set values to fields in a struct object. Choosing to implement it directly
// to keep things simple.
func (data *ExtensionData) convertExtnDataToJSON() {
	oldData := data.getOldExtensionData()
	var err error
	for key, val := range oldData {
		value := val.(string)
		switch key {
		case "pf9_kube_service_state":
			data.ServiceState = value
		case "pf9_kube_node_state":
			data.NodeState = value
		case "pf9_kube_start_attempt":
			data.StartAttempts, err = strconv.Atoi(value)
			if err != nil {
				data.StartAttempts = 0
			}
		case "pf9_cluster_id":
			if value == "\"\"" {
				data.ClusterID = ""
			} else {
				data.ClusterID = value
			}
		case "pf9_cluster_role":
			data.ClusterRole = value
		case "all_status_checks":
			// Values change in phase 3 implementation.
			// So discard old data for all fields below this one.
			data.AllStatusChecks = []string{}
		case "all_tasks":
			data.AllPhases = []string{}
		case "completed_tasks":
			data.CompletedPhases = []string{}
		case "current_status_check":
			data.CurrentStatusCheck = ""
		case "current_task":
			data.CurrentPhase = ""
		case "last_failed_status_check":
			data.LastFailedCheck = ""
		case "last_failed_status_time":
			data.CurrentStatusCheckTime = 0
		case "last_failed_task":
			data.LastFailedPhase = ""
		case "status_check_timestamp":
			data.CurrentStatusCheckTime = 0
		}
	}
	data.log.Infof("Converted extension data to valid JSON data.")
	data.Write()
}

// Load : reads the extension data file and populates the structure
func (data *ExtensionData) Load() error {
	err := data.file.ReadJSONFile(data.path, &data)
	/*
		If err is nil then extension file is already JSON and no further processing is needed.
		if err is not nil, assumption is that extension file is older format and needs to be converted.
	*/
	if err != nil {
		data.convertExtnDataToJSON()
	}

	// Stopping pf9-kube stop saves the node state as `ok` by default.
	// This sets it to `converging` so that sunpike can record that the process
	// is starting up. A subsequent successful status check will set it back
	// to `ok`.
	data.convertNodeState()

	return nil
}

func (data *ExtensionData) convertNodeState() {
	if data.NodeState == constants.OkState {
		data.NodeState = constants.ConvergingState
		data.StartAttempts = 1
	}
}

func (data *ExtensionData) populateExtraFields() {
	if data.CurrentStatusCheckTime-data.LastFailedCheckTime >= constants.FailedStatusCheckReapInterval {
		data.LastFailedCheckTime = 0
		data.LastFailedCheck = ""
	}
	if data.ServiceState == constants.ServiceStateTrue {
		data.NodeState = constants.OkState
		return
	}
	// TODO(mithil) - this logic needs to be simplified
	switch {
	case data.Cfg.KubeServiceState == constants.ServiceStateTrue && !data.KubeRunning:
		fallthrough
	case data.Cfg.KubeServiceState == constants.ServiceStateFalse && data.KubeRunning:
		switch {
		case data.StartAttempts == 1:
			// Trying to start pf9-kube for first time
			data.NodeState = constants.ConvergingState
		case data.StartAttempts > 1 && data.StartAttempts <= constants.NumRetriesForErrorState:
			// Starting pf9-kube failed once and is being retried
			data.NodeState = constants.RetryingState
		case data.StartAttempts > constants.NumRetriesForErrorState:
			// Starting pf9-kube failed more than "numRetriesForErrorState" times and is being retried
			data.NodeState = constants.ErrorState
		}
	case data.Cfg.KubeServiceState == constants.ServiceStateTrue && data.KubeRunning:
		fallthrough
	case data.Cfg.KubeServiceState == constants.ServiceStateFalse && !data.KubeRunning:
		// pf9-kube is supposed to be stopped
		data.NodeState = constants.OkState
	}
}

func (data *ExtensionData) getOldExtensionData() map[string]interface{} {
	fileData := make(map[string]interface{})
	/*
		Read current extension file data and populate fileData.
	*/
	extnFileContents, err := data.file.ReadFileByLine(data.path)
	if err != nil {
		data.log.Warnf("Error reading from %s and will be overwritten. Error : %s", data.path, err)
		return fileData
	}
	for _, line := range extnFileContents {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.SplitN(string(line), " ", 2)
		if len(fields) != 2 {
			data.log.Debugf("failed processing %v tokens. Defaulting to empty string.", fields)
			fields = append(fields, "")
		}
		key := strings.TrimSpace(fields[0])
		value := strings.TrimSpace(fields[1])
		fileData[key] = value
	}
	return fileData
}
