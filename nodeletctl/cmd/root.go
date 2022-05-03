package cmd

import (
	"fmt"
	"os"
	"path"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var cfgFile string
var logDir string
var debugToConsole bool

// hmm how can I avoid global variable
var JSONOuput bool

var (
	clusterBootstrapFile = "/opt/pf9/airctl/conf/nodeletCluster.yaml"
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:     "nodeletctl",
	Short:   "nodeletctl is a cluster manager for nodelets",
	Long:    `nodeletctl is a cluster manager to deploy and configure nodelets on remote machines`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {

	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/airctl-config.yaml)")
	RootCmd.PersistentFlags().BoolVar(&debugToConsole, "verbose", false, "print verbose logs to the console")
	RootCmd.PersistentFlags().BoolVar(&JSONOuput, "json", false, "json output for commands (configure-hosts only currently)")

}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	// Initializing zap log with console and file logging support
	if err := configureGlobalLog(debugToConsole, path.Join("/var/log/", "nodeletctl.log")); err != nil {
		fmt.Printf("log initialization failed: %s", err.Error())
		os.Exit(1)
	}
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".airctl" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName("airctl-config")
	}

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		zap.S().Info("Using config file:", viper.ConfigFileUsed())
	} else {
		zap.S().Errorf("Failed to read config file: %s", err)
	}
}

// ConfigureGlobalLog will log debug to console, else would put logs
// in the home directory
func configureGlobalLog(debugConsole bool, logFile string) error {

	// use lumberjack for log rotation
	f := zapcore.AddSync(&lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    100, // mb
		MaxBackups: 5,
		MaxAge:     30,
		Compress:   true,
	})
	// This seems pretty complicated, but all we are doing is making sure
	// error level is reported on console and console looks more 'production' like
	// whereas our log file is more like development log with stack traces etc.

	devEncoder := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
	prodEncoder := zapcore.NewConsoleEncoder(getProdEncoderConfig())

	// consoleLog will go to stderr
	consoleLogs := zapcore.Lock(os.Stderr)
	// all the logs to file
	fileLogs := zapcore.Lock(f)
	// by default on console we will only print panic error, unless debugConsole is specified
	consoleLvl := zap.PanicLevel
	if debugConsole {
		consoleLvl = zap.DebugLevel
	}

	core := zapcore.NewTee(
		zapcore.NewCore(prodEncoder, consoleLogs, consoleLvl),
		zapcore.NewCore(devEncoder, fileLogs, zap.DebugLevel),
	)

	logger := zap.New(core)
	defer logger.Sync()
	// use the logger we created globally
	zap.ReplaceGlobals(logger)
	// Now start the logging business
	zap.S().Debug("Logger started")
	return nil
}

func getProdEncoderConfig() zapcore.EncoderConfig {
	prodcfg := zap.NewProductionEncoderConfig()
	// by default production encoder has epoch time, using something more readable
	prodcfg.EncodeTime = zapcore.ISO8601TimeEncoder
	return prodcfg
}

func GetConfigFile() string {
	return viper.GetViper().ConfigFileUsed()
}

