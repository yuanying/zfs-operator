/*
Copyright 2020 O.Yuanying <yuanying@fraction.jp>

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

// VolumeSpec defines the desired state of Volume
type VolumeSpec struct {
	// NodeName is a node name where the volume will be placed.
	// +kubebuilder:validation:Required
	NodeName string `json:"nodeName,omitempty"`

	// VolumeName is a ZVOL name
	// +kubebuilder:validation:Pattern=^[-A-Za-z0-9]+(\/[-A-Za-z0-9]+)+$
	// +kubebuilder:validation:Required
	VolumeName string `json:"volumeName,omitempty"`

	// Capacity represents the desired resources of the volume
	// +kubebuilder:validation:Required
	Capacity corev1.ResourceList `json:"capacity,omitempty"`

	// Properties represents the desired zfs properties
	Properties map[string]string `json:"properties,omitempty"`
}

// VolumeStatus defines the observed state of Volume
type VolumeStatus struct {
	// Conditions are the current state of Volume
	Conditions []VolumeCondition `json:"conditions,omitempty"`
}

// VolumeConditionType is a valid value for VolumeCondition.Type
type VolumeConditionType string

const (
	// VolumeConditionTypeReady means that API server on the Volume is ready for service.
	VolumeConditionReady VolumeConditionType = "Ready"
)

type VolumeCondition struct {
	// Type is the type of this condition.
	Type VolumeConditionType `json:"type,omitempty"`
	// Status is the status of this condition.
	Status corev1.ConditionStatus `json:"status,omitempty"`
	// LastTransitionTime is the last time the condition transitioned from one status to another.
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	// Reason is the one-word, CamelCase reason about the last transition.
	Reason string `json:"reason,omitempty"`
	// Message is human readable message about the last transition.
	Message string `json:"message,omitempty"`
}

// +kubebuilder:object:root=true

// Volume is the Schema for the volumes API
// +kubebuilder:resource:shortName=vol,scope=Cluster
// +kubebuilder:subresource:status
type Volume struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VolumeSpec   `json:"spec,omitempty"`
	Status VolumeStatus `json:"status,omitempty"`
}

// SetCondition sets condition of type condType with empty reason and message.
func (volume *Volume) SetCondition(condType VolumeConditionType, status corev1.ConditionStatus, t metav1.Time) {
	volume.SetConditionReason(condType, status, "", "", t)
}

// SetConditionReason is similar to setCondition, but it takes reason and message.
func (volume *Volume) SetConditionReason(condType VolumeConditionType, status corev1.ConditionStatus, reason, msg string, t metav1.Time) {
	cond := volume.GetCondition(condType)
	if cond == nil {
		volume.Status.Conditions = append(volume.Status.Conditions, VolumeCondition{
			Type: condType,
		})
		cond = &volume.Status.Conditions[len(volume.Status.Conditions)-1]
	}

	if cond.Status != status {
		cond.Status = status
		cond.LastTransitionTime = t
	}

	cond.Reason = reason
	cond.Message = msg
}

// GetCondition returns condition of type condType if it exists.  Otherwise returns nil.
func (volume *Volume) GetCondition(condType VolumeConditionType) *VolumeCondition {
	for i := range volume.Status.Conditions {
		cond := &volume.Status.Conditions[i]
		if cond.Type == condType {
			return cond
		}
	}
	return nil
}

func (volume *Volume) Ready() bool {
	r := volume.GetCondition(VolumeConditionReady)
	if r != nil {
		if r.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

// +kubebuilder:object:root=true

// VolumeList contains a list of Volume
type VolumeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Volume `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Volume{}, &VolumeList{})
}
