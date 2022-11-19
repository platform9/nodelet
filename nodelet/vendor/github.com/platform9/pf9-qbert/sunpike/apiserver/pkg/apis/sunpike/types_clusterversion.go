package sunpike

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ClusterVersionPhase is a label for the condition of a ClusterVersion at the
// current time.
type ClusterVersionPhase string

// These are the valid statuses of ClusterVersions.
const (
	// ClusterVersionPhaseActive is a latest version which is available for
	// new cluster deployments and upgrades. Supported by pf9. Equivalent to
	// the “supported“ state in qbert.
	ClusterVersionPhaseActive ClusterVersionPhase = "Active"

	// ClusterVersionPhaseSupported is not recommended for new cluster
	// deployments or as an upgrade target. Supported by pf9. Equivalent to
	// the “unsupported“ state in qbert.
	ClusterVersionPhaseSupported ClusterVersionPhase = "Supported"

	// ClusterVersionPhaseDeprecated is not recommended for new cluster
	// deployments or as an upgrade target. Warning that officially support by
	// pf9 is ending, recommending to upgrade clusters. Equivalent to the
	// “deprecated“ state in qbert.
	ClusterVersionPhaseDeprecated ClusterVersionPhase = "Deprecated"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterVersion is a representation of a supported version for Clusters.
//
// In qbert, this was known as a 'supported_version'.
type ClusterVersion struct {
	metav1.TypeMeta `json:",inline" protobuf:"bytes,1,opt,name=typeMeta"`

	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,2,opt,name=metadata"`

	// Version is the full format of the ClusterVersion. Generally, this should
	// be the same as the ClusterVersion name. However, with the restricted
	// format of the name field, they can differ. In case there is a difference
	// this field should be viewed as authoritative.
	// Example: 1.2.3-pmk.1801
	Version string `json:"version,omitempty" protobuf:"bytes,3,opt,name=version"`

	// KubeVersion is the Kubernetes-portion of the Version.
	// Example: 1.2.3
	KubeVersion string `json:"kubeVersion,omitempty" protobuf:"bytes,4,opt,name=kubeVersion"`

	// PMKVersion contains the PMK-portion of the Version.
	// Example: 1801
	PMKVersion string `json:"pmkVersion,omitempty" protobuf:"bytes,5,opt,name=pmkVersion"`

	// Addons contains addon-specific information about the versioning.
	// +optional
	Addons []ClusterVersionAddon `json:"addons,omitempty" protobuf:"bytes,6,opt,name=addons"`

	// Changelog contains a URL to the location of the changelog for this ClusterVersion.
	// Example: http://example.com/changelog/1.2.3-pmk.1801
	// +optional
	Changelog string `json:"changelog,omitempty" protobuf:"bytes,7,opt,name=changelog"`

	// ReleasedAt specifies the time at which this version was released.
	// +optional
	ReleasedAt metav1.Time `json:"releasedAt,omitempty" protobuf:"bytes,8,opt,name=releasedAt"`

	// Phase indicates the current state of this ClusterVersion.
	// Example: Active
	Phase ClusterVersionPhase `json:"phase,omitempty" protobuf:"bytes,9,opt,name=phase"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterVersionList is a list of ClusterVersion objects.
type ClusterVersionList struct {
	metav1.TypeMeta `json:",inline" protobuf:"bytes,3,opt,name=typeMeta"`

	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Items contains a list of Providers.
	Items []ClusterVersion `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// ClusterVersionAddon contains addon-specific information about the versioning.
type ClusterVersionAddon struct {
	// Name is the name of the Addon.
	Name string `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`

	// Version contains the version of the Addon.
	Version string `json:"version,omitempty" protobuf:"bytes,2,opt,name=version"`
}
