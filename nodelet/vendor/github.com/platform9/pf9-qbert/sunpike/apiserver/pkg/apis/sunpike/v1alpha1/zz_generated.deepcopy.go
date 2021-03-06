//go:build !ignore_autogenerated
// +build !ignore_autogenerated

/*
Copyright (C) 2015-2020 Platform9 Systems, Inc.
*/

// Code generated by deepcopy-gen. DO NOT EDIT.

package v1alpha1

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AddonsOpts) DeepCopyInto(out *AddonsOpts) {
	*out = *in
	out.AppCatalog = in.AppCatalog
	out.CAS = in.CAS
	out.Luigi = in.Luigi
	out.Kubevirt = in.Kubevirt
	out.CPUManager = in.CPUManager
	out.ProfileAgent = in.ProfileAgent
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AddonsOpts.
func (in *AddonsOpts) DeepCopy() *AddonsOpts {
	if in == nil {
		return nil
	}
	out := new(AddonsOpts)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AppCatalogOpts) DeepCopyInto(out *AppCatalogOpts) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AppCatalogOpts.
func (in *AppCatalogOpts) DeepCopy() *AppCatalogOpts {
	if in == nil {
		return nil
	}
	out := new(AppCatalogOpts)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CNIOpts) DeepCopyInto(out *CNIOpts) {
	*out = *in
	out.Calico = in.Calico
	out.Flannel = in.Flannel
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CNIOpts.
func (in *CNIOpts) DeepCopy() *CNIOpts {
	if in == nil {
		return nil
	}
	out := new(CNIOpts)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CPUManagerOpts) DeepCopyInto(out *CPUManagerOpts) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CPUManagerOpts.
func (in *CPUManagerOpts) DeepCopy() *CPUManagerOpts {
	if in == nil {
		return nil
	}
	out := new(CPUManagerOpts)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CalicoOpts) DeepCopyInto(out *CalicoOpts) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CalicoOpts.
func (in *CalicoOpts) DeepCopy() *CalicoOpts {
	if in == nil {
		return nil
	}
	out := new(CalicoOpts)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterAutoScalerOpts) DeepCopyInto(out *ClusterAutoScalerOpts) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterAutoScalerOpts.
func (in *ClusterAutoScalerOpts) DeepCopy() *ClusterAutoScalerOpts {
	if in == nil {
		return nil
	}
	out := new(ClusterAutoScalerOpts)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DockerOpts) DeepCopyInto(out *DockerOpts) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DockerOpts.
func (in *DockerOpts) DeepCopy() *DockerOpts {
	if in == nil {
		return nil
	}
	out := new(DockerOpts)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EtcdOpts) DeepCopyInto(out *EtcdOpts) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EtcdOpts.
func (in *EtcdOpts) DeepCopy() *EtcdOpts {
	if in == nil {
		return nil
	}
	out := new(EtcdOpts)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *FlannelOpts) DeepCopyInto(out *FlannelOpts) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new FlannelOpts.
func (in *FlannelOpts) DeepCopy() *FlannelOpts {
	if in == nil {
		return nil
	}
	out := new(FlannelOpts)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Host) DeepCopyInto(out *Host) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Host.
func (in *Host) DeepCopy() *Host {
	if in == nil {
		return nil
	}
	out := new(Host)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Host) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HostList) DeepCopyInto(out *HostList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Host, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HostList.
func (in *HostList) DeepCopy() *HostList {
	if in == nil {
		return nil
	}
	out := new(HostList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *HostList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HostPhase) DeepCopyInto(out *HostPhase) {
	*out = *in
	in.StartedAt.DeepCopyInto(&out.StartedAt)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HostPhase.
func (in *HostPhase) DeepCopy() *HostPhase {
	if in == nil {
		return nil
	}
	out := new(HostPhase)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HostSpec) DeepCopyInto(out *HostSpec) {
	*out = *in
	if in.ExtraCfg != nil {
		in, out := &in.ExtraCfg, &out.ExtraCfg
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	out.PF9Cfg = in.PF9Cfg
	out.ClusterCfg = in.ClusterCfg
	out.Etcd = in.Etcd
	out.Kubelet = in.Kubelet
	out.Docker = in.Docker
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HostSpec.
func (in *HostSpec) DeepCopy() *HostSpec {
	if in == nil {
		return nil
	}
	out := new(HostSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HostStatus) DeepCopyInto(out *HostStatus) {
	*out = *in
	out.Nodelet = in.Nodelet
	if in.Phases != nil {
		in, out := &in.Phases, &out.Phases
		*out = make([]HostPhase, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.AllStatusChecks != nil {
		in, out := &in.AllStatusChecks, &out.AllStatusChecks
		*out = make([]int32, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HostStatus.
func (in *HostStatus) DeepCopy() *HostStatus {
	if in == nil {
		return nil
	}
	out := new(HostStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KeepalivedOpts) DeepCopyInto(out *KeepalivedOpts) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KeepalivedOpts.
func (in *KeepalivedOpts) DeepCopy() *KeepalivedOpts {
	if in == nil {
		return nil
	}
	out := new(KeepalivedOpts)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KeystoneOpts) DeepCopyInto(out *KeystoneOpts) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KeystoneOpts.
func (in *KeystoneOpts) DeepCopy() *KeystoneOpts {
	if in == nil {
		return nil
	}
	out := new(KeystoneOpts)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubeApiserverOpts) DeepCopyInto(out *KubeApiserverOpts) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubeApiserverOpts.
func (in *KubeApiserverOpts) DeepCopy() *KubeApiserverOpts {
	if in == nil {
		return nil
	}
	out := new(KubeApiserverOpts)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubeClusterOpts) DeepCopyInto(out *KubeClusterOpts) {
	*out = *in
	out.Scheduler = in.Scheduler
	out.ControllerManager = in.ControllerManager
	out.Apiserver = in.Apiserver
	out.CNI = in.CNI
	out.Addons = in.Addons
	out.KubeProxy = in.KubeProxy
	out.MetalLB = in.MetalLB
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubeClusterOpts.
func (in *KubeClusterOpts) DeepCopy() *KubeClusterOpts {
	if in == nil {
		return nil
	}
	out := new(KubeClusterOpts)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubeControllerManagerOpts) DeepCopyInto(out *KubeControllerManagerOpts) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubeControllerManagerOpts.
func (in *KubeControllerManagerOpts) DeepCopy() *KubeControllerManagerOpts {
	if in == nil {
		return nil
	}
	out := new(KubeControllerManagerOpts)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubeProxyOpts) DeepCopyInto(out *KubeProxyOpts) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubeProxyOpts.
func (in *KubeProxyOpts) DeepCopy() *KubeProxyOpts {
	if in == nil {
		return nil
	}
	out := new(KubeProxyOpts)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubeSchedulerOpts) DeepCopyInto(out *KubeSchedulerOpts) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubeSchedulerOpts.
func (in *KubeSchedulerOpts) DeepCopy() *KubeSchedulerOpts {
	if in == nil {
		return nil
	}
	out := new(KubeSchedulerOpts)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubeVirtOpts) DeepCopyInto(out *KubeVirtOpts) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubeVirtOpts.
func (in *KubeVirtOpts) DeepCopy() *KubeVirtOpts {
	if in == nil {
		return nil
	}
	out := new(KubeVirtOpts)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubeletOpts) DeepCopyInto(out *KubeletOpts) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubeletOpts.
func (in *KubeletOpts) DeepCopy() *KubeletOpts {
	if in == nil {
		return nil
	}
	out := new(KubeletOpts)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LuigiOpts) DeepCopyInto(out *LuigiOpts) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LuigiOpts.
func (in *LuigiOpts) DeepCopy() *LuigiOpts {
	if in == nil {
		return nil
	}
	out := new(LuigiOpts)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MetalLBOpts) DeepCopyInto(out *MetalLBOpts) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MetalLBOpts.
func (in *MetalLBOpts) DeepCopy() *MetalLBOpts {
	if in == nil {
		return nil
	}
	out := new(MetalLBOpts)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NodeletStatus) DeepCopyInto(out *NodeletStatus) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NodeletStatus.
func (in *NodeletStatus) DeepCopy() *NodeletStatus {
	if in == nil {
		return nil
	}
	out := new(NodeletStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PF9Opts) DeepCopyInto(out *PF9Opts) {
	*out = *in
	out.Keepalived = in.Keepalived
	out.Keystone = in.Keystone
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PF9Opts.
func (in *PF9Opts) DeepCopy() *PF9Opts {
	if in == nil {
		return nil
	}
	out := new(PF9Opts)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ProfileAgentOpts) DeepCopyInto(out *ProfileAgentOpts) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ProfileAgentOpts.
func (in *ProfileAgentOpts) DeepCopy() *ProfileAgentOpts {
	if in == nil {
		return nil
	}
	out := new(ProfileAgentOpts)
	in.DeepCopyInto(out)
	return out
}
