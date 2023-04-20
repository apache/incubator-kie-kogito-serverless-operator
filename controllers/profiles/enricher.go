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

package profiles

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	operatorapi "github.com/kiegroup/kogito-serverless-operator/api/v1alpha08"
)

// Enricher is useful when you have to apply the content enricher design pattern, adding some info to resources
// See https://www.enterpriseintegrationpatterns.com/patterns/messaging/DataEnricher.html
type Enricher interface {
	Enrich(ctx context.Context, client client.Client, workflow *operatorapi.KogitoServerlessWorkflow) (controllerutil.OperationResult, error)
}
