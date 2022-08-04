/*
Copyright 2022.

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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

type Constant struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Timeout struct {
	WorkflowExecTimeout string `json:"workflowExecTimeout,omitempty"`
	StateExecTimeout    string `json:"stateExecTimeout,omitempty"`
	ActionExecTimeout   string `json:"actionExecTimeout,omitempty"`
	BranchExecTimeout   string `json:"branchExecTimeout,omitempty"`
	EventTimeout        string `json:"eventTimeout,omitempty"`
}

type Error struct {
	Name        string `json:"name"`
	Code        string `json:"code"`
	Description string `json:"description,omitempty"`
}

type BasicAuthProperties struct {
	Username string     `json:"username"`
	Password string     `json:"password"`
	Metadata []Metadata `json:"metadata,omitempty"`
}

type BearerAuthProperties struct {
	Token    string     `json:"token"`
	Metadata []Metadata `json:"metadata,omitempty"`
}

type GrantType string

const (
	PasswordGrantType          GrantType = "password"
	ClientCredentialsGrantType GrantType = "clientCredentials"
	TokenExchangeGrantType     GrantType = "tokenExchange"
)

type OAuth2Properties struct {
	Authority        string     `json:"basic,omitempty"`
	GrantType        GrantType  `json:"grantType"`
	ClientId         string     `json:"clientId"`
	ClientSecret     string     `json:"clientSecret"`
	Scopes           []string   `json:"scopes,omitempty"`
	Username         string     `json:"username,omitempty"`
	Password         string     `json:"password,omitempty"`
	Audiences        []string   `json:"audiences,omitempty"`
	SubjectToken     string     `json:"subjectToken,omitempty"`
	RequestedSubject string     `json:"requestedSubject,omitempty"`
	RequestedIssuer  string     `json:"requestedIssuer,omitempty"`
	Metadata         []Metadata `json:"metadata,omitempty"`
}

type AuthProperties struct {
	Basic  BasicAuthProperties  `json:"basic,omitempty"`
	Bearer BearerAuthProperties `json:"bearer,omitempty"`
	Oauth2 OAuth2Properties     `json:"oauth2,omitempty"`
}

type Auth struct {
	Name       string         `json:"name"`
	Scheme     string         `json:"scheme"`
	Properties AuthProperties `json:"properties"`
}

type EventKind string

const (
	ProducedEventKind EventKind = "produced"
	ConsumedEventKind EventKind = "consumed"
)

type EventCorrelationRule struct {
	ContextAttributeName  string `json:"contextAttributeName"`
	ContextAttributeValue string `json:"contextAttributeValue"`
}

type Metadata struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Event struct {
	Name        string                 `json:"name"`
	Source      string                 `json:"source"`
	Type        string                 `json:"type"`
	Kind        EventKind              `json:"kind,omitempty"`
	Correlation []EventCorrelationRule `json:"correlation,omitempty"`
	DataOnly    bool                   `json:"dataOnly,omitempty"`
	Metadata    []Metadata             `json:"metadata,omitempty"`
}

type FunctionType string

const (
	RestFunctionType       FunctionType = "rest"
	AsyncApiFunctionType   FunctionType = "asyncapi"
	RpcFunctionType        FunctionType = "rpc"
	GraphQLFunctionType    FunctionType = "graphql"
	ODataFunctionType      FunctionType = "odata"
	ExpressionFunctionType FunctionType = "odata"
)

type Function struct {
	Name      string       `json:"name"`
	Operation string       `json:"operation"`
	Type      FunctionType `json:"type"`
	AuthRef   string       `json:"authRef,omitempty"`
	Metadata  []Metadata   `json:"metadata,omitempty"`
}

type Retry struct {
	Name        string `json:"name"`
	Delay       string `json:"delay,omitempty"`
	MaxAttempts int    `json:"maxAttempts,omitempty"`
	MaxDelay    string `json:"maxDelay,omitempty"`
	Increment   string `json:"increment,omitempty"`
	Multiplier  string `json:"multiplier,omitempty"`
	Jitter      string `json:"jitter,omitempty"`
}

type StateType string

const (
	EventStateType     StateType = "event"
	OperationStateType StateType = "operation"
	SwitchStateType    StateType = "switch"
	SleepStateType     StateType = "sleep"
	ParallelStateType  StateType = "parallel"
	InjectStateType    StateType = "inject"
	ForEachStateType   StateType = "foreach"
)

type ActionMode struct{}

//TODO: Define ActionMode values (Should actions be performed sequentially or in parallel?)

type Action struct{}

type CompletionType struct{}

//TODO: Define CompletionType values (Option types on how to complete branch execution. Default is "allOf")

type IterationMode struct{}

//TODO: Define IterationMode values (Specifies how iterations are to be performed (sequentially or in parallel). Default is parallel)

type EventDataFilter struct{}

//TODO: Define EventDataFilter (Callback EventDataFilter definition)

type Timeouts struct{}

//TODO: Define Timeouts (State specific timeout settings)

type StateDataFilter struct{}

//TODO: Define StateDataFilter (State data filter definition)

type Transition struct{}

//TODO: Define Transition

type State struct {
	Name string `json:"name"`
	// +kubebuilder:validation:Enum:=event;operation;switch;sleep;parallel;inject;foreach
	Type StateType `json:"type"`
	// +optional
	Exclusive *bool `json:"exclusive,omitempty"`
	// +optional
	ActionMode *ActionMode `json:"actionMode,omitempty"`
	// +optional
	Actions *[]Action `json:"actions,omitempty"`
	Data    []byte    `json:"data,omitempty"`
	//TODO: Define a type for DataCondition objects
	DataConditions []string `json:"dataConditions,omitempty"`
	//TODO: Define a type for EventContitions objects
	EventConditions []string `json:"eventConditions,omitempty"`
	//TODO: Define a type for DefaultCondition object
	DefaultCondition string `json:"defaultCondition,omitempty"`
	//TODO: Double-check that we can use the Event type here
	OnEvents []Event  `json:"onEvents,omitempty"`
	Duration string   `json:"duration,omitempty"`
	Branches []string `json:"branches,omitempty"`
	// +optional
	CompletionType CompletionType `json:"completionType,omitempty"`
	// +optional
	NumCompleted     *int   `json:"numCompleted,omitempty"`
	InputCollection  string `json:"inputCollection,omitempty"`
	OutputCollection string `json:"outputCollection,omitempty"`
	// +optional
	IterationParam *string `json:"iterationParam,omitempty"`
	// +optional
	BatchSize *int `json:"batchSize,omitempty"`
	// +optional
	Mode     IterationMode `json:"mode,omitempty"`
	EventRef string        `json:"eventRef,omitempty"`
	// +optional
	EventDataFilter *EventDataFilter `json:"eventDataFilter,omitempty"`
	// +optional
	Timeouts *Timeouts `json:"timeouts,omitempty"`
	// +optional
	StateDataFilter *StateDataFilter `json:"stateDataFilter,omitempty"`
	// +optional
	Transition *Transition `json:"transition,omitempty"`
	// +optional
	OnErrors *[]string `json:"onErrors,omitempty"`
	// +optional
	End *bool `json:"end,omitempty"`
	// +optional
	CompensatedBy *string `json:"compensatedBy,omitempty"`
	// +optional
	UsedForCompensation *bool `json:"usedForCompensation,omitempty"`
	// +optional
	Metadata *Metadata `json:"metadata,omitempty"`
}

// KogitoServerlessWorkflowSpec defines the desired state of KogitoServerlessWorkflow
type KogitoServerlessWorkflowSpec struct {
	Constants   []Constant   `json:"conditions,omitempty"`
	Secrets     *[]v1.Secret `json:"secrets,omitempty"`
	Start       string       `json:"start"`
	Timeouts    []Timeout    `json:"timeouts,omitempty"`
	Errors      []Error      `json:"errors,omitempty"`
	KeepAlive   bool         `json:"keepAlive"`
	Auth        Auth         `json:"auth,omitempty"`
	Events      *[]Event     `json:"events,omitempty"`
	Functions   []Function   `json:"functions,omitempty"`
	AutoRetries bool         `json:"autoRetries"`
	Retries     Retry        `json:"retries,omitempty"`
	States      []State      `json:"states"`
}

type Endpoint struct {
	IP       string `json:"ip,omitempty"`
	Port     int    `json:"port,omitempty"`
	PortName string `json:"portName,omitempty"`
	Protocol string `json:"protocol,omitempty"` // "TCP" or "UDP"; never empty
}

type StatusCondition string

const (
	BuildingStatusCondition StatusCondition = "Building"
	ReadyStatusCondition    StatusCondition = "Ready"
)

// KogitoServerlessWorkflowStatus defines the observed state of KogitoServerlessWorkflow
type KogitoServerlessWorkflowStatus struct {
	Conditions StatusCondition    `json:"conditions,omitempty"`
	Endpoints  []Endpoint         `json:"endpoints"`
	Address    duckv1.Addressable `json:"address,omitempty"`
}

// KogitoServerlessWorkflow is the Schema for the kogitoserverlessworkflows API
//+kubebuilder:object:root=true
//+kubebuilder:subresource:items
type KogitoServerlessWorkflow struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KogitoServerlessWorkflowSpec   `json:"spec,omitempty"`
	Status KogitoServerlessWorkflowStatus `json:"status,omitempty"`
}

// KogitoServerlessWorkflowList contains a list of KogitoServerlessWorkflow
//+kubebuilder:object:root=true
type KogitoServerlessWorkflowList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KogitoServerlessWorkflow `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KogitoServerlessWorkflow{}, &KogitoServerlessWorkflowList{})
}
