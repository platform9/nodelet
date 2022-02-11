package config

import (
	"errors"
	"fmt"
	"os"
	"path"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/imdario/mergo"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/platform9/nodelet/pkg/utils/constants"
	"github.com/platform9/nodelet/pkg/utils/fileio"
)

// DefaultConfig contains sane defaults for nodelet service
var DefaultConfig = Config{
	Debug:                   "false",
	ClusterRole:             constants.RoleNone,
	KubeServiceState:        constants.ServiceStateIgnore,
	TransportURL:            "localhost:8111",
	ConnectTimeout:          20,
	FullRetryCount:          10,
	UseCgroups:              true,
	PhaseRetry:              3,
	CPULimit:                40,                            // percentage
	LoopInterval:            constants.DefaultLoopInterval, // seconds
	PhaseScriptsDir:         constants.DefaultPhaseBaseDir,
	ExtensionOutputFile:     constants.ExtensionOutputFile,
	DisableSunpike:          false,
	DisableLoop:             false,
	DisableExtFile:          false,
	DisableScripts:          false,
	KubeEnvPath:             constants.KubeEnvPath,
	ResmgrKubeEnvPath:       constants.DefaultResmgrKubeEnvPath,
	SunpikeConfigPath:       constants.DefaultSunpikeConfigPath,
	SunpikeKubeEnvPath:      constants.DefaultSunpikeKubeEnvPath,
	GRPCRetryMax:            3,
	GRPCRetryTimeoutSeconds: 5,
	NumCmdOutputLinesToLog:  10, // 0 indicates no command lines to be logged
}

// Config a struct to load the values from viper for future use.
type Config struct {
	Debug                     string  `mapstructure:"DEBUG"`
	ClusterRole               string  `mapstructure:"ROLE"`
	ClusterID                 string  `mapstructure:"CLUSTER_ID"`
	HostID                    string  `mapstructure:"HOSTID"`
	TransportURL              string  `mapstructure:"TRANSPORT_URL"`
	ConnectTimeout            int     `mapstructure:"CONNECTION_TIMEOUT"`
	KubeServiceState          string  `mapstructure:"KUBE_SERVICE_STATE"`
	FullRetryCount            int     `mapstructure:"FULL_RETRY_COUNT"`
	UseCgroups                bool    `mapstructure:"USE_CGROUPS"`
	PhaseRetry                int     `mapstructure:"PHASE_RETRY"`
	CPULimit                  float64 `mapstructure:"CPU_LIMIT"`
	PF9StatusThresholdSeconds int     `mapstructure:"PF9_STATUS_THRESHOLD_SECONDS"`
	LoopInterval              int     `mapstructure:"LOOP_INTERVAL"`
	PhaseScriptsDir           string  `mapstructure:"PHASE_SCRIPTS_DIR"`
	ExtensionOutputFile       string  `mapstructure:"EXTENSION_OUTPUT_FILE"`
	KubeEnvPath               string  `mapstructure:"KUBE_ENV_PATH"`
	ResmgrKubeEnvPath         string  `mapstructure:"RESMGR_KUBE_ENV_PATH"`
	SunpikeConfigPath         string  `mapstructure:"SUNPIKE_CONFIG_PATH"`
	SunpikeKubeEnvPath        string  `mapstructure:"SUNPIKE_KUBE_ENV_PATH"`
	DisableSunpike            bool    `mapstructure:"DISABLE_SUNPIKE"`
	DisableLoop               bool    `mapstructure:"DISABLE_LOOP"`
	DisableExtFile            bool    `mapstructure:"DISABLE_EXTFILE"`
	DisableScripts            bool    `mapstructure:"DISABLE_SCRIPTS"`
	DisableConfigUpdate       bool    `mapstructure:"DISABLE_CONFIGUPDATE"`
	DisableExitOnUpdate       bool    `mapstructure:"DISABLE_EXITONUPDATE"`
	GRPCRetryMax              uint    `mapstructure:"GRPC_RETRY_MAX"`
	GRPCRetryTimeoutSeconds   int     `mapstructure:"GRPC_RETRY_TIMEOUT_SECONDS"`
	NumCmdOutputLinesToLog    int     `mapstructure:"NUM_CMD_OP_LINES_TO_LOG"`
	CloudProviderType         string  `mapstructure:"CLOUD_PROVIDER_TYPE"`
	UseHostname               string  `mapstructure:"USE_HOSTNAME"`
	MasterIp                  string  `mapstructure:"MASTER_IP"`
	K8sApiPort                int     `mapstructure:"K8S_API_PORT"`
}

// ToStringMap converts the Config struct to a map of strings
func (c Config) ToStringMap() map[string]string {
	result := map[string]string{}
	cfgVal := reflect.ValueOf(c)
	cfgType := reflect.ValueOf(c).Type()
	for i := 0; i < cfgVal.NumField(); i++ {
		key := cfgType.Field(i).Tag.Get("mapstructure")
		if key == "" {
			continue
		}
		result[key] = fmt.Sprintf("%v", cfgVal.Field(i).Interface())
	}
	return result
}

// IsDebug is a convenience function to check if the debug is enabled in config
func (c Config) IsDebug() bool {
	val, err := strconv.ParseBool(c.Debug)
	if err != nil {
		return false
	}
	return val
}

func setDefaults() {
	for key, value := range DefaultConfig.ToStringMap() {
		viper.SetDefault(key, value)
	}
}

func getDefaultConfig() *Config {
	cfg := &Config{}
	setDefaults()
	_ = viper.Unmarshal(cfg)
	return cfg
}

/*
GetConfigFromDir : Tries to load YAML config files from configDir i.e. /etc/pf9/nodelet directory.
			This function returns an error if the directory is inaccessible or if no config files could be loaded
*/
func GetConfigFromDir(configDir string) (*Config, error) {
	pf9File := fileio.New()
	filesLoaded := 0
	cfg := getDefaultConfig()
	files, err := pf9File.ListFiles(configDir)
	if err != nil {
		zap.S().Errorf("cannot read config files in directory: %s. Error was: %s", configDir, err.Error())
		return cfg, err
	}
	sort.Strings(files)
	viper.SetConfigType("yaml")
	for _, filename := range files {
		if !strings.HasSuffix(filename, ".yaml") && !strings.HasSuffix(filename, ".yml") {
			continue
		}

		_, err := GetConfigFromFile(path.Join(configDir, filename))
		if err != nil {
			return nil, err
		}
		filesLoaded++
	}
	if filesLoaded == 0 {
		return cfg, errors.New("no config files could be loaded")
	}

	// while transitioning to nodelet phase 3 keeping all config options
	// backward compatible hence not setting new config option for logging.
	if err := viper.BindEnv("DEBUG"); err != nil {
		// This should never occur, as BindEnv only fails when the provided key is empty.
		return nil, err
	}

	// Unmarshall the config into struct before returning
	err = viper.Unmarshal(cfg)
	if err != nil {
		return cfg, err
	}
	return cfg, nil
}

func GetConfigFromFile(configFile string) (*Config, error) {
	viper.SetConfigType("yaml")
	setDefaults()
	fileObj, err := os.Open(configFile)
	if err != nil {
		return nil, fmt.Errorf("could not open config file '%s': %v", configFile, err)
	}
	defer fileObj.Close()
	err = viper.MergeConfig(fileObj)
	if err != nil {
		return nil, err
	}

	cfg := &Config{}
	err = viper.Unmarshal(cfg)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

// GetDefaultConfig returns a copy of the default config.
func GetDefaultConfig() (*Config, error) {
	cfg := &Config{}
	err := mergo.Merge(cfg, DefaultConfig)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}
