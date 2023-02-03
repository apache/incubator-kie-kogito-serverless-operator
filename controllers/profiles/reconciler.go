/*
 * Copyright 2023 Red Hat, Inc. and/or its affiliates.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package profiles

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	operatorapi "github.com/kiegroup/kogito-serverless-operator/api/v1alpha08"
)

// ProfileReconciler is the public interface to have access to this package and perform the actual reconciliation
type ProfileReconciler interface {
	Reconcile(ctx context.Context) (ctrl.Result, error)
	GetProfile() Profile
}

// reconcilerSupport is the shared structure with common accessors used throughout the whole reconciliation profiles
type reconcilerSupport struct {
	logger logr.Logger
	client client.Client
}

// performStatusUpdate updates the KogitoServerlessWorkflow Status conditions
func (s reconcilerSupport) performStatusUpdate(ctx context.Context, workflow *operatorapi.KogitoServerlessWorkflow) (bool, error) {
	var err error
	workflow.Status.Applied = workflow.Spec
	if err = s.client.Status().Update(ctx, workflow); err != nil {
		s.logger.Error(err, "Failed to update Workflow status")
		return false, err
	}
	return true, err
}

// baseReconciler is the base structure used by every reconciliation profile
type baseReconciler struct {
	*reconcilerSupport
	workflow          *operatorapi.KogitoServerlessWorkflow
	scheme            *runtime.Scheme
	reconcilerHandler *reconciliationHandlersDelegate
	objects           []client.Object
}

// Reconcile does the actual reconciliation algorithm based on a set of ReconciliationHandler
func (b baseReconciler) Reconcile(ctx context.Context) (ctrl.Result, error) {
	result, objects, err := b.reconcilerHandler.do(ctx, b.workflow)
	if err != nil {
		return result, err
	}
	b.objects = objects
	return result, err
}

// ReconciliationHandler is an interface implemented internally by various elements to perform the adequate logic for a given workflow profile
type ReconciliationHandler interface {
	// CanReconcile checks if this handler can perform its reconciliation task
	CanReconcile(workflow *operatorapi.KogitoServerlessWorkflow) bool
	// Do perform the reconciliation task. It returns the controller result, the objects updated, and an error if any.
	// Objects can be nil if the reconciliation task doesn't perform any updates in any Kubernetes object.
	Do(ctx context.Context, workflow *operatorapi.KogitoServerlessWorkflow) (ctrl.Result, []client.Object, error)
}

// newReconciliationHandlersDelegate builder for the reconciliationHandlersDelegate
func newReconciliationHandlersDelegate(logger logr.Logger, handlers ...ReconciliationHandler) *reconciliationHandlersDelegate {
	return &reconciliationHandlersDelegate{
		handlers: handlers,
		logger:   logger,
	}
}

// reconciliationHandlersDelegate implements (sort of) the command pattern and delegate to a chain of ReconciliationHandler
// the actual task to reconcile in a given workflow condition
type reconciliationHandlersDelegate struct {
	handlers []ReconciliationHandler
	logger   logr.Logger
}

func (r *reconciliationHandlersDelegate) do(ctx context.Context, workflow *operatorapi.KogitoServerlessWorkflow) (ctrl.Result, []client.Object, error) {
	for _, h := range r.handlers {
		if h.CanReconcile(workflow) {
			return h.Do(ctx, workflow)
		}
	}
	r.logger.Info(fmt.Sprintf("Workflow %s is in status %s but at the moment we are not supporting it!", workflow.Name, workflow.Status.Condition))
	return ctrl.Result{}, nil, nil
}

// NewReconciler creates a new ProfileReconciler based on the given workflow and context.
func NewReconciler(client client.Client, logger logr.Logger, scheme *runtime.Scheme, workflow *operatorapi.KogitoServerlessWorkflow) ProfileReconciler {
	return profileBuilder(workflow)(client, logger, scheme, workflow)
}
