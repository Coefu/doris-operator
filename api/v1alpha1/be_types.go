/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// BeSpec defines the desired state of Be
type BeSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Image            string                        `json:"image"`
	Command          []string                      `json:"command,omitempty"`
	Autoscaling      bool                          `json:"autoscaling,omitempty"`
	Storage          string                        `json:"storage,omitempty"`
	Resources        corev1.ResourceRequirements   `json:"resources,omitempty"`
	StorageClass     *string                       `json:"storageClass,omitempty"`
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`
}

// BeStatus defines the observed state of Be
type BeStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Cluster string `json:"cluster"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Be is the Schema for the bes API
type Be struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BeSpec   `json:"spec,omitempty"`
	Status BeStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// BeList contains a list of Be
type BeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Be `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Be{}, &BeList{})
}
