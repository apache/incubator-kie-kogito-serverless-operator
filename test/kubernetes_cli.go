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

package test

import (
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/kiegroup/kogito-serverless-operator/api/v1alpha08"
)

// NewKogitoClientBuilder creates a new fake.ClientBuilder with the right scheme references
func NewKogitoClientBuilder() *fake.ClientBuilder {
	s := scheme.Scheme
	utilruntime.Must(v1alpha08.AddToScheme(s))
	return fake.NewClientBuilder().WithScheme(s)
}
