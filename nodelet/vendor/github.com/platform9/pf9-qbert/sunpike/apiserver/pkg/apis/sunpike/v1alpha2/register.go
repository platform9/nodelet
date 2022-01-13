package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike"
)

const GroupName = sunpike.GroupName
const GroupVersion = "v1alpha2"

// SchemeGroupVersion is group version used to register these objects
var SchemeGroupVersion = schema.GroupVersion{Group: GroupName, Version: GroupVersion}

var (
	// TODO: move SchemeBuilder with zz_generated.deepcopy.go to k8s.io/api.
	// localSchemeBuilder and AddToScheme will stay in k8s.io/kubernetes.
	SchemeBuilder      runtime.SchemeBuilder
	localSchemeBuilder = &SchemeBuilder
	AddToScheme        = localSchemeBuilder.AddToScheme
)

func init() {
	// We only register manually written functions here. The registration of the
	// generated functions takes place in the generated files. The separation
	// makes the code compile even when the generated files are missing.
	localSchemeBuilder.Register(addKnownTypes, addDefaultingFuncs)
}

// Adds the list of known types to the given scheme.
func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&Host{},
		&HostList{},
		&Cluster{},
		&ClusterList{},
		&CloudProvider{},
		&CloudProviderList{},
		&ClusterAddon{},
		&ClusterAddonList{},
		&ClusterProfile{},
		&ClusterProfileList{},
		&ClusterProfileBinding{},
		&ClusterProfileBindingList{},
		&ClusterProfileDetail{},
		&ClusterProfileBindingDetail{},
	)
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}

// Resource takes an unqualified resource and returns a Group qualified GroupResource
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}
