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

// Alias for api/v1alpha08 to enable client-gen
// We had to add the `sonataflow` directory as a group name, so the client code could be correctly generated.
// Every time we have a new type that needs to have a client generated, just add the alias in the types.go file and run
// `make generate`

// +k8s:deepcopy-gen=package
// +groupName=sonataflow.org

package v1alpha08
