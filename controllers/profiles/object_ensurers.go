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

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	operatorapi "github.com/kiegroup/kogito-serverless-operator/api/v1alpha08"
)

func newObjectEnsurer(scheme *runtime.Scheme, client client.Client, logger logr.Logger, creator objectCreator) *objectEnsurer {
	return &objectEnsurer{
		scheme:  scheme,
		client:  client,
		logger:  logger,
		creator: creator,
	}
}

// objectEnsurer provides the engine for a ReconciliationHandler that needs to create or update a given Kubernetes object during the reconciliation cycle.
type objectEnsurer struct {
	scheme  *runtime.Scheme
	client  client.Client
	logger  logr.Logger
	creator objectCreator
}

var emptyMutateHandler = func(object client.Object) controllerutil.MutateFn {
	return func() error {
		return nil
	}
}

// mutateHandler creates the mutate function for the controller utils.
// Callers can mutate the object before an update or create k8s API call via the just created resource by the objectEnsurer.
type mutateHandler func(object client.Object) controllerutil.MutateFn

func (d objectEnsurer) ensure(ctx context.Context, workflow *operatorapi.KogitoServerlessWorkflow, handler mutateHandler) (client.Object, controllerutil.OperationResult, error) {
	object := d.creator(workflow)
	var result controllerutil.OperationResult
	var err error
	if err = controllerutil.SetControllerReference(workflow, object, d.scheme); err != nil {
		return nil, controllerutil.OperationResultNone, err
	}
	if result, err = controllerutil.CreateOrUpdate(ctx, d.client, object, handler(object)); err != nil {
		return nil, result, err
	}
	return object, result, nil
}
