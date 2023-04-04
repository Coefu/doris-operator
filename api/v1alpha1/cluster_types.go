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

// ClusterSpec defines the desired state of Cluster
type ClusterSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// It is not allowed to modify a cluster when it has already been created,
	// If data needs to be migrated, manually migrate the data
	StorageClass *string `json:"storageClass,omitempty"`

	// Can change it when the cluster is running
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`

	Fe Feconfig `json:"fe"`

	Be Beconfig `json:"be"`
}

type Feconfig struct {

	// Fe replicas uses the Paxos protocol, but it can only be an odd number up to 9,like: 1,3,5,7,9
	Replicas *int32 `json:"replicas"`

	// It can be modified at run time because of version update requirements
	// TODO: This is an experimental feature and has not been implemented yet
	Image string `json:"image"`

	// As the image changes, the command line may change with it
	Command []string `json:"command,omitempty"`

	// As the fe Web domain name, it can be changed at any time after the cluster runs
	Domain string `json:"domain,omitempty"`

	// It is not allowed to modify a cluster when it has already been created
	Storage string `json:"storage,omitempty"`

	// You can change it when the cluster is running
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
}

type Beconfig struct {

	// Be Replicas can have an unlimited number of replicas, but are limited by network initialization,
	// currently up to 150, and can Be increased by modifying the network capacity.
	Replicas *int32 `json:"replicas"`

	// It can be modified at run time because of version update requirements
	Image string `json:"image"`

	// As the image changes, the command line may change with it
	Command []string `json:"command,omitempty"`

	// Should be able to cancel and open at any time
	// TODO: This is an experimental feature and has not been implemented yet
	Autoscaling bool `json:"autoscaling,omitempty"`

	// It is not allowed to modify a cluster when it has already been created
	Storage string `json:"storage,omitempty"`

	// You can change it when the cluster is running
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
}

// ClusterStatus defines the observed state of Cluster
type ClusterStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Fe_replicas int32    `json:"fe-replicas"`
	Be_replicas int32    `json:"be-replicas"`
	Brokers     []string `json:"brokers,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Cluster is the Schema for the clusters API
type Cluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterSpec   `json:"spec,omitempty"`
	Status ClusterStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ClusterList contains a list of Cluster
type ClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Cluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Cluster{}, &ClusterList{})
}
