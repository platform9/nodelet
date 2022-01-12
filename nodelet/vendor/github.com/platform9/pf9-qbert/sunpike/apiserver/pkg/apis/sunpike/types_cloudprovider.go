package sunpike

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CloudProviderPhase is the status of a CloudProvider.
type CloudProviderPhase string

const (
	// CloudProviderStateUnknown is the state when the object has not been
	// evaluated yet. Default state.
	CloudProviderStateUnknown CloudProviderPhase = ""

	// CloudProviderPhaseConfiguring can occur when the controller is
	// busy validating/initializing/testing the provider.
	CloudProviderPhaseConfiguring CloudProviderPhase = "configuring"

	// CloudProviderPhaseUnavailable can occur when the cloud provider is
	// temporary unavailable or has been disabled.
	CloudProviderPhaseUnavailable CloudProviderPhase = "unavailable"

	// CloudProviderPhaseErrored will occur when, for example, the
	// CloudProvider has been configured incorrectly or the credentials
	// have been revoked.
	CloudProviderPhaseErrored CloudProviderPhase = "errored"

	// CloudProviderPhaseAvailable indicates that the ClusterProvider is
	// configured and ready to be used to create Hosts and Clusters.
	CloudProviderPhaseAvailable CloudProviderPhase = "available"
)

// CloudProviderType indicates the name of a supported CloudProvider
type CloudProviderType string

const (
	CloudProviderTypeUnknown CloudProviderType = "" // Default
	CloudProviderTypeAWS     CloudProviderType = "aws"
	CloudProviderTypeAzure   CloudProviderType = "azure"
	CloudProviderTypeLocal   CloudProviderType = "local"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CloudProvider is a representation of an infrastructure provider, which is used
// to initialize Hosts and/or Clusters.
type CloudProvider struct {
	metav1.TypeMeta `json:",inline" protobuf:"bytes,4,opt,name=typeMeta"`

	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Specification of the desired behavior of the CloudProvider.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status
	Spec CloudProviderSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`

	// Most recently observed status of the CloudProvider.
	// This data may not be up to date.
	// Populated by the system.
	// Read-only.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status
	Status CloudProviderStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CloudProviderList is a list of CloudProvider objects.
type CloudProviderList struct {
	metav1.TypeMeta `json:",inline" protobuf:"bytes,3,opt,name=typeMeta"`

	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Items contains a list of Providers.
	Items []CloudProvider `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// CloudProviderSpec contains the specification of the desired configuration of the CloudProvider.
type CloudProviderSpec struct {
	// AWS contains all AWS-specific configuration for a CloudProvider.
	//
	// Any cluster-specific configuration goes into its type.
	// Only one of these should be non-nil.
	// +optional
	AWS *AWSCloudProviderSpec `json:"aws,omitempty" protobuf:"bytes,2,opt,name=aws"`

	// Azure contains all Azure-specific configuration for a CloudProvider.
	//
	// Any cluster-specific configuration goes into its type.
	// Only one of these should be non-nil.
	// +optional
	Azure *AzureCloudProviderSpec `json:"azure,omitempty" protobuf:"bytes,3,opt,name=azure"`

	// Local contains all configuration specific to a on-premise CloudProvider.
	//
	// Any cluster-specific configuration goes into its type.
	// Only one of these should be non-nil.
	// +optional
	Local *LocalCloudProviderSpec `json:"local,omitempty" protobuf:"bytes,4,opt,name=local"`
}

// CloudProviderStatus represents information about the status of a CloudProvider.
type CloudProviderStatus struct {
	// Phase describes the current phase of the CloudProvider.
	Phase CloudProviderPhase `json:"phase,omitempty" protobuf:"bytes,1,opt,name=phase"`

	// Type describes the controller-observed type of the CloudProvider. This is
	// based on what subfields are set in the CloudProviderSpec.
	// +optional
	Type CloudProviderType `json:"type,omitempty" protobuf:"bytes,2,opt,name=type"`

	// Conditions defines current service state of the CloudProvider.
	// +optional
	Conditions Conditions `json:"conditions,omitempty" protobuf:"bytes,3,opt,name=conditions"`

	// ObservedGeneration is the latest generation observed by the controller.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty" protobuf:"varint,4,opt,name=observedGeneration"`

	// Regions contains discovered and available regions in the cloudprovider.
	// +optional
	Regions []Region `json:"regions,omitempty" protobuf:"bytes,5,opt,name=regions"`

	// LastChecked specifies the last time that the accessibility to AWS was checked.
	// +optional
	LastChecked metav1.Time `json:"lastChecked,omitempty" protobuf:"bytes,6,opt,name=lastChecked"`
}

// Region resembles a cloud region.
type Region struct {
	// Name is unique identifier of the region.
	//
	// Example: us-west-2
	Name string `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`

	// DisplayName is a human-readable version of the Name.
	//
	// Generally the same as Name but it can differ based on the CloudProvider.
	// +optional
	DisplayName string `json:"displayName,omitempty" protobuf:"bytes,2,opt,name=displayName"`
}

// AWSCloudProviderSpec contains all AWS-specific configuration for a CloudProvider.
type AWSCloudProviderSpec struct {
	// SecretName contains a reference to the secret in the same namespace in
	// which the AWS credentials are stored.
	//
	// The referenced secret should contains the following fields in its data:
	// (1) accessKeyID: the access key ID of the user that should be used for this cloud provider.
	// (2) secretAccessKey: the secret access key associated with the access key ID.
	// (3) region: an AWS region which should be used by default for new resources.
	//
	// More info: https://docs.aws.amazon.com/general/latest/gr/aws-sec-cred-types.html
	SecretName string `json:"secretName" protobuf:"bytes,1,opt,name=secretName"`

	// Region can contain a AWS region which should be used by default for
	// new resources. If non-empty, it overrides any region set in the secret.
	Region string `json:"region,omitempty" protobuf:"bytes,2,opt,name=region"`
}

type AWSCloudProviderCredentials struct {
	// AccessKeyID is the access key ID of the user that should be used for
	// this cloud provider.
	//
	// More info: https://docs.aws.amazon.com/general/latest/gr/aws-sec-cred-types.html
	AccessKeyID string `json:"accessKeyID,omitempty" protobuf:"bytes,1,opt,name=accessKeyID"`

	// SecretAccessKey is the secret access key associated with the access key ID.
	//
	// More info: https://docs.aws.amazon.com/general/latest/gr/aws-sec-cred-types.html
	SecretAccessKey string `json:"secretAccessKey,omitempty" protobuf:"bytes,2,opt,name=secretAccessKey"`

	// Region can contain a AWS region which should be used by default for
	// new resources.
	Region string `json:"region,omitempty" protobuf:"bytes,3,opt,name=region"`
}

// AzureCloudProviderSpec contains all Azure-specific configuration for a CloudProvider.
type AzureCloudProviderSpec struct {
	// SecretName contains a reference to the secret in the same namespace in
	// which the Azure credentials are stored.
	//
	// The referenced secret should be of type AzureCloudProviderSecret and
	// therefore contains the following fields in its data:
	// (1) clientID: the unique identifier of the Azure user account.
	// (2) clientSecret: the secret access key associated with the client ID.
	// (3) subscriptionID: can contain a AWS region which should be used by default for new resources.
	// (3) tenantID: contains the tenant identifier.
	//
	// More info: https://docs.aws.amazon.com/general/latest/gr/aws-sec-cred-types.html
	SecretName string `json:"secretName" protobuf:"bytes,1,opt,name=secretName"`
}

type AzureCloudProviderCredentials struct {
	// ClientID is the unique identifier of the Azure user account.
	ClientID string `json:"clientID,omitempty" protobuf:"bytes,1,opt,name=clientID"`

	// ClientSecret is the secret access key associated with the client ID.
	ClientSecret string `json:"clientSecret,omitempty" protobuf:"bytes,2,opt,name=clientSecret"`

	// SubscriptionID is a unique alphanumeric string that identifies your Azure subscription.
	SubscriptionID string `json:"subscriptionID,omitempty" protobuf:"bytes,3,opt,name=subscriptionID"`

	// TenantID contains the tenant identifier.
	TenantID string `json:"tenantID,omitempty" protobuf:"bytes,4,opt,name=tenantID"`
}

// LocalCloudProviderSpec contains all configuration specific to a on-premise CloudProvider.
type LocalCloudProviderSpec struct{}
