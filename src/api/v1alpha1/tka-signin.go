// +kubebuilder:object:generate=true
// +groupName=tka.specht-labs.de

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type TkaSigninSpec struct {
	Username       string `json:"username"`
	Role           string `json:"role"`
	ValidityPeriod string `json:"validity_period"`
}

type TkaSigninStatus struct {
	Provisioned bool   `json:"provisioned"`
	ValidUntil  string `json:"valid_until"`
	SignedInAt  string `json:"signed_in"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=signin
// +kubebuilder:printcolumn:name="provisioned",type=boolean,JSONPath=`.status.provisioned`,description="true if the user signin was processed and the ServiceAccount was created"
// +kubebuilder:printcolumn:name="since",type=string,JSONPath=`.status.signed_in`,description="timestamp when the user signed in"
// +kubebuilder:printcolumn:name="period",type=string,JSONPath=`.status.validity_period`,description="For how long this session is valid"
// +kubebuilder:printcolumn:name="until",type=string,JSONPath=`.spec.valid_until`,description="timestamp until when the signin is valid"
type TkaSignin struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TkaSigninSpec   `json:"spec,omitempty"`
	Status TkaSigninStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type TkaSigninList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TkaSignin `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TkaSignin{}, &TkaSigninList{})
}
