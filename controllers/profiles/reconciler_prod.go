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
	"time"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/kiegroup/container-builder/api"
	operatorapi "github.com/kiegroup/kogito-serverless-operator/api/v1alpha08"
	"github.com/kiegroup/kogito-serverless-operator/builder"
	"github.com/kiegroup/kogito-serverless-operator/platform"
	"github.com/kiegroup/kogito-serverless-operator/utils"
)

var _ ProfileReconciler = &productionProfile{}

type productionProfile struct {
	baseReconciler
}

func newProductionProfile(client client.Client, logger logr.Logger, scheme *runtime.Scheme, workflow *operatorapi.KogitoServerlessWorkflow) ProfileReconciler {
	ctx := &reconcilerSupport{
		logger: logger,
		client: client,
	}
	// the reconciliation state machine
	handler := newReconciliationHandlersDelegate(
		logger,
		&newBuilderReconciliationHandler{reconcilerSupport: ctx},
		&ensureBuilderReconciliationHandler{reconcilerSupport: ctx},
		&followBuildStatusReconciliationHandler{reconcilerSupport: ctx},
		&deployWorkflowReconciliationHandler{reconcilerSupport: ctx},
	)
	reconciler := &productionProfile{
		baseReconciler{
			workflow:          workflow,
			scheme:            scheme,
			reconcilerSupport: ctx,
			reconcilerHandler: handler,
		},
	}

	return reconciler
}

func (p productionProfile) GetProfile() Profile {
	return Production
}

type newBuilderReconciliationHandler struct {
	*reconcilerSupport
}

func (h *newBuilderReconciliationHandler) CanReconcile(workflow *operatorapi.KogitoServerlessWorkflow) bool {
	return workflow.Status.Condition == operatorapi.NoneConditionType ||
		workflow.Status.Condition == operatorapi.WaitingForPlatformConditionType
}

func (h *newBuilderReconciliationHandler) Do(ctx context.Context, workflow *operatorapi.KogitoServerlessWorkflow) (ctrl.Result, error) {
	buildable := builder.NewBuildable(h.client, ctx)
	_, err := platform.GetActivePlatform(ctx, h.client, workflow.Namespace)
	if err != nil {
		h.logger.Error(err, "No active Platform for namespace %s so the workflow cannot be built. Waiting for an active platform")
		workflow.Status.Condition = operatorapi.WaitingForPlatformConditionType
		_, err = h.performStatusUpdate(ctx, workflow)
		return ctrl.Result{RequeueAfter: 5 * time.Second}, err
	}
	// If there is an active platform we have got all the information to build but...
	// ...let's check before if we have got already a build!
	build, err := buildable.GetWorkflowBuild(workflow.Name, workflow.Namespace)
	if err != nil {
		return ctrl.Result{}, err
	}
	if build == nil {
		//If there isn't a build let's create and start the first one!
		build, err = buildable.CreateWorkflowBuild(workflow.Name, workflow.Namespace)
		if err != nil {
			return ctrl.Result{}, err
		}
	}
	//If there is a build, let's ask to restart it
	build.Status.BuildPhase = api.BuildPhaseNone
	build.Status.Builder.Status = api.BuildStatus{}
	if err = h.client.Status().Update(ctx, build); err != nil {
		h.logger.Error(err, fmt.Sprintf("Failed to update Build status for Workflow %s", workflow.Name))
		return ctrl.Result{}, err
	}
	workflow.Status.Condition = operatorapi.BuildingConditionType
	_, err = h.performStatusUpdate(ctx, workflow)
	return ctrl.Result{}, err
}

type ensureBuilderReconciliationHandler struct {
	*reconcilerSupport
}

func (h *ensureBuilderReconciliationHandler) CanReconcile(workflow *operatorapi.KogitoServerlessWorkflow) bool {
	return workflow.Status.Condition == operatorapi.RunningConditionType
}

func (h *ensureBuilderReconciliationHandler) Do(ctx context.Context, workflow *operatorapi.KogitoServerlessWorkflow) (ctrl.Result, error) {
	buildable := builder.NewBuildable(h.client, ctx)
	build, err := buildable.GetWorkflowBuild(workflow.Name, workflow.Namespace)
	if build != nil &&
		(build.Status.Builder.Status.Phase == api.BuildPhaseSucceeded ||
			build.Status.Builder.Status.Phase == api.BuildPhaseFailed ||
			build.Status.Builder.Status.Phase == api.BuildPhaseError) {
		//If we have finished a build and the workflow is running, we have to rebuild it because there was a change in the workflow definition and requeue the request
		if !utils.Compare(utils.GetWorkflowSpecHash(workflow.Status.Applied), utils.GetWorkflowSpecHash(workflow.Spec)) { // Let's check that the 2 workflow definition are different
			workflow.Status.Condition = operatorapi.NoneConditionType
			_, err = h.performStatusUpdate(ctx, workflow)
			return ctrl.Result{Requeue: true}, err
		}
	}
	return ctrl.Result{}, nil
}

type followBuildStatusReconciliationHandler struct {
	*reconcilerSupport
}

func (h *followBuildStatusReconciliationHandler) CanReconcile(workflow *operatorapi.KogitoServerlessWorkflow) bool {
	return workflow.Status.Condition == operatorapi.BuildingConditionType
}

func (h *followBuildStatusReconciliationHandler) Do(ctx context.Context, workflow *operatorapi.KogitoServerlessWorkflow) (ctrl.Result, error) {
	// Let's retrieve the build to check the status
	build := &operatorapi.KogitoServerlessBuild{}
	err := h.client.Get(ctx, types.NamespacedName{Namespace: workflow.Namespace, Name: workflow.Name}, build)
	if err != nil {
		if !errors.IsNotFound(err) {
			return ctrl.Result{}, err
		}
		h.logger.Error(err, "Build not found for this workflow", "Workflow", workflow.Name)
		return ctrl.Result{}, nil
	}

	if build.Status.Builder.Status.Phase == api.BuildPhaseSucceeded {
		//If we have finished a build and the workflow is not running, we will start the provisioning phase
		workflow.Status.Condition = operatorapi.ProvisioningConditionType
		_, err = h.performStatusUpdate(ctx, workflow)
	} else if build.Status.Builder.Status.Phase == api.BuildPhaseFailed || build.Status.Builder.Status.Phase == api.BuildPhaseError {
		h.logger.Info(fmt.Sprintf("Workflow %s build is failed!", workflow.Name))
		workflow.Status.Condition = operatorapi.FailedConditionType
		_, err = h.performStatusUpdate(ctx, workflow)
	} else {
		if workflow.Status.Condition != operatorapi.BuildingConditionType {
			workflow.Status.Condition = operatorapi.BuildingConditionType
			_, err = h.performStatusUpdate(ctx, workflow)
		}
	}
	return ctrl.Result{}, err
}

type deployWorkflowReconciliationHandler struct {
	*reconcilerSupport
	ensurers []ObjectEnsurer
}

func (h *deployWorkflowReconciliationHandler) CanReconcile(workflow *operatorapi.KogitoServerlessWorkflow) bool {
	return workflow.Status.Condition == operatorapi.ProvisioningConditionType ||
		workflow.Status.Condition == operatorapi.DeployingConditionType
}

func (h *deployWorkflowReconciliationHandler) Do(ctx context.Context, workflow *operatorapi.KogitoServerlessWorkflow) (ctrl.Result, error) {
	pl, err := platform.GetActivePlatform(ctx, h.client, workflow.Namespace)
	if err != nil {
		h.logger.Error(err, "No active Platform for namespace %s so the workflow cannot be deployed. Waiting for an active platform")
		return ctrl.Result{RequeueAfter: 5 * time.Second}, err
	}
	image := pl.Spec.BuildPlatform.Registry.Address + "/" + workflow.Name + utils.GetWorkflowImageTag(workflow)
	return h.handleDeployment(ctx, workflow, image)
}

func (h *deployWorkflowReconciliationHandler) handleDeployment(ctx context.Context, workflow *operatorapi.KogitoServerlessWorkflow, image string) (reconcile.Result, error) {
	// Check if this Deployment already exists
	found := &appsv1.Deployment{}
	err := h.client.Get(ctx, types.NamespacedName{Name: workflow.Name, Namespace: workflow.Namespace}, found)
	var result *reconcile.Result
	// TODO: h.ensurers.deployment.Ensure(ctx)
	deploymentHandler := &defaultDeploymentHandler{
		workflow: workflow,
		scheme:   h.client.Scheme(),
		client:   h.client,
		logger:   h.logger,
	}
	result, err = deploymentHandler.ensureDeploymentObject(ctx, image)
	if result != nil {
		h.logger.Error(err, "Deployment Not ready")
		if workflow.Status.Condition != operatorapi.DeployingConditionType {
			workflow.Status.Condition = operatorapi.DeployingConditionType
			if _, err := h.performStatusUpdate(ctx, workflow); err != nil {
				return *result, err
			}
		}
		result.RequeueAfter = 5 * time.Second
		return *result, err
	}

	// Check if this Service already exists
	// TODO: h.ensurers.service.Ensure(ctx)
	serviceHandler := &defaultServiceHandler{
		workflow: workflow,
		scheme:   h.client.Scheme(),
		client:   h.client,
	}
	result, err = serviceHandler.ensureServiceObject()
	if result != nil {
		h.logger.Error(err, "Service Not ready")
		if workflow.Status.Condition != operatorapi.DeployingConditionType {
			workflow.Status.Condition = operatorapi.DeployingConditionType
			if _, err := h.performStatusUpdate(ctx, workflow); err != nil {
				return *result, err
			}
		}
		result.RequeueAfter = 5 * time.Second
		return *result, err
	}

	// Deployment and Service already exists - don't requeue
	h.logger.Info("Skip reconcile: Deployment and service already exists",
		"Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)

	//We can now update the workflow status to running
	if workflow.Status.Condition != operatorapi.RunningConditionType {
		workflow.Status.Condition = operatorapi.RunningConditionType
		if _, err := h.performStatusUpdate(ctx, workflow); err != nil {
			return *result, err
		}
		return reconcile.Result{Requeue: false}, err
	}

	return reconcile.Result{Requeue: false}, err
}
