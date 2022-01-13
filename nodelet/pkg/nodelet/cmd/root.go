package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/erwinvaneyk/goversion"
	goversionext "github.com/erwinvaneyk/goversion/pkg/extensions"
	"github.com/ghodss/yaml"
	"github.com/imdario/mergo"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/platform9/nodelet/pkg/nodelet"
	"github.com/platform9/nodelet/pkg/utils/config"
	"github.com/platform9/nodelet/pkg/utils/constants"
)

type RootOptions struct {
	Debug               bool
	ConfigFileOrDirPath string
	NodeletConfig       config.Config
	LoopInterval        time.Duration
}

func NewCmdRoot() *cobra.Command {
	opts := &RootOptions{
		ConfigFileOrDirPath: constants.ConfigDir,
	}

	cmd := &cobra.Command{
		Use:   "nodeletd",
		Short: "Platform9 agent responsible for bootstrapping the local host into a Kubernetes node.",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := Context()
			defer cancel()

			err := opts.Complete(cmd, args)
			if err != nil {
				return err
			}

			err = opts.Validate()
			if err != nil {
				return err
			}

			return opts.Run(ctx)
		},
	}
	cmd.Flags().AddFlagSet(opts.Flags())
	cmd.AddCommand(newPhasesCommand())
	cmd.AddCommand(newAdvancedCommand())
	cmd.AddCommand(goversionext.NewCobraCmdWithDefaults())

	return cmd
}

func Execute() {
	if err := NewCmdRoot().Execute(); err != nil {
		zap.L().Fatal(err.Error())
	}
}

func (o *RootOptions) Flags() *pflag.FlagSet {
	flagSet := pflag.NewFlagSet("", pflag.ContinueOnError)
	flagSet.BoolVar(&o.Debug, "debug", o.Debug,
		"Run nodelet with more verbose logging enabled.")
	flagSet.StringVar(&o.ConfigFileOrDirPath, "config", o.ConfigFileOrDirPath,
		"Path to the config directory or file.")
	flagSet.StringVar(&o.NodeletConfig.PhaseScriptsDir, "scripts", o.NodeletConfig.PhaseScriptsDir,
		"Path to the base directory of the phase scripts to run.")
	flagSet.StringVar(&o.NodeletConfig.ExtensionOutputFile, "extfile", o.NodeletConfig.ExtensionOutputFile,
		"Path to the extension output file to persist the state to.")
	flagSet.StringVar(&o.NodeletConfig.KubeEnvPath, "kube.env", o.NodeletConfig.KubeEnvPath,
		"Path where the kube.env symlink should be created.")
	flagSet.DurationVar(&o.LoopInterval, "loop-interval", o.LoopInterval,
		"The duration between two successive status checks/script runs.")
	flagSet.StringVar(&o.NodeletConfig.HostID, "host", o.NodeletConfig.HostID,
		"ID of the host. Overwrites the value in the config file if set.")
	flagSet.StringVar(&o.NodeletConfig.ClusterID, "cluster", o.NodeletConfig.ClusterID,
		"ID of the cluster. Overwrites the value in the config file if set.")
	flagSet.StringVar(&o.NodeletConfig.ClusterRole, "role", o.NodeletConfig.ClusterRole,
		fmt.Sprintf("The role of the host. For example: '%s' or '%s'. Overwrites the value in the config file if set.", constants.RoleMaster, constants.RoleWorker))
	flagSet.StringVar(&o.NodeletConfig.KubeServiceState, "kube-service-state", o.NodeletConfig.KubeServiceState,
		fmt.Sprintf("The desired kubernetes state of this host (%s = add to the cluster, %s = remove from cluster, %s = do not do anything).", constants.ServiceStateTrue, constants.ServiceStateFalse, constants.ServiceStateIgnore))
	flagSet.BoolVar(&o.NodeletConfig.DisableLoop, "disable-loop", o.NodeletConfig.DisableLoop,
		"Run only one iteration of the reconciliation loop and exit.")
	flagSet.BoolVar(&o.NodeletConfig.DisableSunpike, "disable-sunpike", o.NodeletConfig.DisableSunpike,
		"Disable status reporting through sunpike.")
	flagSet.BoolVar(&o.NodeletConfig.DisableExtFile, "disable-extfile", o.NodeletConfig.DisableExtFile,
		"Disable persisting of state to the extension file.")
	flagSet.BoolVar(&o.NodeletConfig.DisableScripts, "disable-scripts", o.NodeletConfig.DisableScripts,
		"Disable loading and running of all (start, stop and status) scripts.")
	flagSet.BoolVar(&o.NodeletConfig.DisableConfigUpdate, "disable-configupdate", o.NodeletConfig.DisableConfigUpdate,
		"If set, do not update config files (kube.env and nodelet/config.yaml) with newer configuration from sunpike.")
	flagSet.StringVar(&o.NodeletConfig.SunpikeKubeEnvPath, "kube_sunpike.env", o.NodeletConfig.SunpikeKubeEnvPath,
		"Path to the file where the kube.env generated using the Sunpike config should be stored and retrieved from.")
	flagSet.StringVar(&o.NodeletConfig.SunpikeConfigPath, "config_sunpike", o.NodeletConfig.SunpikeConfigPath,
		"Path to the file where the config received from Sunpike should be stored and retrieved from.")
	return flagSet
}

func (o *RootOptions) Complete(cmd *cobra.Command, args []string) error {

	// Complete the flag-filled Config
	if o.Debug {
		o.NodeletConfig.Debug = "true"
	}
	if o.LoopInterval > 0 {
		o.NodeletConfig.LoopInterval = int(o.LoopInterval.Seconds())
	}

	// If a config path is set, load it and merge it into the config.
	if o.ConfigFileOrDirPath != "" {
		// Check whether the provided path is a directory or a file
		fd, err := os.Stat(o.ConfigFileOrDirPath)
		if err != nil {
			return fmt.Errorf("failed to find %s file or directory: %v", o.ConfigFileOrDirPath, err)
		}
		var cfgFromFS *config.Config
		if fd.IsDir() {
			cfgFromFS, err = config.GetConfigFromDir(o.ConfigFileOrDirPath)
		} else {
			cfgFromFS, err = config.GetConfigFromFile(o.ConfigFileOrDirPath)
		}
		if err != nil {
			return fmt.Errorf("failed to load config file(s) from '%s': %v", o.ConfigFileOrDirPath, err)
		}
		err = mergo.Merge(&o.NodeletConfig, cfgFromFS)
		if err != nil {
			return err
		}
	}
	// Finally, merge the defaults into the config.
	err := mergo.Merge(&o.NodeletConfig, &config.DefaultConfig)
	if err != nil {
		return err
	}

	SetupLogger(o.Debug)

	return nil
}

func (o *RootOptions) Validate() error {
	if o.NodeletConfig.LoopInterval < 30 {
		return errors.New("loop interval cannot be lower then 30 seconds")
	}

	return nil
}

func (o *RootOptions) Run(ctx context.Context) error {
	// Print version
	log.Infof("Nodelet version info:\n%s", goversion.Get().ToPrettyJSON())

	// Print the used config
	b, err := yaml.Marshal(o.NodeletConfig)
	if err != nil {
		return err
	}
	log.Infof("Using Nodelet config:\n%s", string(b))

	// Start Nodelet
	nodeletd, err := nodelet.CreateNodeletFromConfig(ctx, &o.NodeletConfig)
	if err != nil {
		// An error in the CreateNodeletFromConfig is unrecoverable
		zap.S().Fatal(err)
	}
	err = nodeletd.Run(ctx)
	if err != nil {
		zap.S().Fatal(err)
	}
	return nil
}

func Context() (ctx context.Context, cancel func()) {
	// trap Ctrl+C and call cancel on the context
	ctx, origCancel := context.WithCancel(context.Background())
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGQUIT)

	// Define cancel here to avoid a race condition.
	cancel = func() {
		func() {
			signal.Stop(signalChan)
			origCancel()
		}()
	}

	// Cancel the context on a received signal.
	go func() {
		select {
		case sig := <-signalChan:
			zap.L().Warn("Signal received: ", zap.String("", sig.String()))
			cancel()
		case <-ctx.Done():
		}
	}()

	return ctx, cancel
}

func SetupLogger(debug bool) {
	lvl := zap.InfoLevel
	if debug {
		lvl = zap.DebugLevel
	}
	// This log config is a combination of zap development and production loggers
	// Takes json encoding and short json keys from production config and
	// takes the ISO 8601 timestamp format from development log config.
	// ISO 8601 timestamp format is the human readable format e.g. 2020-10-20 00:00:00
	logCfg := zap.Config{
		Level:             zap.NewAtomicLevelAt(lvl),
		Development:       false,
		DisableCaller:     false,
		DisableStacktrace: true,
		Encoding:          "json",
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "T",
			LevelKey:       "L",
			NameKey:        "N",
			CallerKey:      "C",
			MessageKey:     "M",
			StacktraceKey:  "S",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.CapitalLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.StringDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stdout"},
	}
	logger, err := logCfg.Build()
	if err != nil {
		panic("Incorrect zap log config")
	}
	zap.ReplaceGlobals(logger)
}
