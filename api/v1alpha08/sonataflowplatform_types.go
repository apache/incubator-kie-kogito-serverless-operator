/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

package v1alpha08

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/apache/incubator-kie-kogito-serverless-operator/api"
)

const (
	// SonataFlowPlatformKind is the Kind name of the SonataFlowPlatform CR
	SonataFlowPlatformKind string = "SonataFlowPlatform"
)

// SonataFlowPlatformSpec defines the desired state of SonataFlowPlatform
// +k8s:openapi-gen=true
type SonataFlowPlatformSpec struct {
	// Build Attributes for building workflows in the target platform
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Build"
	Build BuildPlatformSpec `json:"build,omitempty"`
	// DevMode Attributes for running workflows in devmode (immutable, no build required)
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="DevMode"
	DevMode DevModePlatformSpec `json:"devMode,omitempty"`
	// Services attributes for deploying supporting applications like Data Index & Job Service.
	// Only workflows without the `sonataflow.org/profile: dev` annotation will be configured to use these service(s).
	// Setting this will override the use of any cluster-scoped services that might be defined via `SonataFlowClusterPlatform`.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Services"
	Services *ServicesPlatformSpec `json:"services,omitempty"`
	// Persistence defines the platform persistence configuration. When this field is set,
	// the configuration is used as the persistence for platform services and sonataflow instances
	// that don't provide one of their own.
	// +optional
	Persistence *PlatformPersistenceSpec `json:"persistence,omitempty"`
}

// PlatformPersistenceSpec configures the DataBase support for both platform services and workflows. For services, it allows
// configuring a generic database connectivity if the service does not come with its own configured. In case of the workflow
// the configuration is defined by the image name of the supported database. When the workflow is configured to require
// a database backend type supported by the platform (that is, the platform CR contains an image name for that database type)
// , the workflow pod is configured to contain a sidecar container running the database image defined here, and the workflow
// JDBC connectivity is setup to point to the DB container.
// +optional
type PlatformPersistenceSpec struct {
	// Connect configured services to a postgresql database.
	// +optional
	PostgreSQL *PostgreSQLPlatformSpec `json:"postgresql,omitempty"`
}

// PostgreSQLPlatformSpec provides the generic configuration details to configure the JDBC URL and establish a connection for each managed services when they don't provide their own configuration.
type PostgreSQLPlatformSpec struct {
	// SecretRef contains the database user credentials
	SecretRef SecretReference `json:"secretRef"`
	// ServiceRef contains the K8s service name and namespace location of the PostgreSQL service.
	ServiceRef ServiceReference `json:"serviceRef,omitempty"`
	// Name of postgresql database to be used. Defaults to "sonataflow"
	// +kubebuilder:default:=sonataflow
	DatabaseName string `json:"databaseName,omitempty"`
}

type ServiceReference struct {
	// Name contains the name of the kubernetes service. This field is mandatory.
	// +required
	Name string `json:"name"`
	// Namespace contains the name of the namespace where the kubernetes service resides. This field is optional.
	// +optional
	Namespace string `json:"namespace"`
	// Port contains the port number associated to the kubernetes service. This field is mandatory.
	// +required
	Port int `json:"port,omitempty"`
}

// SecretReference use of a secret to store the credentials to authenticate in the JDBC connection.
type SecretReference struct {
	// Name of the postgresql credentials secret. This field is mandatory.
	Name string `json:"name"`
	// +optional
	UserKey string `json:"userKey,omitempty"`
	// +optional
	PasswordKey string `json:"passwordKey,omitempty"`
}

// PlatformCluster is the kind of orchestration cluster the platform is installed into
// +kubebuilder:validation:Enum=kubernetes;openshift
type PlatformCluster string

const (
	// PlatformClusterOpenShift is used when targeting an OpenShift cluster
	PlatformClusterOpenShift PlatformCluster = "openshift"
	// PlatformClusterKubernetes is used when targeting a Kubernetes cluster
	PlatformClusterKubernetes PlatformCluster = "kubernetes"
)

const (
	PlatformCreatingReason   = "Creating"
	PlatformWarmingReason    = "Warming"
	PlatformFailureReason    = "Failure"
	PlatformDuplicatedReason = "Duplicated"
)

// SonataFlowPlatformStatus defines the observed state of SonataFlowPlatform
// +k8s:openapi-gen=true
type SonataFlowPlatformStatus struct {
	api.Status `json:",inline"`
	// Cluster what kind of cluster you're running (ie, plain Kubernetes or OpenShift)
	//+operator-sdk:csv:customresourcedefinitions:type=status,displayName="cluster"
	Cluster PlatformCluster `json:"cluster,omitempty"`
	// Version the operator version controlling this Platform
	//+operator-sdk:csv:customresourcedefinitions:type=status,displayName="version"
	Version string `json:"version,omitempty"`
	// Info generic information related to the build
	//+operator-sdk:csv:customresourcedefinitions:type=status,displayName="info"
	Info map[string]string `json:"info,omitempty"`
	// ClusterPlatformRef information related to the (optional) active SonataFlowClusterPlatform
	//+operator-sdk:csv:customresourcedefinitions:type=status,displayName="clusterPlatformRef"
	ClusterPlatformRef *SonataFlowClusterPlatformRefStatus `json:"clusterPlatformRef,omitempty"`
}

// SonataFlowClusterPlatformRefStatus information related to the (optional) active SonataFlowClusterPlatform
// +k8s:openapi-gen=true
type SonataFlowClusterPlatformRefStatus struct {
	// Name of the active SonataFlowClusterPlatform
	Name string `json:"name,omitempty"`
	// PlatformRef displays which SonataFlowPlatform has been referenced by the active SonataFlowClusterPlatform
	PlatformRef SonataFlowPlatformRef `json:"platformRef,omitempty"`
	// Services displays which cluster-wide services are being used by this SonataFlowPlatform
	Services *PlatformServicesStatus `json:"services,omitempty"`
}

// PlatformServicesStatus displays which cluster-wide services are being used by a SonataFlowPlatform
// +k8s:openapi-gen=true
type PlatformServicesStatus struct {
	// DataIndexRef displays information on the cluster-wide Data Index service
	DataIndexRef *PlatformServiceRefStatus `json:"dataIndexRef,omitempty"`
	// JobServiceRef displays information on the cluster-wide Job Service
	JobServiceRef *PlatformServiceRefStatus `json:"jobServiceRef,omitempty"`
}

// PlatformServiceRefStatus displays information on a cluster-wide service
// +k8s:openapi-gen=true
type PlatformServiceRefStatus struct {
	// Url displays the base url of a cluster-wide service
	Url string `json:"url,omitempty"`
}

func (in *SonataFlowPlatformStatus) GetTopLevelConditionType() api.ConditionType {
	return api.SucceedConditionType
}

func (in *SonataFlowPlatformStatus) IsReady() bool {
	return in.GetTopLevelCondition().IsTrue()
}

func (in *SonataFlowPlatformStatus) GetTopLevelCondition() *api.Condition {
	return in.GetCondition(in.GetTopLevelConditionType())
}

func (in *SonataFlowPlatformStatus) Manager() api.ConditionsManager {
	return api.NewConditionManager(in, api.SucceedConditionType)
}

func (in *SonataFlowPlatformStatus) IsCreating() bool {
	cond := in.GetTopLevelCondition()
	return cond.IsFalse() && cond.Reason == PlatformCreatingReason
}

func (in *SonataFlowPlatformStatus) IsWarming() bool {
	cond := in.GetTopLevelCondition()
	return cond.IsFalse() && cond.Reason == PlatformWarmingReason
}

func (in *SonataFlowPlatformStatus) IsDuplicated() bool {
	cond := in.GetTopLevelCondition()
	return cond.IsFalse() && cond.Reason == PlatformDuplicatedReason
}

func (in *SonataFlowPlatformStatus) IsFailure() bool {
	cond := in.GetTopLevelCondition()
	return cond.IsFalse() && cond.Reason == PlatformFailureReason
}

// SonataFlowPlatform is the descriptor for the workflow platform infrastructure.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:object:generate=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName={"sfp", "sfplatform", "sfplatforms"}
// +kubebuilder:printcolumn:name="Cluster",type=string,JSONPath=`.status.cluster`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=='Succeed')].status`
// +kubebuilder:printcolumn:name="Reason",type=string,JSONPath=`.status.conditions[?(@.type=='Succeed')].reason`
// +operator-sdk:csv:customresourcedefinitions:resources={{Namespace,v1,"The Namespace controlled by the platform"}}
type SonataFlowPlatform struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SonataFlowPlatformSpec   `json:"spec,omitempty"`
	Status SonataFlowPlatformStatus `json:"status,omitempty"`
}

// SonataFlowPlatformList contains a list of SonataFlowPlatform
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:object:generate=true
type SonataFlowPlatformList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SonataFlowPlatform `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SonataFlowPlatform{}, &SonataFlowPlatformList{})
}
