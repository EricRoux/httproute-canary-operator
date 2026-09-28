package v1alpha1

import (
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// CustomRolloutConfigSpec defines the desired state of CustomRolloutConfig
type CustomRolloutConfigSpec struct {
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:default=canary
	ServiceCanaryPrefix string                                    `json:"serviceCanaryPrefix"`
	ApplicationType     string                                    `json:"applicationType,omitempty"`
	TargetRef           autoscalingv1.CrossVersionObjectReference `json:"targetRef"`
}

// CustomRolloutConfigStatus defines the observed state of CustomRolloutConfig.
type CustomRolloutConfigStatus struct {
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// CustomRolloutConfig is the Schema for the customrolloutconfigs API
type CustomRolloutConfig struct {
	metav1.TypeMeta `json:",inline"`

	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// +required
	Spec CustomRolloutConfigSpec `json:"spec"`

	// +optional
	Status CustomRolloutConfigStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// CustomRolloutConfigList contains a list of CustomRolloutConfig
type CustomRolloutConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []CustomRolloutConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(SchemeGroupVersion, &CustomRolloutConfig{}, &CustomRolloutConfigList{})
		return nil
	})
}
