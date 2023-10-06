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

// ServicesPlatformSpec describes the desired service configuration for "prod" workflows.
type ServicesPlatformSpec struct {
	// Deploys the Data Index service for use by "prod" profile workflows.
	// +optional
	DataIndex *ServicesPodTemplateSpec `json:"dataIndex,omitempty"`
	// Persists service(s) to a datasource of choice. Ephemeral by default.
	// +optional
	Persistence *PersistenceOptions `json:"persistence,omitempty"`
}

// ServicesPodTemplateSpec describes the desired custom Kubernetes PodTemplate definition for the deployed flow.
//
// The FlowContainer describes the container where the actual service is running. It will override any default definitions.
// For example, to override the image one can use `.spec.services.dataIndex.container.image = my/image:tag`.
type ServicesPodTemplateSpec struct {
	// Container is the Kubernetes container where the service should run.
	// One can change this attribute in order to override the defaults provided by the operator.
	// +optional
	Container *FlowContainer `json:"container,omitempty"`
	// Describes the PodSpec for the internal service deployment based on the default Kubernetes PodSpec API
	// +optional
	FlowPodSpec `json:",inline"`
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`
	// Determines whether "prod" profile workflows should be configured to use this service
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
}

// PersistenceOptions configure the services to persist to a datasource of choice
// +kubebuilder:validation:MaxProperties=1
type PersistenceOptions struct {
	// Connect configured services to a postgresql database.
	// +optional
	PostgreSql *PersistencePostgreSql `json:"postgresql,omitempty"`
	// +optional
	// Infinispan *PersistenceInfinispan `json:"infinispan,omitempty"`
}

// PersistencePostgreSql ...
type PersistencePostgreSql struct {
	// Secret reference to the database user credentials
	SecretRef PostgreSqlSecretOptions `json:"secretRef"`
	// Service reference to postgresql datasource
	ServiceRef PostgreSqlServiceOptions `json:"serviceRef"`
	// Name of postgresql database to be used. Defaults to "sonataflow"
	// +optional
	DatabaseName string `json:"databaseName,omitempty"`
}

// PostgreSqlSecretOptions ...
type PostgreSqlSecretOptions struct {
	// Name of the postgresql credentials secret.
	Name string `json:"name"`
	// Defaults to POSTGRESQL_USER
	// +optional
	UserKey string `json:"userKey,omitempty"`
	// Defaults to POSTGRESQL_PASSWORD
	// +optional
	PasswordKey string `json:"passwordKey,omitempty"`
}

// PostgreSqlServiceOptions ...
type PostgreSqlServiceOptions struct {
	// Name of the postgresql k8s service.
	Name string `json:"name"`
	// Namespace of the postgresql k8s service. Defaults to the SonataFlowPlatform's local namespace.
	// +optional
	Namespace string `json:"namespace,omitempty"`
	// Port to use when connecting to the postgresql k8s service. Defaults to 5432.
	// +optional
	Port *int `json:"port,omitempty"`
}
