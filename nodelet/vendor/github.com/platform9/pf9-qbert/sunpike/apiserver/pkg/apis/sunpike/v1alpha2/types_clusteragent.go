package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ClusterAgentPhase string

const (
	ClusterAgentPhaseConnected ClusterAgentPhase = "connected"
	ClusterAgentPhaseErrored   ClusterAgentPhase = "errored"
	ClusterAgentPhaseOffline   ClusterAgentPhase = "offline"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ClusterAgent struct {
	metav1.TypeMeta `json:",inline" protobuf:"bytes,1,opt,name=typeMeta"`
	// Standard object's metadata.
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,2,opt,name=metadata"`

	Spec ClusterAgentSpec `json:"spec" protobuf:"bytes,3,name=spec"`

	// Current phase of the agent.
	// Valid values - connected, offline, errored
	// +optional
	Status ClusterAgentStatus `json:"status,omitempty" protobuf:"bytes,4,opt,name=status"`
}

type ClusterAgentSpec struct {

	// ClusterName represents the identifier of the cluster for which
	// this ClusterAgent is intended.
	ClusterName string `json:"clusterName" protobuf:"bytes,1,name=clusterName"`
}

type ClusterAgentStatus struct {
	// Current phase of the cluster agent.
	// Enum of - connected, offline, errored
	Phase ClusterAgentPhase `json:"phase" protobuf:"bytes,1,name=phase"`

	// Message contains any additional information to augment the Agent field
	// It will also contain the reason for ClusterAgent in "errored" phase
	// +optional
	Message string `json:"message,omitempty" protobuf:"bytes,2,opt,name=message"`

	// LastHeartBeat contains information about the timestamp at which the last
	// heartbeat from the ClusterAgent was recorded. It's in UTC.
	LastHeartBeat metav1.Time `json:"lastHeartBeat" protobuf:"bytes,3,name=lastHeartBeat"`

	// Version determines the reported version of the ClusterAgent.
	Version string `json:"version" protobuf:"bytes,4,name=version"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ClusterAgentList struct {
	metav1.TypeMeta `json:",inline" protobuf:"bytes,1,opt,name=typeMeta"`
	// Standard object's metadata.
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,2,opt,name=metadata"`

	// Items contains the list of cluster profiles
	Items []ClusterAgent `json:"items" protobuf:"bytes,3,rep,name=items"`
}
