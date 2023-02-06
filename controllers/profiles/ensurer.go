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
	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	operatorapi "github.com/kiegroup/kogito-serverless-operator/api/v1alpha08"
)

// newObjectEnsurer see objectEnsurer
func newObjectEnsurer(client client.Client, logger logr.Logger, creator objectCreator, mutator objectEnforcer) *objectEnsurer {
	return &objectEnsurer{
		client:   client,
		logger:   logger,
		creator:  creator,
		enforcer: mutator,
	}
}

// objectEnsurer provides the engine for a ReconciliationState that needs to create or update a given Kubernetes object during the reconciliation cycle.
type objectEnsurer struct {
	client client.Client
	logger logr.Logger
	// creator is the func that creates the initial reference object, if the object doesn't exist in the cluster, this one is created.
	// Can be used as a reference to keep the object immutable
	creator objectCreator
	// enforcer is called before updating the object, so any attributes in the given object that MUST be guaranteed by the controller should be updated here.
	// If the object is immutable, use the immutableObject func reference.
	enforcer objectEnforcer
}

// mutateVisitor is a visitor function that mutates the given object before performing any updates in the cluster.
// It gets called after the objectEnforcer reference.
//
// The objectEnsurer will call the returned function after creating the given object structure, so callers is ensured to have the default reference of the given object.
//
// Usually you can safely do `object.(*<kubernetesType>).Spec...` since you control the objectCreator.
//
// Example: `object.(*appsv1.Deployment).Spec.Template.Name="myApp"` to change the pod's name.
type mutateVisitor func(object client.Object) controllerutil.MutateFn

// naiveApplyImageDeploymentMutateVisitor creates a visitor that mutates a vanilla Kubernetes Deployment to apply the given image in the first container
func naiveApplyImageDeploymentMutateVisitor(image string) mutateVisitor {
	return func(object client.Object) controllerutil.MutateFn {
		return func() error {
			object.(*appsv1.Deployment).Spec.Template.Spec.Containers[0].Image = image
			return nil
		}
	}
}

func (d objectEnsurer) ensure(ctx context.Context, workflow *operatorapi.KogitoServerlessWorkflow, visitors ...mutateVisitor) (client.Object, controllerutil.OperationResult, error) {
	result := controllerutil.OperationResultNone
	object, err := d.creator(workflow)
	if err != nil {
		return nil, result, err
	}
	if err = controllerutil.SetControllerReference(workflow, object, d.client.Scheme()); err != nil {
		return nil, result, err
	}
	if result, err = controllerutil.CreateOrPatch(ctx, d.client, object,
		func() error {
			if d.enforcer != nil {
				if mutateErr := d.enforcer(workflow, object); mutateErr != nil {
					return mutateErr
				}
			}
			for _, v := range visitors {
				if visitorErr := v(object)(); visitorErr != nil {
					return visitorErr
				}
			}
			return nil
		}); err != nil {
		return nil, result, err
	}
	d.logger.Info("Object operation finalized", "result", result, "kind", object.GetObjectKind().GroupVersionKind().String(), "name", object.GetName(), "namespace", object.GetNamespace())
	return object, result, nil
}
