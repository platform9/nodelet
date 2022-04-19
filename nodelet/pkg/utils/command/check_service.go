package command

import (
	"context"

	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type ServiceUtil interface {
	RunAction(ctx context.Context, action string) ([]string, error)
}
type ServiceUtility struct {
	ServiceName string
	log         *zap.SugaredLogger
	cmd         CLI
}

func NewServiceUtil(name string) ServiceUtil {
	return &ServiceUtility{
		ServiceName: name,
		log:         zap.S(),
		cmd:         New(),
	}
}

func (su *ServiceUtility) RunAction(ctx context.Context, action string) ([]string, error) {

	su.log.Infof("Running %s %s", su.ServiceName, action)
	exitCode, output, err := su.cmd.RunCommandWithStdOut(ctx, nil, 0, "", "/bin/sudo", "/usr/bin/systemctl", action, su.ServiceName)
	if exitCode != 0 || err != nil {
		return nil, errors.Wrapf(err, "could not %s %s. exitcode:%v", action, su.ServiceName, exitCode)
	}
	su.log.Infof("%s %s succeeded", su.ServiceName, action)
	return output, nil
}
