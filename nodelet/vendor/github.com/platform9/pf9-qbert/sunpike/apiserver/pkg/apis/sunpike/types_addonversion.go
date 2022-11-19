package sunpike

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AddonVersion is a representation of a supported add on version for kube versions.
type AddonVersion struct {
	metav1.TypeMeta `json:",inline" protobuf:"bytes,1,opt,name=typeMeta"`

	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,2,opt,name=metadata"`

	// KubeVersion is the Kubernetes-portion of the Version.
	// Example: 1.18, 1.19, 1.20, 1.21
	KubeVersion string `json:"kubeVersion,omitempty" protobuf:"bytes,3,opt,name=kubeVersion"`

	// Addons contains addon name and supported versions for given kube version
	Addons []AddonTypeVersion `json:"addons,omitempty" protobuf:"bytes,4,opt,name=addons"`
}

// AddonVersions contains supported addon versions
type AddonTypeVersion struct {
	// Name is the name of the Addon.
	Name string `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`

	// Versions contains the supported version list of the Addon.
	Versions []string `json:"versions,omitempty" protobuf:"bytes,2,opt,name=versions"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type AddonVersionList struct {
	metav1.TypeMeta `json:",inline" protobuf:"bytes,1,opt,name=typeMeta"`

	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,2,opt,name=metadata"`

	Items []AddonVersion `json:"items" protobuf:"bytes,3,rep,name=items"`
}
