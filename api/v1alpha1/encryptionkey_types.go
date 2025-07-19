/*
Copyright 2025.

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// EncryptionKeySpec defines the desired state of EncryptionKey.
type EncryptionKeySpec struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=2048
	// KeyID is the identifier of the KMS key to be used for encryption.
	KeyID string `json:"KeyID,omitempty"`
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=2048
	// AccountID is the AWS account ID that will be assumed to access the KMS key.
	AccountID string `json:"AccountID,omitempty"`
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	// KubernetesServiceAccount is the Kubernetes service account where we will request a token from.
	KubernetesServiceAccount string `json:"KubernetesServiceAccount,omitempty"`
}

// EncryptionKeyStatus defines the observed state of EncryptionKey.
type EncryptionKeyStatus struct {
	LastReconciledTime *metav1.Time `json:"lastReconciledTime,omitempty"`
	// KeyMetadata contains metadata about the KMS key
	Arn                   string       `json:"arn,omitempty"`
	CreationDate          *metav1.Time `json:"creationDate,omitempty"`
	Description           string       `json:"description,omitempty"`
	Enabled               bool         `json:"enabled,omitempty"`
	KeyState              string       `json:"keyState,omitempty"`
	KeyUsage              string       `json:"keyUsage,omitempty"`
	Origin                string       `json:"origin,omitempty"`
	DeletionDate          *metav1.Time `json:"deletionDate,omitempty"`
	ValidTo               *metav1.Time `json:"validTo,omitempty"`
	CustomKeyStoreId      string       `json:"customKeyStoreId,omitempty"`
	CloudHsmClusterId     string       `json:"cloudHsmClusterId,omitempty"`
	KeyManager            string       `json:"keyManager,omitempty"`
	CustomerMasterKeySpec string       `json:"customerMasterKeySpec,omitempty"`
	KeySpec               string       `json:"keySpec,omitempty"`
	EncryptionAlgorithms  []string     `json:"encryptionAlgorithms,omitempty"`
	SigningAlgorithms     []string     `json:"signingAlgorithms,omitempty"`
	MultiRegion           bool         `json:"multiRegion,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// EncryptionKey is the Schema for the encryptionkeys API.
type EncryptionKey struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EncryptionKeySpec   `json:"spec,omitempty"`
	Status EncryptionKeyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// EncryptionKeyList contains a list of EncryptionKey.
type EncryptionKeyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EncryptionKey `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EncryptionKey{}, &EncryptionKeyList{})
}
