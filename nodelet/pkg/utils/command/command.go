package command

import (
	"bufio"
	"context"
	"fmt"

	"go.uber.org/zap"

	//"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

// CLI interface contains the ways in which one can trigger commands
// from command line
type CLI interface {
	RunCommand(context.Context, map[string]string, int, string, string, ...string) (int, error)
	RunCommandWithStdOut(context.Context, map[string]string, int, string, string, ...string) (int, []string, error)
	RunCommandWithStdErr(context.Context, map[string]string, int, string, string, ...string) (int, []string, error)
	RunCommandWithStdOutStdErr(context.Context, map[string]string, int, string, string, ...string) (int, []string, []string, error)
}

// Pf9Cmd represents an encapsulated command object with bells
// and whistles
type Pf9Cmd struct{}

// New returns an instance of Pf9Cmd
func New() CLI {
	return &Pf9Cmd{}
}

// RunCommand runs a command
func (c *Pf9Cmd) RunCommand(ctx context.Context, env map[string]string, timeout int, cwd, path string, args ...string) (int, error) {

	var cmd *exec.Cmd
	exitCode := -1

	if timeout > 0 {
		ctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
		defer cancel()

		cmd = exec.CommandContext(ctx, path, args...)
	} else {
		cmd = exec.Command(path, args...)
	}

	for k, v := range env {
		cmd.Env = append(os.Environ(), fmt.Sprintf("%s=%s", k, v))
	}

	if _, err := os.Stat(cwd); err == nil {
		cmd.Dir = cwd
	}

	zap.S().Infof("Running command '%s %s' from wd: '%s'", path, strings.Join(args, " "), cmd.Dir)

	if err := cmd.Run(); err != nil {
		zap.S().Errorf("Error: %s", err.Error())
		if exitError, ok := err.(*exec.ExitError); ok {
			if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
				zap.S().Errorf("Exit Status: %d", status.ExitStatus())
				return status.ExitStatus(), err
			}
		}
		return exitCode, err
	}
	return 0, nil
}

// RunCommandWithStdOut runs a command and prints all the contents
// from STDOUT
func (c *Pf9Cmd) RunCommandWithStdOut(ctx context.Context, env map[string]string, timeout int,
	cwd, path string, args ...string) (int, []string, error) {

	stdOut := []string{}
	var cmd *exec.Cmd
	exitCode := -1

	if timeout > 0 {
		ctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
		defer cancel()

		cmd = exec.CommandContext(ctx, path, args...)
	} else {
		cmd = exec.Command(path, args...)
	}

	for k, v := range env {
		cmd.Env = append(os.Environ(), fmt.Sprintf("%s=%s", k, v))
	}

	if _, err := os.Stat(cwd); err == nil && cwd != "" {
		cmd.Dir = cwd
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		zap.S().Errorf("Error: %s", err.Error())
		return exitCode, nil, err
	}

	zap.S().Infof("Running command '%s %s' from wd: '%s'", path, strings.Join(args, " "), cmd.Dir)

	if err := cmd.Start(); err != nil {
		zap.S().Errorf("Error: %s", err.Error())
		return exitCode, stdOut, err
	}

	scanner := bufio.NewScanner(stdoutPipe)
	// Can use bufio.ScanWords for fetching individual words
	// or bufio.ScanRunes for individual runes/characters
	scanner.Split(bufio.ScanLines)
	zap.S().Infof("STDOUT: ")
	for scanner.Scan() {
		m := scanner.Text()
		zap.S().Infof(m)
		stdOut = append(stdOut, m)
	}
	if err := cmd.Wait(); err != nil {
		zap.S().Errorf("Error: %s", err.Error())
		if exitError, ok := err.(*exec.ExitError); ok {
			if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
				zap.S().Errorf("Exit Status: %d", status.ExitStatus())
				return status.ExitStatus(), stdOut, err
			}
		}
		return exitCode, stdOut, err
	}
	return 0, stdOut, nil
}

// RunCommandWithStdErr runs a command and prints all the contents
// from STDERR
func (c *Pf9Cmd) RunCommandWithStdErr(ctx context.Context, env map[string]string, timeout int,
	cwd, path string, args ...string) (int, []string, error) {

	stdErr := []string{}
	var cmd *exec.Cmd
	exitCode := -1

	if timeout > 0 {
		ctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
		defer cancel()

		cmd = exec.CommandContext(ctx, path, args...)
	} else {
		cmd = exec.Command(path, args...)
	}

	for k, v := range env {
		cmd.Env = append(os.Environ(), fmt.Sprintf("%s=%s", k, v))
	}

	if _, err := os.Stat(cwd); err == nil {
		cmd.Dir = cwd
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		zap.S().Errorf("Error: %s", err.Error())
		return exitCode, nil, err
	}

	zap.S().Infof("Running command '%s %s' from wd: '%s'", path, strings.Join(args, " "), cmd.Dir)

	if err := cmd.Start(); err != nil {
		zap.S().Errorf("Error: %s", err.Error())
		return exitCode, stdErr, err
	}

	scanner := bufio.NewScanner(stderrPipe)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		m := scanner.Text()
		zap.S().Infof("STDERR: %s", m)
		stdErr = append(stdErr, m)
	}
	if err := cmd.Wait(); err != nil {
		zap.S().Errorf("Error: %s", err.Error())
		if exitError, ok := err.(*exec.ExitError); ok {
			if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
				zap.S().Errorf("Exit Status: %d", status.ExitStatus())
				return status.ExitStatus(), stdErr, err
			}
		}
		return exitCode, stdErr, err
	}
	return 0, stdErr, nil
}

// RunCommandWithStdOutStdErr runs a command and prints all the contents
// from STDOUT and STDERR together
func (c *Pf9Cmd) RunCommandWithStdOutStdErr(ctx context.Context, env map[string]string, timeout int,
	cwd, path string, args ...string) (int, []string, []string, error) {

	stdOut, stdErr := []string{}, []string{}
	var cmd *exec.Cmd
	exitCode := -1

	if timeout > 0 {
		ctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
		defer cancel()

		cmd = exec.CommandContext(ctx, path, args...)
	} else {
		cmd = exec.Command(path, args...)
	}

	for k, v := range env {
		cmd.Env = append(os.Environ(), fmt.Sprintf("%s=%s", k, v))
	}

	if _, err := os.Stat(cwd); err == nil {
		cmd.Dir = cwd
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		zap.S().Errorf("Error: %s", err.Error())
		return exitCode, stdOut, stdErr, err
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		zap.S().Errorf("Error: %s", err.Error())
		return exitCode, stdOut, stdErr, err
	}

	zap.S().Infof("Running command '%s %s' from wd: '%s'", path, strings.Join(args, " "), cmd.Dir)

	if err := cmd.Start(); err != nil {
		zap.S().Errorf("Error: %s", err.Error())
		return exitCode, stdOut, stdErr, err
	}

	stdoutScanner := bufio.NewScanner(stdoutPipe)
	stdoutScanner.Split(bufio.ScanLines)
	zap.S().Infof("STDOUT: ")
	for stdoutScanner.Scan() {
		m := stdoutScanner.Text()
		zap.S().Infof(m)
		stdOut = append(stdOut, m)
	}

	stderrScanner := bufio.NewScanner(stderrPipe)
	stderrScanner.Split(bufio.ScanLines)
	for stderrScanner.Scan() {
		m := stderrScanner.Text()
		zap.S().Infof("STDERR: %s", m)
		stdErr = append(stdErr, m)
	}
	if err := cmd.Wait(); err != nil {
		zap.S().Errorf("Error: %s", err.Error())
		if exitError, ok := err.(*exec.ExitError); ok {
			if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
				zap.S().Errorf("Exit Status: %d", status.ExitStatus())
				return status.ExitStatus(), stdOut, stdErr, err
			}
		}
		return exitCode, stdOut, stdErr, err
	}
	return 0, stdOut, stdErr, nil
}
