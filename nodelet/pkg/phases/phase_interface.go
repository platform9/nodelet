package phases

import (
	"context"

	"github.com/platform9/nodelet/nodelet/pkg/utils/config"
	sunpikev1alpha1 "github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike/v1alpha1"
)

// PhaseInterface is an interface to interact with the phases
type PhaseInterface interface {
	GetHostPhase() sunpikev1alpha1.HostPhase
	Status(context.Context, config.Config) error
	Start(context.Context, config.Config) error
	Stop(context.Context, config.Config) error
	GetPhaseName() string
	GetOrder() int
}
