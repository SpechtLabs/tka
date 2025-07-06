package v1alpha1

import (
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ClusterRoleMapping struct {
	ClusterRole string `json:"clusterRole"`
	CapRole     string `json:"capRule"`
}

type TkaSpec struct {
	// +kubebuilder:validation:Required
	RoleMap []ClusterRoleMapping `json:"roleMap"`

	// +kubebuilder:validation:Optional
	AdditionalClusterRole []rbacv1.ClusterRole `json:"additionalClusterRoles,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced

type TKA struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec TkaSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

type TKAList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TKA `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TKA{}, &TKAList{})
}
