package sunpike

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AddonPhase is a label for the status of a addon.
type AddonPhase string

// A toleration operator is the set of operators that can be used in a toleration.
type TolerationOperator string

const (
	TolerationOpExists TolerationOperator = "Exists"
	TolerationOpEqual  TolerationOperator = "Equal"
)

type TaintEffect string

// These are the valid statuses of Addons.
const (
	// AddonPhaseInstalling means the Addon has been picked up by the system,
	// for installation.
	AddonPhaseInstalling AddonPhase = "Installing"

	// AddonPhaseUnInstalling means the Addon has been picked up by the system,
	// for uninstallation.
	AddonPhaseUnInstalling AddonPhase = "Uninstalling"

	// AddonPhaseInstalled means the Addon has been successfully installed
	AddonPhaseInstalled AddonPhase = "Installed"

	// AddonPhaseUnInstalled means the Addon has been successfully uninstalled
	AddonPhaseUnInstalled AddonPhase = "Uninstalled"

	// AddonPhaseInstallError means that there was an error installing the Addon
	AddonPhaseInstallError AddonPhase = "InstallAddonError"

	// AddonPhaseUnInstallError means that there was an error uninstalling the Addon
	AddonPhaseUnInstallError AddonPhase = "UninstallAddonError"

	// AddonPhaseTerminating means that the addon has been scheduled for
	// deletion, but still has resources awaiting clean up.
	AddonPhaseTerminating AddonPhase = "Terminating"

	// AddonPhaseFailed means that the addon is in an error state and is
	// likely not operational. Manual intervention might be needed to remediate
	// the situation.
	AddonPhaseFailed AddonPhase = "Failed"

	// AddonPhaseUnknown means that for some reason the state of the addon
	// could not be determined.
	AddonPhaseUnknown AddonPhase = ""
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterAddon is a list of all addons to be installed on a cluster
type ClusterAddon struct {
	metav1.TypeMeta `json:",inline" protobuf:"bytes,1,opt,name=typeMeta"`

	// Standard object's metadata.
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,2,opt,name=metadata"`

	// Specification of the desired behavior of the Addon
	Spec ClusterAddonSpec `json:"spec,omitempty" protobuf:"bytes,3,opt,name=spec"`

	// Most recently observed status of the Addon
	Status ClusterAddonStatus `json:"status,omitempty" protobuf:"bytes,4,opt,name=status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterAddonList is a list of Addons objects.
type ClusterAddonList struct {
	metav1.TypeMeta `json:",inline" protobuf:"bytes,1,opt,name=typeMeta"`

	// Standard list metadata
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,2,opt,name=metadata"`

	// Items contains a list of Addons
	Items []ClusterAddon `json:"items" protobuf:"bytes,3,rep,name=items"`
}

// ClusterAddonSpec contains the specification of the desired behavior of the Addon.
type ClusterAddonSpec struct {
	// +optional
	// ClusterID
	ClusterID string `json:"clusterID" protobuf:"bytes,1,opt,name=clusterID"`
	// Version of the Addon
	Version string `json:"version" protobuf:"bytes,2,opt,name=version"`
	// Type of addon, should be one supported by the operator
	Type string `json:"type" protobuf:"bytes,3,opt,name=type"`
	// Override is optional override params for the addon
	// +optional
	Override Override `json:"override,omitempty" protobuf:"bytes,4,opt,name=override"`
	// Watch resources deployed by the Addon and not allow manual changes
	// +optional
	Watch bool `json:"watch" protobuf:"bool,5,opt,name=watch"`
	// If specified, the addon pod's tolerations.
	// +optional
	Tolerations []Toleration `json:"tolerations,omitemoty" protobuf:"bytes,6,rep,name=tolerations"`
}

// ClusterAddonStatus represents information about the status of a Addon. Status may
type ClusterAddonStatus struct {
	// Phase represents the current phase of addon.
	// E.g. Installing, Uninstalling, Installed Successfully etc.
	// +optional
	Phase AddonPhase `json:"phase,omitempty" protobuf:"bytes,1,opt,name=phase"`

	// Message is a human-readable string that summarizes why the Addon is in this phase.
	// +optional
	Message string `json:"message,omitempty" protobuf:"bytes,4,opt,name=message"`

	// Healthy is true if the Addon is installed and healthy
	// +optional
	Healthy bool `json:"healthy,omitempty" protobuf:"bool,2,rep,name=healthy"`
	// LastChecked specifies the last time that the Addon object on the Cluster was checked
	// +optional
	LastChecked metav1.Time `json:"lastChecked,omitempty" protobuf:"bytes,3,opt,name=lastChecked"`
}

// The pod this Toleration is attached to tolerates any taint that matches
// the triple <key,value,effect> using the matching operator <operator>.
type Toleration struct {
	// Key is the taint key that the toleration applies to. Empty means match all taint keys.
	// If the key is empty, operator must be Exists; this combination means to match all values and all keys.
	// +optional
	Key string `json:"key" protobuf:"bytes,1,opt,name=key"`
	// Operator represents a key's relationship to the value.
	// Valid operators are Exists and Equal. Defaults to Equal.
	// Exists is equivalent to wildcard for value, so that a pod can
	// tolerate all taints of a particular category.
	// +optional
	Operator TolerationOperator `json:"operator" protobuf:"bytes,2,opt,name=operator,casttype=TolerationOperator"`
	// Value is the taint value the toleration matches to.
	// If the operator is Exists, the value should be empty, otherwise just a regular string.
	// +optional
	Value string `json:"value" protobuf:"bytes,3,opt,name=value"`
	// Effect indicates the taint effect to match. Empty means match all taint effects.
	// When specified, allowed values are NoSchedule, PreferNoSchedule and NoExecute.
	// +optional
	Effect TaintEffect `json:"effect" protobuf:"bytes,4,opt,name=effect,casttype=TaintEffect"`
	// TolerationSeconds represents the period of time the toleration (which must be
	// of effect NoExecute, otherwise this field is ignored) tolerates the taint. By default,
	// it is not set, which means tolerate the taint forever (do not evict). Zero and
	// negative values will be treated as 0 (evict immediately) by the system.
	// +optional
	TolerationSeconds *int64 `json:"tolerationSeconds" protobuf:"varint,5,opt,name=tolerationSeconds"`
}

const (
	// Do not allow new pods to schedule onto the node unless they tolerate the taint,
	// but allow all pods submitted to Kubelet without going through the scheduler
	// to start, and allow all already-running pods to continue running.
	// Enforced by the scheduler.
	TaintEffectNoSchedule TaintEffect = "NoSchedule"
	// Like TaintEffectNoSchedule, but the scheduler tries not to schedule
	// new pods onto the node, rather than prohibiting new pods from scheduling
	// onto the node entirely. Enforced by the scheduler.
	TaintEffectPreferNoSchedule TaintEffect = "PreferNoSchedule"
	// NOT YET IMPLEMENTED. TODO: Uncomment field once it is implemented.
	// Like TaintEffectNoSchedule, but additionally do not allow pods submitted to
	// Kubelet without going through the scheduler to start.
	// Enforced by Kubelet and the scheduler.
	// TaintEffectNoScheduleNoAdmit TaintEffect = "NoScheduleNoAdmit"

	// Evict any already-running pods that do not tolerate the taint.
	// Currently enforced by NodeController.
	TaintEffectNoExecute TaintEffect = "NoExecute"
)

// Override defines params to override in the addon
type Override struct {
	// Params list of override params
	Params []Params `json:"params,omitempty" protobuf:"bytes,1,rep,name=params"`
}

// Params defines params to override in the addon
type Params struct {
	// Name of the parameter to override, should be present in the yaml
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`
	// Value of the overridden parameter
	Value string `json:"value" protobuf:"bytes,2,opt,name=value"`
}
