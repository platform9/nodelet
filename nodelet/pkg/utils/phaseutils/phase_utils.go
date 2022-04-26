package phaseutils

import (
	sunpikev1alpha1 "github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike/v1alpha1"
)

func SetHostStatus(hostPhase *sunpikev1alpha1.HostPhase, status string, message string) {
	//TODO: Retry
	hostPhase.Status = status
	hostPhase.Message = message
}
