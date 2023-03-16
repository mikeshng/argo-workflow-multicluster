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
	argov1alpha1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// WorkflowStatusResultSpec defines the desired state of WorkflowStatusResult
type WorkflowStatusResultSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of WorkflowStatusResult. Edit workflowstatusresult_types.go to remove/update
	Foo string `json:"foo,omitempty"`
}

// WorkflowStatusResultStatus defines the observed state of WorkflowStatusResult
type WorkflowStatusResultStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// WorkflowStatusResult is the Schema for the workflowstatusresults API
type WorkflowStatusResult struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	WorkflowStatus argov1alpha1.WorkflowStatus `json:"workflowStatus" protobuf:"bytes,2,opt,name=workflowStatus"`
}

//+kubebuilder:object:root=true

// WorkflowStatusResultList contains a list of WorkflowStatusResult
type WorkflowStatusResultList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WorkflowStatusResult `json:"items" protobuf:"bytes,2,rep,name=items"`
}

func init() {
	SchemeBuilder.Register(&WorkflowStatusResult{}, &WorkflowStatusResultList{})
}
