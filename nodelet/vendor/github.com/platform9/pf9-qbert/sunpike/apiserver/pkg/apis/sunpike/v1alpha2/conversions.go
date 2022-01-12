package v1alpha2

import (
	"k8s.io/apimachinery/pkg/conversion"

	"github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike"
)

func Convert_v1alpha2_HostSpec_To_sunpike_HostSpec(in *HostSpec, out *sunpike.HostSpec, s conversion.Scope) error {
	if err := autoConvert_v1alpha2_HostSpec_To_sunpike_HostSpec(in, out, s); err != nil {
		return err
	}

	out.PF9Cfg.ClusterID = in.ClusterID
	out.PF9Cfg.ClusterRole = in.ClusterRole
	out.PF9Cfg.KubeServiceState = in.KubeServiceState
	out.PF9Cfg.Debug = in.Debug
	return nil
}

func Convert_sunpike_HostSpec_To_v1alpha2_HostSpec(in *sunpike.HostSpec, out *HostSpec, s conversion.Scope) error {
	if err := autoConvert_sunpike_HostSpec_To_v1alpha2_HostSpec(in, out, s); err != nil {
		return err
	}

	out.ClusterID = in.PF9Cfg.ClusterID
	out.ClusterRole = in.PF9Cfg.ClusterRole
	out.KubeServiceState = in.PF9Cfg.KubeServiceState
	out.Debug = in.PF9Cfg.Debug
	return nil
}

func Convert_sunpike_HostStatus_To_v1alpha2_HostStatus(in *sunpike.HostStatus, out *HostStatus, s conversion.Scope) error {
	if err := autoConvert_sunpike_HostStatus_To_v1alpha2_HostStatus(in, out, s); err != nil {
		return err
	}

	// Not supported in v1alpha2: ClusterID
	return nil
}
