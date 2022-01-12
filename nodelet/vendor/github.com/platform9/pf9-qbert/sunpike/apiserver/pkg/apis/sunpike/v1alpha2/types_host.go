package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// HostState is the status of the Host.
type HostState string

const (
	HostStateUnknown    HostState = "Unknown"
	HostStateOk         HostState = "Ok"
	HostStateConverging HostState = "Converging"
	HostStateRetrying   HostState = "Retrying"
	HostStateFailed     HostState = "Failed"
)

var HostStateSet = map[HostState]struct{}{
	HostStateUnknown:    {},
	HostStateOk:         {},
	HostStateConverging: {},
	HostStateRetrying:   {},
	HostStateFailed:     {},
}

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Host is a representation of the configuration and status of a single machine.
type Host struct {
	metav1.TypeMeta `json:",inline" protobuf:"bytes,4,opt,name=typeMeta"`

	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Specification of the desired behavior of the Host.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status
	Spec HostSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`

	// Most recently observed status of the Host.
	// This data may not be up to date.
	// Populated by the system.
	// Read-only.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status
	Status HostStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// HostList is a list of Host objects.
type HostList struct {
	metav1.TypeMeta `json:",inline" protobuf:"bytes,3,opt,name=typeMeta"`

	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Items contains a list of Hosts.
	Items []Host `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// HostSpec contains the specification of the desired behavior of the Host.
//
// Dev note: This type MUST be kept in sync with server/sunpike.js in qbert.
// Make sure update both data structures when adding/modifying new fields.
type HostSpec struct {
	// ClusterID contains the name of the cluster to which this Host belongs.
	// The configuration of this cluster will be used to configure this Host.
	//
	// See the Cluster type for more info.
	ClusterID string `json:"clusterID" protobuf:"bytes,1,opt,name=clusterID"`

	// ClusterRole specifies the role that this Host should take within the
	// cluster. Options:
	//
	// - master:	turn the host into a Kubernetes master node.
	// - worker:	turn the host into a Kubernetes worker node.
	// - none:		do not turn the host into a Kubernetes node.
	ClusterRole string `json:"clusterRole,omitempty" protobuf:"bytes,2,opt,name=clusterRole" kube.env:"ROLE"`

	// KubeServiceState is the desired state of this Host in relation to
	// the target cluster. If set to "true" the Host should be added to the
	// cluster as a Node; if set to "false" the Host should be removed from the
	// cluster as a Node; and, if set to another value (commonly "" or "ignore")
	// the Host should simply be ignored and left in whatever state it is.
	KubeServiceState string `json:"kubeServiceState,omitempty" protobuf:"bytes,3,opt,name=kubeServiceState" kube.env:"KUBE_SERVICE_STATE"`

	// Debug will increase the verbosity of logging if set.
	Debug bool `json:"debug,omitempty" protobuf:"bool,4,opt,name=debug" kube.env:"DEBUG"`
}

// HostStatus represents information about the status of a Host. Status may
// trail the actual state of a system, especially if the Host is not able to
// contact Sunpike.
type HostStatus struct {
	//
	// State
	//

	// HostState constains the overall state of this Host.
	HostState HostState `json:"hostState,omitempty" protobuf:"bytes,1,rep,name=hostState"`

	// ClusterRole is the role that is currently observed to be installed on
	// the Host.
	ClusterRole string `json:"clusterRole,omitempty" protobuf:"bytes,2,opt,name=clusterRole"`

	// ServiceState is the current desired state of the Host. This should be
	// equivalent to the KubeServiceState in the HostSpec, but is also stored
	// in the HostStatus for backward-compatibility.
	//
	// If true the Host is being added to the
	// cluster as a Node; if set to false the Host is being removed from the
	// cluster as a Node; and, if set to another value (commonly "" or "ignore")
	// the Host is simply being ignored and left in whatever state it is.
	ServiceState bool `json:"serviceState,omitempty" protobuf:"bool,4,opt,name=serviceState"`

	// Hostname contains the hostname of the Host.
	Hostname string `json:"hostname,omitempty" protobuf:"bytes,5,opt,name=hostname"`

	// Nodelet contains information about the current state of the Nodelet
	// process on the Host.
	Nodelet NodeletStatus `json:"nodelet,omitempty" protobuf:"bytes,6,opt,name=nodelet"`

	//
	// Phases
	//

	// Phases provides details about the progress of the scripts (phases)
	// responsible for bringing the Host into or out of the Cluster. The phases
	// are run in the order they are shown here.
	Phases []HostPhase `json:"phases,omitempty" protobuf:"bytes,10,rep,name=phases"`

	// PhaseCompleted specifies the order of the phase until which the chain has
	// been completed successfully so far.
	PhaseCompleted int32 `json:"phaseCompleted,omitempty" protobuf:"varint,11,opt,name=phaseCompleted"`

	// LastFailedPhase will be the order of the phase which failed to start in
	// last attempt. If none have failed, this will be -1.
	LastFailedPhase int32 `json:"lastFailedPhase,omitempty" protobuf:"varint,12,opt,name=lastFailedPhase"`

	// StartAttempts describes the number of times Nodelet has attempted to run
	// the start scripts.
	StartAttempts int32 `json:"startAttempts,omitempty" protobuf:"varint,13,opt,name=startAttempts"`

	// CurrentPhase is the order of the phase which is currently running.
	CurrentPhase int32 `json:"currentPhase,omitempty" protobuf:"varint,14,opt,name=currentPhase"`

	//
	// StatusChecks
	//

	// AllStatusChecks will be the list of integer orders of the phases that
	// should be run as part of a status check.
	AllStatusChecks []int32 `json:"allStatusChecks,omitempty" protobuf:"bytes,20,rep,name=allStatusChecks"`

	// LastFailedCheck will be the order of the phase for which status check failed.
	LastFailedCheck int32 `json:"lastFailedCheck,omitempty" protobuf:"varint,21,opt,name=lastFailedCheck"`

	// LastFailedCheckTime contains the timestamp at which the last failed
	// status check was performed. The field is formatted as UNIX timestamp.
	LastFailedCheckTime int64 `json:"lastFailedCheckTime,omitempty" protobuf:"varint,22,opt,name=lastFailedCheckTime"`

	// Conditions defines current service state of the Cluster.
	// +optional
	Conditions Conditions `json:"conditions,omitempty" protobuf:"bytes,30,opt,name=conditions"`

	// ObservedGeneration is the latest generation observed by the controller.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty" protobuf:"varint,31,opt,name=observedGeneration"`

	// KubeVersion specifies the current version of Kubernetes running
	// on the corresponding Node. This is meant to be a means of bubbling
	// up status from the Node to the Machine.
	// It is entirely optional, but useful for end-user UX if it’s present.
	// +optional
	KubeVersion string `json:"kubeVersion,omitempty" protobuf:"bytes,32,opt,name=kubeVersion"`

	// PrimaryIP contains the primary IP of this Host.
	// +optional
	PrimaryIP string `json:"primaryIP,omitempty" protobuf:"bytes,33,opt,name=primaryIP"`
}

// NodeletStatus contains information about the Nodelet process
// itself (rather than about the Host).
type NodeletStatus struct {
	// Version is the version of Nodelet that is running on the Host.
	Version string `json:"version,omitempty" protobuf:"bytes,1,opt,name=version"`
}

// HostPhase contains the status of a single phase script.
type HostPhase struct {
	// Name provides a human-readable name for this phase. These names can
	// contain special characters and spaces.
	//
	// Example: "Configure and start auth web hook / pf9-bouncer"
	Name string `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`

	// Order is a unique number that is used to determine the order of the
	// phases. The phases are ordered ascendingly based on this field.
	Order int32 `json:"order,omitempty" protobuf:"varint,2,opt,name=order"`

	// StartedAt indicates when this phase started running.
	StartedAt metav1.Time `json:"startedAt,omitempty" protobuf:"bytes,3,opt,name=startedAt"`

	// Operation describes the type of operation that the phase is currently
	// performing. Options: start|stop|status.
	Operation string `json:"operation,omitempty" protobuf:"bytes,4,opt,name=operation"`

	// Status describes the current state of the phase.
	//
	// Options:
	// - not-started: state when no operation has been performed.
	// - running:     state when start operation was successful.
	// - stopped:     state when stop operation was successful.
	// - failed:      state when start operation has failed.
	// - executing:   state when an operation is being performed on a phase.
	Status string `json:"status,omitempty" protobuf:"bytes,5,opt,name=status"`

	// Message is a human readable description indicating details about why the
	// phase is in this status.
	//
	// When the status is an error, the Message contains the error message.
	Message string `json:"message,omitempty" protobuf:"bytes,6,opt,name=message"`
}
