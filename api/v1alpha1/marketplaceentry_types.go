package v1alpha1

import (
	apisv1alpha1 "github.com/kcp-dev/kcp/sdk/apis/apis/v1alpha1"
	extensionapiv1alpha1 "github.com/platform-mesh/extension-manager-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// MarketplaceEntrySpec defines the desired state of MarketplaceEntry.
type MarketplaceEntrySpec struct {

	// Installed indicates whether the marketplace entry is currently installed in the account
	Installed bool `json:"installed"`

	// ProviderMetadata contains metadata about the provider of the marketplace entry.
	ProviderMetadata extensionapiv1alpha1.ProviderMetadata `json:"providerMetadata"`

	// PermissionClaims are the permission claims associated with the marketplace entry.
	APIExport apisv1alpha1.APIExport `json:"apiExport"`
}

// MarketplaceEntryStatus defines the observed state of MarketplaceEntry.
type MarketplaceEntryStatus struct {
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster

// MarketplaceEntry is the Schema for the marketplaceentries API.
type MarketplaceEntry struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MarketplaceEntrySpec   `json:"spec,omitempty"`
	Status MarketplaceEntryStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// MarketplaceEntryList contains a list of MarketplaceEntry.
type MarketplaceEntryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MarketplaceEntry `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(GroupVersion,
			&MarketplaceEntry{},
			&MarketplaceEntryList{},
		)
		metav1.AddToGroupVersion(s, GroupVersion)
		return nil
	})
}
