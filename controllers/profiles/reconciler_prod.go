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
	return &productionProfile{
		baseReconciler{
			workflow: workflow,
			scheme:   scheme,
			client:   client,
			logger:   logger,
		},
	}
}

func (p productionProfile) GetProfile() Profile {
	return Production
}

func (p productionProfile) Reconcile(ctx context.Context) (ctrl.Result, error) {
	buildable := builder.NewBuildable(p.client, ctx)

	switch p.workflow.Status.Condition {

	// If the status condition is None or Waiting for platform let's try to start a build!
	case operatorapi.NoneConditionType, operatorapi.WaitingForPlatformConditionType:
		_, err := platform.GetActivePlatform(ctx, p.client, p.workflow.Namespace)
		if err != nil {
			p.logger.Error(err, "No active Platform for namespace %s so the workflow cannot be built. Waiting for an active platform")
			p.workflow.Status.Condition = operatorapi.WaitingForPlatformConditionType
			_, err = p.performStatusUpdate(ctx)
			return ctrl.Result{RequeueAfter: 5 * time.Second}, err
		}
		// If there is an active platform we have got all the information to build but...
		// ...let's check before if we have got already a build!
		build, err := buildable.GetWorkflowBuild(p.workflow.Name, p.workflow.Namespace)
		if err != nil {
			return ctrl.Result{}, err
		}
		if build == nil {
			//If there isn't a build let's create and start the first one!
			build, err = buildable.CreateWorkflowBuild(p.workflow.Name, p.workflow.Namespace)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
		//If there is a build, let's ask to restart it
		build.Status.BuildPhase = api.BuildPhaseNone
		build.Status.Builder.Status = api.BuildStatus{}
		if err = p.client.Status().Update(ctx, build); err != nil {
			p.logger.Error(err, fmt.Sprintf("Failed to update Build status for Workflow %s", p.workflow.Name))
			return ctrl.Result{}, err
		}
		p.workflow.Status.Condition = operatorapi.BuildingConditionType
		_, err = p.performStatusUpdate(ctx)
		return ctrl.Result{}, err

	case operatorapi.RunningConditionType:
		build, err := buildable.GetWorkflowBuild(p.workflow.Name, p.workflow.Namespace)
		if build != nil &&
			(build.Status.Builder.Status.Phase == api.BuildPhaseSucceeded ||
				build.Status.Builder.Status.Phase == api.BuildPhaseFailed ||
				build.Status.Builder.Status.Phase == api.BuildPhaseError) {
			//If we have finished a build and the workflow is running, we have to rebuild it because there was a change in the workflow definition and requeue the request
			if !utils.Compare(utils.GetWorkflowSpecHash(p.workflow.Status.Applied), utils.GetWorkflowSpecHash(p.workflow.Spec)) { // Let's check that the 2 workflow definition are different
				p.workflow.Status.Condition = operatorapi.NoneConditionType
				_, err = p.performStatusUpdate(ctx)
				return ctrl.Result{Requeue: true}, err
			}
		}
	// If the status condition is Building let's check if the build is finished!
	case operatorapi.BuildingConditionType:
		// Let's retrieve the build to check the status
		build := &operatorapi.KogitoServerlessBuild{}
		err := p.client.Get(ctx, types.NamespacedName{Namespace: p.workflow.Namespace, Name: p.workflow.Name}, build)
		if err != nil {
			if !errors.IsNotFound(err) {
				return ctrl.Result{}, err
			}
			p.logger.Error(err, "Build not found for this workflow", "Workflow", p.workflow.Name)
			return ctrl.Result{}, nil
		}

		if build.Status.Builder.Status.Phase == api.BuildPhaseSucceeded {
			//If we have finished a build and the workflow is not running, we will start the provisioning phase
			p.workflow.Status.Condition = operatorapi.ProvisioningConditionType
			_, err = p.performStatusUpdate(ctx)
			return ctrl.Result{}, err
		} else if build.Status.Builder.Status.Phase == api.BuildPhaseFailed || build.Status.Builder.Status.Phase == api.BuildPhaseError {
			p.logger.Info(fmt.Sprintf("Workflow %s build is failed!", p.workflow.Name))
			p.workflow.Status.Condition = operatorapi.FailedConditionType
			_, err = p.performStatusUpdate(ctx)

		} else {
			if p.workflow.Status.Condition != operatorapi.BuildingConditionType {
				p.workflow.Status.Condition = operatorapi.BuildingConditionType
				_, err = p.performStatusUpdate(ctx)
			}
			return ctrl.Result{}, err
		}

	// If the status condition is Deploying let's check if the deployment is finished!
	case operatorapi.ProvisioningConditionType, operatorapi.DeployingConditionType:
		pl, err := platform.GetActivePlatform(ctx, p.client, p.workflow.Namespace)
		if err != nil {
			p.logger.Error(err, "No active Platform for namespace %s so the workflow cannot be deployed. Waiting for an active platform")
			return ctrl.Result{RequeueAfter: 5 * time.Second}, err
		}
		return p.handleWorkflowDeployment(ctx, pl)

	default:
		p.logger.Info(fmt.Sprintf("Workflow %s is in status %s but at the moment we are not supporting it!", p.workflow.Name, p.workflow.Status.Condition))
		return ctrl.Result{}, nil
	}
	return ctrl.Result{}, nil
}

func (p productionProfile) handleWorkflowDeployment(ctx context.Context, platform *operatorapi.KogitoServerlessPlatform) (reconcile.Result, error) {
	// Check if this Deployment already exists
	found := &appsv1.Deployment{}
	err := p.client.Get(ctx, types.NamespacedName{Name: p.workflow.Name, Namespace: p.workflow.Namespace}, found)
	var result *reconcile.Result
	result, err = p.EnsureDeployment(ctx, p.getImageName(platform.Spec.BuildPlatform.Registry.Address))
	if result != nil {
		p.logger.Error(err, "Deployment Not ready")
		if p.workflow.Status.Condition != operatorapi.DeployingConditionType {
			p.workflow.Status.Condition = operatorapi.DeployingConditionType
			if _, err := p.performStatusUpdate(ctx); err != nil {
				return *result, err
			}
		}
		result.RequeueAfter = 5 * time.Second
		return *result, err
	}

	// Check if this Service already exists
	result, err = p.EnsureService()
	if result != nil {
		p.logger.Error(err, "Service Not ready")
		if p.workflow.Status.Condition != operatorapi.DeployingConditionType {
			p.workflow.Status.Condition = operatorapi.DeployingConditionType
			if _, err := p.performStatusUpdate(ctx); err != nil {
				return *result, err
			}
		}
		result.RequeueAfter = 5 * time.Second
		return *result, err
	}

	// Deployment and Service already exists - don't requeue
	p.logger.Info("Skip reconcile: Deployment and service already exists",
		"Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)

	//We can now update the workflow status to running
	if p.workflow.Status.Condition != operatorapi.RunningConditionType {
		p.workflow.Status.Condition = operatorapi.RunningConditionType
		if _, err := p.performStatusUpdate(ctx); err != nil {
			return *result, err
		}
		return reconcile.Result{Requeue: false}, err
	}

	return reconcile.Result{Requeue: false}, err
}

func (p productionProfile) getImageName(registryAddress string) string {
	return registryAddress + "/" + p.workflow.Name + utils.GetWorkflowImageTag(p.workflow)
}
