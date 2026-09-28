package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// CustomHttprouteConfigSpec defines the desired state of CustomHttprouteConfig
type CustomHttprouteConfigSpec struct {
	// +kubebuilder:default=true
	IncludeDefaultRoute bool        `json:"includeDefaultRoute,omitempty"`
	Route               RouteConfig `json:"route,omitempty"`
}
type RouteConfig struct {
	HeaderMatches []RouteHeaderMatch `json:"headerMatches,omitempty"`
	Backends      []RouteBackend     `json:"backends,omitempty"`
}

type RouteHeaderMatch struct {
	// +kubebuilder:validation:MinLength=1
	Name            string `json:"name"`
	ApplicationType string `json:"applicationType,omitempty"`
	// +kubebuilder:validation:MinLength=1
	Value string `json:"value"`
}
type RouteBackend struct {
	// +required
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	Port       int32  `json:"port"`
	PathPrefix string `json:"pathPrefix,omitempty"`
}

// CustomHttprouteConfigStatus defines the observed state of CustomHttprouteConfig.
type CustomHttprouteConfigStatus struct {
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// CustomHttprouteConfig is the Schema for the customhttprouteconfigs API
type CustomHttprouteConfig struct {
	metav1.TypeMeta `json:",inline"`

	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// +required
	Spec CustomHttprouteConfigSpec `json:"spec"`

	// +optional
	Status CustomHttprouteConfigStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// CustomHttprouteConfigList contains a list of CustomHttprouteConfig
type CustomHttprouteConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []CustomHttprouteConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(SchemeGroupVersion, &CustomHttprouteConfig{}, &CustomHttprouteConfigList{})
		return nil
	})
}
