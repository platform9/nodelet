package cmd

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/imdario/mergo"
	"github.com/olekukonko/tablewriter"
	"github.com/platform9/nodelet/nodelet/pkg/nodelet"
	"github.com/platform9/nodelet/nodelet/pkg/utils/command"
	"github.com/platform9/nodelet/nodelet/pkg/utils/config"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type phasesOpts struct {
	Phase         int
	Verbose       bool
	nodeletConfig *config.Config
	nodelet       *nodelet.Nodelet
	Single        bool
	Force         bool
	RegenCerts    bool
}

func newPhasesCommand() *cobra.Command {
	opts := &phasesOpts{}
	rootSvcCmd := &cobra.Command{
		Use:   "phases",
		Short: "Commands related to phases related to bring up of k8s stack",
	}

	startSvcCmd := &cobra.Command{
		Use:   "start",
		Short: "starts pf9 kube stack. Takes optional --from-phase param to allow starting from the specific phase",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := Context()
			defer cancel()

			err := opts.complete(ctx)
			if err != nil {
				return err
			}

			err = opts.validate(ctx, true)
			if err != nil {
				return err
			}

			err = opts.start(ctx)
			if err != nil {
				return nil
			}
			return nil
		},
	}

	stopSvcCmd := &cobra.Command{
		Use:   "stop",
		Short: "stops pf9 kube stack. Takes optional --till-phase param to allow stopping till the specific phase",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := Context()
			defer cancel()

			err := opts.complete(ctx)
			if err != nil {
				return err
			}

			err = opts.validate(ctx, true)
			if err != nil {
				return err
			}

			err = opts.stop(ctx, false)
			if err != nil {
				return nil
			}
			return nil
		},
	}

	restartSvcCmd := &cobra.Command{
		Use:   "restart",
		Short: "restarts pf9 kube stack. Takes optional --phase param to allow restarting from the specific phase",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := Context()
			defer cancel()

			err := opts.complete(ctx)
			if err != nil {
				return err
			}

			// If RegenCerts is false, skip running gen_certs phase in stop
			if !opts.RegenCerts {
				opts.nodelet.SkipGenCertsPhase()
			}

			err = opts.validate(ctx, true)
			if err != nil {
				return err
			}

			err = opts.restart(ctx)
			if err != nil {
				return err
			}
			return nil
		},
	}

	statusSvcCmd := &cobra.Command{
		Use:   "status",
		Short: "checks the status of Platform9 Kube on this host. Takes optional --phase param to check the status of a specific phase",
		// don't print usage on error
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := Context()
			defer cancel()

			err := opts.complete(ctx)
			if err != nil {
				return err
			}

			err = opts.validate(ctx, false)
			if err != nil {
				return err
			}

			err = opts.status(ctx)
			if err != nil {
				return err
			}
			return nil
		},
	}

	listPhasesCmd := &cobra.Command{
		Use:   "list",
		Short: "Lists the phases and their index numbers to use with rest of commands",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := Context()
			defer cancel()

			err := opts.complete(ctx)
			if err != nil {
				return err
			}

			opts.listPhases()
			return nil
		},
	}
	startSvcCmd.Flags().AddFlagSet(opts.startFlags())
	restartSvcCmd.Flags().AddFlagSet(opts.restartFlags())
	stopSvcCmd.Flags().AddFlagSet(opts.stopFlags())
	statusSvcCmd.Flags().AddFlagSet(opts.statusFlags())
	rootSvcCmd.AddCommand(startSvcCmd, stopSvcCmd, restartSvcCmd, statusSvcCmd, listPhasesCmd)
	return rootSvcCmd
}

// flags to be used in phases start command
func (o *phasesOpts) startFlags() *pflag.FlagSet {
	flagSet := pflag.NewFlagSet("", pflag.ContinueOnError)
	flagSet.BoolVarP(&o.Verbose, "verbose", "v", o.Verbose, "Prints more information")
	flagSet.IntVarP(&o.Phase, "from-phase", "p", o.Phase, "The number of the phase from which to start/restart. See 'nodeletd phases list' for the phases and their numbers.")
	// No shorthand notation for this flag to reduce accidental use.
	flagSet.BoolVarP(&o.Single, "single", "", o.Single, "Must be specified when trying to operate on a single phase (EXPERIMENTAL)")
	return flagSet
}

// flags to be used in phases restart command
func (o *phasesOpts) restartFlags() *pflag.FlagSet {
	flagSet := pflag.NewFlagSet("", pflag.ContinueOnError)
	flagSet.BoolVarP(&o.Verbose, "verbose", "v", o.Verbose, "Prints more information")
	flagSet.BoolVarP(&o.RegenCerts, "regen-certs", "", o.RegenCerts, "Option to regenerate certs by running gen_certs phase while stopping and starting")
	flagSet.IntVarP(&o.Phase, "from-phase", "p", o.Phase, "The number of the phase from which to restart. See 'nodeletd phases list' for the phases and their numbers.")
	// No shorthand notation for this flag to reduce accidental use.
	flagSet.BoolVarP(&o.Single, "single", "", o.Single, "Must be specified when trying to operate on a single phase (EXPERIMENTAL)")
	return flagSet
}

// flags to be used in phases stop command
func (o *phasesOpts) stopFlags() *pflag.FlagSet {
	flagSet := pflag.NewFlagSet("", pflag.ContinueOnError)
	flagSet.BoolVarP(&o.Verbose, "verbose", "v", o.Verbose, "Prints more information")
	flagSet.IntVarP(&o.Phase, "till-phase", "p", o.Phase, "The number of the phase till which to stop. See 'nodeletd phases list' for the phases and their numbers.")
	// No shorthand notation for flags below to reduce accidental use
	flagSet.BoolVarP(&o.Single, "single", "", o.Single, "Must be specified when trying to operate on a single phase (EXPERIMENTAL)")
	flagSet.BoolVarP(&o.Force, "force", "", o.Force, "Force causes the entire chain to be stopped irrespective of any failure during the stop chain")
	return flagSet
}

// flags to be used in phases status command
func (o *phasesOpts) statusFlags() *pflag.FlagSet {
	flagSet := pflag.NewFlagSet("", pflag.ContinueOnError)
	flagSet.BoolVarP(&o.Verbose, "verbose", "v", o.Verbose, "Prints more information")
	return flagSet
}

func (o *phasesOpts) complete(ctx context.Context) error {
	var err error
	o.nodeletConfig, err = config.GetConfigFromDir(constants.ConfigDir)
	o.nodeletConfig.Debug = strconv.FormatBool(o.Verbose)
	o.nodeletConfig.DisableConfigUpdate = true
	o.nodeletConfig.DisableExtFile = true
	o.nodeletConfig.DisableSunpike = true
	o.nodeletConfig.DisableLoop = true
	o.nodeletConfig.PF9StatusThresholdSeconds = 0
	o.nodeletConfig.PhaseRetry = 1
	err = mergo.Merge(o.nodeletConfig, &config.DefaultConfig)
	if err != nil {
		return fmt.Errorf("Error creating a default config: %+v", err)
	}
	o.nodelet, err = nodelet.CreateNodeletFromConfig(ctx, o.nodeletConfig)
	if err != nil {
		return fmt.Errorf("Error initializing nodelet instance: %+v", err)
	}
	setupLoggerForCommands()
	return nil
}

func (o *phasesOpts) start(ctx context.Context) error {
	var err error
	if o.Single {
		err = o.nodelet.StartSinglePhase(ctx, o.Phase)
	} else {
		_, err = o.nodelet.Start(ctx, o.Phase)
	}
	if err != nil {
		return fmt.Errorf("Error starting phase: %v", err)
	}
	o.displayPhaseStatus()
	return nil
}

func (o *phasesOpts) stop(ctx context.Context, isRestart bool) error {
	var err error
	if o.Single {
		err = o.nodelet.StopSinglePhase(ctx, o.Phase)
	} else {
		err = o.nodelet.Stop(ctx, o.Phase, o.Force)
	}
	if !isRestart {
		o.displayPhaseStatus()
	}
	return err
}

func (o *phasesOpts) restart(ctx context.Context) error {
	err := o.stop(ctx, true)
	if err != nil {
		fmt.Printf("Failed to cleanly stop pf9 kube. Attempting to start anyway.")
	}
	return o.start(ctx)
}

func (o *phasesOpts) status(ctx context.Context) error {
	o.nodelet.Status(ctx)
	o.displayPhaseStatus()
	if !o.nodelet.IsK8sRunning() {
		return fmt.Errorf("Platform9 Kubernetes stack is not running")
	}
	fmt.Printf("Platform9 Kubernetes stack is running\n")
	return nil
}

func (o *phasesOpts) validate(ctx context.Context, checkAgents bool) error {
	if checkAgents && arePF9AgentsRunning(ctx) {
		return fmt.Errorf("cannot run this command while hostagent and/or nodeletd is running. Stop pf9-hostagent and pf9-nodeletd before retrying")
	}

	// Phase number is invalid if it is higher than number of phases or less than 0.
	//Phase number == 0 is a valid case which is used to indicate a full stop or full restart.
	if o.Phase > o.nodelet.NumPhases() || o.Phase < 0 {
		return fmt.Errorf("Invalid phase index number: %d", o.Phase)
	}
	// Convert human readable number to zero-based array index
	if o.Phase > 0 {
		o.Phase = o.Phase - 1
	}
	return nil
}

func (o *phasesOpts) displayPhaseStatus() {
	headers := []string{"Index Number", "File", "Name", "Phase Status"}
	data := o.nodelet.PhasesStatus()
	displayTable(headers, data)
}

func (o *phasesOpts) listPhases() {
	headers := []string{"Index Number", "File", "Name", "Status Check"}
	data := o.nodelet.ListPhases()
	displayTable(headers, data)
}

func displayTable(headers []string, data [][]string) {
	table := tablewriter.NewWriter(os.Stdout)
	// Setup table options to make the table output look similar to kubectl output
	table.SetHeader(headers)
	table.SetAutoFormatHeaders(true)
	table.SetAutoWrapText(false)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetRowSeparator("")
	table.SetHeaderLine(false)
	table.SetBorder(false)
	table.SetTablePadding("\t")
	table.AppendBulk(data)

	// display the table
	table.Render()
}

func arePF9AgentsRunning(ctx context.Context) bool {
	localCmd := command.New()
	nodeletCheck, _ := localCmd.RunCommand(ctx, nil, -1, "", "systemctl", "status", "pf9-nodeletd")
	hostagentCheck, _ := localCmd.RunCommand(ctx, nil, -1, "", "systemctl", "status", "pf9-hostagent")

	return nodeletCheck == 0 || hostagentCheck == 0
}

func setupLoggerForCommands() {
	// Configures the logger to work similar to running /etc/init.d/pf9-kube commands
	// i.e. logging in console format without any field identifiers and printing just the message.
	logCfg := zap.Config{
		Level:             zap.NewAtomicLevelAt(zap.DebugLevel),
		Development:       false,
		DisableCaller:     false,
		DisableStacktrace: true,
		Encoding:          "console",
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        zapcore.OmitKey,
			LevelKey:       zapcore.OmitKey,
			NameKey:        zapcore.OmitKey,
			CallerKey:      zapcore.OmitKey,
			MessageKey:     "M",
			StacktraceKey:  zapcore.OmitKey,
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
