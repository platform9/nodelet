package bashscriptbasedphases

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/platform9/nodelet/pkg/utils/command"
	"github.com/platform9/nodelet/pkg/utils/config"
	"github.com/platform9/nodelet/pkg/utils/constants"
	sunpikev1alpha1 "github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike/v1alpha1"
	"go.uber.org/zap"
)

// Phase is the type defining a phase which is run as part of bring up or tear down of k8s services.
// It extends the sunpike Phase with additional fields needed during runtime on the host
type Phase struct {
	*sunpikev1alpha1.HostPhase
	Filename string
	Retry    int
}

// GetLocalCmd makes it convenient to mock command.New in unit tests
var LocalCmd = command.New()

// GetHostPhase returns the embedded HostPhase struct
func (p *Phase) GetHostPhase() sunpikev1alpha1.HostPhase {
	return *p.HostPhase
}

// GetOrder returns the phase order from embedded HostPhase struct
func (p *Phase) GetOrder() int {
	return int(p.HostPhase.Order)
}

func (p *Phase) GetPhaseName() string {
	return p.Name
}

// RunCommand allows running different operations of a phase script. This function can be
// private however gomock library used in unit test does not work properly with unexported methods.
func (p *Phase) runCommand(ctx context.Context, op string, cfg config.Config) ([]string, int) {
	baseCmd := constants.BaseCommand
	if _, ok := constants.ValidCgroupOps[op]; ok && cfg.UseCgroups {
		baseCmd = constants.BaseCgroupCommand
	}
	command := append(baseCmd, p.Filename, op)
	if cfg.IsDebug() {
		command = append(command, "--debug")
	}
	exec := command[0]
	args := command[1:]
	exitCode, output, _ := LocalCmd.RunCommandWithStdOut(ctx, nil, -1, "", exec, args...)
	return output, exitCode
}

// Status runs the "status" operation of a particular phase. Return value of 0 indicates that phase is ok/running.
// Otherwise it returns the exit code of phase script.
func (p *Phase) Status(ctx context.Context, cfg config.Config) error {
	exitCode := 1
	var cmdOutput []string
	// avoid false positives with single failed status check
	statusFn := func() error {
		cmdOutput, exitCode = p.runCommand(ctx, "status", cfg)
		if exitCode != 0 {
			return fmt.Errorf("non-zero exit code running status check: %s", p.Filename)
		}
		return nil
	}
	statusBackoff := getBackOff(p.Retry - 1)
	backoff.Retry(statusFn, statusBackoff)
	if exitCode == 0 {
		p.setHostStatus(constants.RunningState, "")
		return nil
	}
	p.setHostStatus(constants.FailedState, pruneAndLogCmdOutput(cmdOutput, cfg.NumCmdOutputLinesToLog))
	return fmt.Errorf("status check failed for phase: %s with exit code: %d", p.Name, exitCode)
}

// Start runs the "start" operation of a particular phase. Return value of 0 indicates that phase was started properly.
// Otherwise it returns the exit code of phase script.
func (p *Phase) Start(ctx context.Context, cfg config.Config) error {
	exitCode := 1
	var cmdOutput []string
	cmdOutput, exitCode = p.runCommand(ctx, "start", cfg)
	if exitCode != 0 {
		zap.S().Errorf("Error running phase: %s", p.Filename)
		p.setHostStatus(constants.FailedState, pruneAndLogCmdOutput(cmdOutput, cfg.NumCmdOutputLinesToLog))
		return fmt.Errorf("failed to start phase: %s with exit code: %d", p.Name, exitCode)
	}
	p.setHostStatus(constants.RunningState, "")
	return nil
}

// Stop runs the "stop" operation of a particular phase. Return value of 0 indicates that phase was stopped properly.
// Otherwise it returns the exit code of phase script.
func (p *Phase) Stop(ctx context.Context, cfg config.Config) error {
	exitCode := 1
	var cmdOutput []string
	stopFn := func() error {
		cmdOutput, exitCode = p.runCommand(ctx, "stop", cfg)
		if exitCode != 0 {
			errMsg := fmt.Sprintf("Non-zero exit code stopping: %s", p.Filename)
			return errors.New(errMsg)
		}
		return nil
	}
	stopBackoff := getBackOff(p.Retry - 1)
	backoff.Retry(stopFn, stopBackoff)
	if exitCode != 0 {
		p.setHostStatus(constants.StoppedState, pruneAndLogCmdOutput(cmdOutput, cfg.NumCmdOutputLinesToLog))
		return fmt.Errorf("failed to stop phase: %s with exit code: %d", p.Name, exitCode)
	}
	p.setHostStatus(constants.StoppedState, "")
	return nil
}

func (p *Phase) setHostStatus(status, message string) {
	// TODO: avoid mutating phase.HostPhase here. Instead try to handle it in calling
	// function with this function providing the necessary information.
	p.HostPhase.Status = status
	p.HostPhase.Message = message
}

func getBackOff(retry int) backoff.BackOff {
	backof := backoff.NewExponentialBackOff()
	backof.InitialInterval = 1 * time.Second
	backof.Multiplier = 2
	if retry <= 0 {
		retry = 1
	}
	return backoff.WithMaxRetries(backoff.NewExponentialBackOff(), uint64(retry))
}

// Convenience function to retain the last few lines as specified by maxLines. This function then proceeds to log the pruned result and returns the string of the same.
func pruneAndLogCmdOutput(output []string, maxLines int) string {
	prunedOutput := output
	if len(output) > maxLines {
		prunedOutput = output[len(output)-maxLines:]
	}
	for _, line := range prunedOutput {
		zap.S().Infof("%s", line)
	}
	return fmt.Sprintf("%v", prunedOutput)
}
