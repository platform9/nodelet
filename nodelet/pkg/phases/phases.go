package phases

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/platform9/nodelet/pkg/utils/command"
	"github.com/platform9/nodelet/pkg/utils/config"
	"github.com/platform9/nodelet/pkg/utils/constants"

	"go.uber.org/zap"
)

// GetLocalCmd makes it convenient to mock command.New in unit tests
var GetLocalCmd = command.New

// InitAndLoadRolePhases initializes and then returns an map of order -> phase
// when successful otherwise returns an error
func InitAndLoadRolePhases(ctx context.Context, cfg config.Config) ([]PhaseInterface, error) {
	var err error
	if cfg.UseCgroups {
		if cfg.DisableScripts {
			zap.S().Warnf("Running scripts is disabled; not running cgroup setup.")
		} else {
			err = setupCgroup(ctx, cfg)
			if err != nil {
				zap.S().Warnf("Disabling use of cgroups as there was an error during setup. Err: %s", err.Error())
				cfg.UseCgroups = false
			}
		}
	}
	var phaseList []PhaseInterface
	switch cfg.ClusterRole {
	case constants.RoleNone:
		phaseList, err = GetNoRolePhases()
	case constants.RoleWorker:
		phaseList, err = GetWorkerPhases()
	case constants.RoleMaster:
		phaseList, err = GetMasterPhases()
	}

	if err != nil {
		zap.S().Errorf("error loading phases: %w", err)
		return []PhaseInterface{}, err
	}

	return phaseList, nil
}

func setupCgroup(ctx context.Context, cfg config.Config) error {
	localCmd := GetLocalCmd()
	commands := [][]string{}
	// CPU limit percentage
	cpuQuotaPtc := cfg.CPULimit
	if cpuQuotaPtc <= 0 || cpuQuotaPtc > 100 {
		zap.S().Warnf("Incorrect value set of CPU_LIMIT option: %f", cpuQuotaPtc)
		return errors.New("invalid value for CPU_LIMIT")
	}
	// Convert CPU limit percentage to time slice in microseconds
	// Refer last example here - https://www.kernel.org/doc/Documentation/scheduler/sched-bwc.txt
	// We are trying to calculate a quota limit considering the period of 1s i.e. 1000000us
	cpuQuota := cpuQuotaPtc / 100 * float64((1 * time.Second).Microseconds())
	cpuQuotaCmd := append(constants.CgroupQuotaCmd, fmt.Sprintf(constants.CgroupQuotaParam, cpuQuota), constants.CgroupName)
	commands = append(commands, constants.CgroupCreateCmd)
	commands = append(commands, constants.CgroupPeriodCmd)
	commands = append(commands, cpuQuotaCmd)
	for _, command := range commands {
		exec := command[0]
		args := command[1:]
		_, err := localCmd.RunCommand(ctx, nil, -1, "", exec, args...)
		if err != nil {
			zap.S().Warnf("Error running command: %v", command)
			return err
		}
	}
	return nil
}
