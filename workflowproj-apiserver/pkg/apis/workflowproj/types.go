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

package workflowproj

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SonataFlowProjInstance ...
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type SonataFlowProjInstance struct {
	metav1.TypeMeta
	metav1.ObjectMeta

	Spec   SonataFlowProjInstanceSpec
	Status SonataFlowProjInstanceStatus
}

type SonataFlowProjInstanceSpec struct {
}

type SonataFlowProjInstanceStatus struct {
	ObjectsRef ProjObjectsRef
}

type ProjObjectsRef struct {
	Workflow   v1.LocalObjectReference
	Properties v1.LocalObjectReference
	Resources  []v1.LocalObjectReference
}

// SonataFlowProjInstanceList ...
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type SonataFlowProjInstanceList struct {
	metav1.TypeMeta
	metav1.ListMeta

	Items []SonataFlowProjInstance
}

// SonataFlowProjInstantiateRequestOptions ...
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type SonataFlowProjInstantiateRequestOptions struct {
	metav1.TypeMeta
	metav1.ObjectMeta

	// Main workflow file name in the package sent to the server in case of multiple workflow files in the same package.
	//
	// The main workflow file will be deployed as a new SonataFlow custom resource, the remaining as subflows within the main workflow.
	MainWorkflowFile string
}
