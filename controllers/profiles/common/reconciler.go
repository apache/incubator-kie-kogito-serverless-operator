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

package common

import (
	"context"
	"fmt"

	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	operatorapi "github.com/kiegroup/kogito-serverless-operator/api/v1alpha08"
	"github.com/kiegroup/kogito-serverless-operator/controllers/profiles"
	"github.com/kiegroup/kogito-serverless-operator/log"
)

// StateSupport is the shared structure with common accessors used throughout the whole reconciliation profiles
type StateSupport struct {
	C client.Client
}

// PerformStatusUpdate updates the SonataFlow Status conditions
func (s StateSupport) PerformStatusUpdate(ctx context.Context, workflow *operatorapi.SonataFlow) (bool, error) {
	var err error
	workflow.Status.ObservedGeneration = workflow.Generation
	if err = s.C.Status().Update(ctx, workflow); err != nil {
		klog.V(log.E).ErrorS(err, "Failed to update Workflow status")
		return false, err
	}
	return true, err
}

// PostReconcile function to perform all the other operations required after the reconciliation - placeholder for null pattern usages
func (s StateSupport) PostReconcile(ctx context.Context, workflow *operatorapi.SonataFlow) error {
	//By default, we don't want to perform anything after the reconciliation, and so we will simply return no error
	return nil
}

// BaseReconciler is the base structure used by every reconciliation profile.
// Use NewBaseProfileReconciler to build a new reference.
type BaseReconciler struct {
	*StateSupport
	reconciliationStateMachine *ReconciliationStateMachine
	objects                    []client.Object
}

func NewBaseProfileReconciler(support *StateSupport, stateMachine *ReconciliationStateMachine) BaseReconciler {
	return BaseReconciler{
		StateSupport:               support,
		reconciliationStateMachine: stateMachine,
	}
}

// Reconcile does the actual reconciliation algorithm based on a set of ReconciliationState
func (b BaseReconciler) Reconcile(ctx context.Context, workflow *operatorapi.SonataFlow) (ctrl.Result, error) {
	workflow.Status.Manager().InitializeConditions()
	result, objects, err := b.reconciliationStateMachine.do(ctx, workflow)
	if err != nil {
		return result, err
	}
	b.objects = objects
	klog.V(log.I).InfoS("Returning from reconciliation", "Result", result)

	return result, err
}

// NewReconciliationStateMachine builder for the ReconciliationStateMachine
func NewReconciliationStateMachine(states ...profiles.ReconciliationState) *ReconciliationStateMachine {
	return &ReconciliationStateMachine{
		states: states,
	}
}

// ReconciliationStateMachine implements (sort of) the command pattern and delegate to a chain of ReconciliationState
// the actual task to reconcile in a given workflow condition
//
// TODO: implement state transition, so based on a given condition we do the status update which actively transition the object state
type ReconciliationStateMachine struct {
	states []profiles.ReconciliationState
}

func (r *ReconciliationStateMachine) do(ctx context.Context, workflow *operatorapi.SonataFlow) (ctrl.Result, []client.Object, error) {
	for _, h := range r.states {
		if h.CanReconcile(workflow) {
			klog.V(log.I).InfoS("Found a condition to reconcile.", "Conditions", workflow.Status.Conditions)
			result, objs, err := h.Do(ctx, workflow)
			if err != nil {
				return result, objs, err
			}
			if err = h.PostReconcile(ctx, workflow); err != nil {
				klog.V(log.E).ErrorS(err, "Error in Post Reconcile actions.", "Workflow", workflow.Name, "Conditions", workflow.Status.Conditions)
			}
			return result, objs, err
		}
	}
	return ctrl.Result{}, nil, fmt.Errorf("the workflow %s in the namespace %s is in an unknown state condition. Can't reconcilie. Status is: %v", workflow.Name, workflow.Namespace, workflow.Status)
}
