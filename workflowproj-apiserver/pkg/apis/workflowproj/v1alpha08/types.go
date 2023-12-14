// Copyright 2023 Red Hat, Inc. and/or its affiliates
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1alpha08

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SonataFlowProjInstance ...
// +genclient
// +genclient:onlyVerbs=get
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type SonataFlowProjInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Spec   SonataFlowProjInstanceSpec   `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
	Status SonataFlowProjInstanceStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

type SonataFlowProjInstanceSpec struct {
}

type SonataFlowProjInstanceStatus struct {
	ObjectsRef ProjObjectsRef `json:"objectsRef,omitempty"`
}

type ProjObjectsRef struct {
	Workflow   v1.LocalObjectReference   `json:"workflow,omitempty"`
	Properties v1.LocalObjectReference   `json:"properties,omitempty"`
	Resources  []v1.LocalObjectReference `json:"resources,omitempty"`
}

// SonataFlowProjInstanceList ...
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type SonataFlowProjInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Items []SonataFlowProjInstance `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// SonataFlowProjInstantiateRequestOptions ...
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type SonataFlowProjInstantiateRequestOptions struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Main workflow file name in the package sent to the server in case of multiple workflow files in the same package.
	//
	// The main workflow file will be deployed as a new SonataFlow custom resource, the remaining as subflows within the main workflow.
	MainWorkflowFile string `json:"mainWorkflowFile,omitempty"`
}
