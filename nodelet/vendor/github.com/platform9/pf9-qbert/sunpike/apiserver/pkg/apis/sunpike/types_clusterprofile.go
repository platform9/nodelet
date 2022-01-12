package sunpike

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ClusterProfilePhase string
type ClusterProfileBindingPhase string

const (
	ProfileDraft      ClusterProfilePhase        = "draft"
	ProfilePublished  ClusterProfilePhase        = "published"
	ProfileDeleted    ClusterProfilePhase        = "deleting"
	ProfileCreating   ClusterProfilePhase        = "creating"
	ProfileErrored    ClusterProfilePhase        = "errored"
	BindingErrored    ClusterProfileBindingPhase = "errored"
	BindingApply      ClusterProfileBindingPhase = "applying"
	BindingSuccessful ClusterProfileBindingPhase = "success"
	BindingDeleting   ClusterProfileBindingPhase = "deleting"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ClusterProfile struct {
	metav1.TypeMeta `json:",inline" protobuf:"bytes,1,opt,name=typeMeta"`
	// Standard object's metadata.
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,2,opt,name=metadata"`

	// Contents of the profile are stored separately but the location will be stored here
	Spec ClusterProfileSpec `json:"spec" protobuf:"bytes,3,name=spec"`

	// Current phase of the profile.
	// Valid values - draft, published, deleting, creating and errored
	// +optional
	Status ClusterProfileStatus `json:"status,omitempty" protobuf:"bytes,4,opt,name=status"`

	// User friendly description for the cluster profile
	// +optional
	Description string `json:"descrption,omitempty" protobuf:"bytes,5,opt,name=description"`
}

type ClusterProfileSpec struct {
	// NOTE: Either one of the following 2 fields, Location or ClonedFrom, must be specified
	// Cluster profile data i.e. the rules in the profile will be stored separately.
	// This field captures the same.
	// +optional
	Location string `json:"location,omitempty" protobuf:"bytes,1,opt,name=location"`

	// The identifier of the cluster or cluster profile from which this profile should be cloned.
	// Can either refer a cluster or a cluster profile
	// to refer to cluster profile the format must be "projectID (namespace)/clusterName"
	// Can be empty
	// +optional
	CloneFrom string `json:"cloneFrom,omitempty" protobuf:"bytes,2,opt,name=cloneFrom"`

	// List of namespace scoped resources to be included in the profile.
	// Takes effect only when cloning from an existing cluster.
	// Each field must follow the format of "NamespaceName/Resource type name/Resource Name" e.g.
	// "kube-system/roles/extension-apiserver-authentication-reader"
	// "kube-system/rolebindings/system::extension-apiserver-authentication-reader"
	// +optional
	NamespaceScopedResources []string `json:"namespaceScopedResources,omitempty" protobuf:"bytes,3,opt,name=namespaceScopedResources"`

	// List of cluster scoped resources to be included in the profile.
	// Takes effect only when cloning from an existing cluster.
	// Each field must follow the format of "Resource type name/Resource Name" e.g.
	// "clusterroles/view"
	// "clusterrolebindings/system:basic-user"
	// +optional
	ClusterScopedResources []string `json:"clusterScopedResources,omitempty" protobuf:"bytes,4,opt,name=clusterScopedResources"`

	// Time in minutes after which the profile object will be removed from apiserver post deletion
	// default is 10 min
	// +optional
	ReapInterval int32 `json:"reapInterval,omitempty" protobuf:"int32,5,opt,name=reapInterval"`
}

type ClusterProfileStatus struct {
	// Current phase of the profile.
	// Enum of - draft, published, deleted, create, error and uploading
	Phase ClusterProfilePhase `json:"phase" protobuf:"bytes,1,name=phase"`

	// Conditions defines current service state of the ClusterProfile.
	// +optional
	Conditions Conditions `json:"conditions,omitempty" protobuf:"bytes,2,opt,name=conditions"`

	// Message contains any additional information to augment the Phase field
	// It will also contain the reason for ClusterProfile in "errored" phase
	// +optional
	Message string `json:"message,omitempty" protobuf:"bytes,3,opt,name=message"`

	// RetryCount is used to keep track of how many times the controller loop has
	// run since ClusterProfile resource is created till corresponding
	// ClusterProfileDetail is created.
	RetryCount int32 `json:"retryCount,omitempty" protobuf:"int32,4,opt,name=retryCount"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ClusterProfileList struct {
	metav1.TypeMeta `json:",inline" protobuf:"bytes,1,opt,name=typeMeta"`
	// Standard object's metadata.
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,2,opt,name=metadata"`

	// Items contains the list of cluster profiles
	Items []ClusterProfile `json:"items" protobuf:"bytes,3,rep,name=items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ClusterProfileBinding struct {
	metav1.TypeMeta `json:",inline" protobuf:"bytes,1,opt,name=typeMeta"`
	// Standard object's metadata.
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,2,opt,name=metadata"`

	// Spec will contain the cluster and the profile from which the drift is to be evaluated
	Spec ClusterProfileBindingSpec `json:"spec" protobuf:"bytes,3,name=spec"`

	// Current status of the binding a profile to a cluster.
	// Read only.
	// +optional
	Status ClusterProfileBindingStatus `json:"status,omitempty" protobuf:"bytes,4,opt,name=status"`
}

type ClusterProfileBindingStatus struct {
	// Current phase of the binding.
	// Valid values are - error, applying and ok
	Phase ClusterProfileBindingPhase `json:"phase" protobuf:"bytes,1,name=phase"`

	// Message field will contain any additional data to augment the phase field.

	Message string `json:"message,omitempty" protobuf:"bytes,2,opt,name=message"`

	// RetryCount is used to keep track of how many times the controller loop has
	// run since ClusterProfile resource is created till corresponding
	// ClusterProfileDetail is created.
	RetryCount int32 `json:"retryCount,omitempty" protobuf:"int32,3,opt,name=retryCount"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ClusterProfileBindingList struct {
	metav1.TypeMeta `json:",inline" protobuf:"bytes,1,opt,name=typeMeta"`
	// Standard object's metadata.
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,2,opt,name=metadata"`

	// Items contains the list of cluster profile bindings
	Items []ClusterProfileBinding `json:"items" protobuf:"bytes,3,rep,name=items"`
}

type ClusterProfileBindingSpec struct {
	// ClusterRef contains the name of the cluster to which the clusterprofile should be applied
	// Points to cluster.metadata.name
	ClusterRef string `json:"clusterRef" protobuf:"bytes,1,opt,name=clusterRef"`

	// ClusterRef contains the namespaced name of the profile which should be applied
	// format: namespace (projectID)/clusterprofile.metadata.name
	ProfileRef string `json:"profileRef" protobuf:"bytes,2,opt,name=profileRef"`

	// Dry run should be set to true to analyse the current resources on the cluster against the profile
	// If DryRun is set to true, the clusterProfileBinding resource will be removed automatically after
	// 10 minutes after completion i.e. Status.Phase == (ok/error)
	// +optional
	DryRun bool `json:"dryRun,omitempty" protobuf:"bool,3,opt,name=dryRun"`

	// Time in minutes after which the profile object will be removed from apiserver post deletion
	// default is 10 min
	// +optional
	ReapInterval int32 `json:"reapInterval,omitempty" protobuf:"int32,4,opt,name=reapInterval"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ClusterProfileDetail struct {
	metav1.TypeMeta `json:",inline" protobuf:"bytes,1,opt,name=typeMeta"`
	// Standard object's metadata.
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,3,opt,name=metadata"`

	// Data contains the actual contents of the cluster profile.
	Data string `json:"data,omitempty" protobuf:"bytes,2,opt,name=data"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ClusterProfileBindingDetail struct {
	metav1.TypeMeta `json:",inline" protobuf:"bytes,1,opt,name=typeMeta"`
	// Standard object's metadata.
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,4,opt,name=metadata"`

	// Data contains the actual contents of the cluster profile.
	Data string `json:"data,omitempty" protobuf:"bytes,2,opt,name=data"`

	// analysis contains the drift analysis of the resources on cluster against the profile for which this detail is uploaded.
	// same field to be used for "dryrun" operation
	Analysis string `json:"analysis,omitempty" protobuf:"bytes,3,opt,name=analysis"`
}
