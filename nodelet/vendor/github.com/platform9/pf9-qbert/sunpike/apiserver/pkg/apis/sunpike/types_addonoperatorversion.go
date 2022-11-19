package sunpike

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AddonOperatorVersion is a representation of a supported add on version for kube versions.
type AddonOperatorVersion struct {
	metav1.TypeMeta `json:",inline" protobuf:"bytes,1,opt,name=typeMeta"`

	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,2,opt,name=metadata"`

	// KubeVersion is the Kubernetes-portion of the Version  as x.y.z
	// Example: 1.21.3
	KubeVersion string `json:"kubeVersion,omitempty" protobuf:"bytes,3,opt,name=kubeVersion"`

	// Addons contains addon name and supported versions for given kube version
	Versions []string `json:"versions,omitempty" protobuf:"bytes,4,opt,name=versions"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type AddonOperatorVersionList struct {
	metav1.TypeMeta `json:",inline" protobuf:"bytes,1,opt,name=typeMeta"`

	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,2,opt,name=metadata"`

	Items []AddonOperatorVersion `json:"items" protobuf:"bytes,3,rep,name=items"`
}
