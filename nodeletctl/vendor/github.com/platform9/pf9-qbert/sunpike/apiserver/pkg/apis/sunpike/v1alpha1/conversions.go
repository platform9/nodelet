package v1alpha1

import (
	"k8s.io/apimachinery/pkg/conversion"

	"github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike"
)

func Convert_v1alpha1_HostStatus_To_sunpike_HostStatus(in *HostStatus, out *sunpike.HostStatus, s conversion.Scope) error {
	if err := autoConvert_v1alpha1_HostStatus_To_sunpike_HostStatus(in, out, s); err != nil {
		return err
	}

	// Ignored in internal type: CurrentStatusCheck, CurrentStatusCheckTime
	return nil
}

func Convert_sunpike_HostStatus_To_v1alpha1_HostStatus(in *sunpike.HostStatus, out *HostStatus, s conversion.Scope) error {
	if err := autoConvert_sunpike_HostStatus_To_v1alpha1_HostStatus(in, out, s); err != nil {
		return err
	}

	// Not supported in v1alpha1: Conditions, ObservedGeneration, KubeVersion, PrimaryIP
	return nil
}
