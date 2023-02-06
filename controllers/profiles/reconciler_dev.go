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
	"fmt"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	operatorapi "github.com/kiegroup/kogito-serverless-operator/api/v1alpha08"
	kubeutil "github.com/kiegroup/kogito-serverless-operator/utils/kubernetes"
)

var _ ProfileReconciler = &developmentProfile{}

const (
	// TODO: read from the platform config, open a JIRA to track it down. Default tag MUST align with the current operator's version
	defaultKogitoServerlessWorkflowDevImage = "quay.io/kiegroup/kogito-swf-builder-nightly:latest"
	configMapWorkflowDefVolumeName          = "workflow_definition"
	configMapWorkflowDefMountPath           = "/home/kogito/serverless-workflow-project/src/main/resources/workflows"
)

type developmentProfile struct {
	baseReconciler
}

type developmentObjectEnsurers struct {
	deployment        *objectEnsurer
	service           *objectEnsurer
	workflowConfigMap *objectEnsurer
}

func newDevelopmentObjectEnsurers(support *stateSupport) *developmentObjectEnsurers {
	return &developmentObjectEnsurers{
		deployment:        newObjectEnsurer(support.client, support.logger, defaultDeploymentCreator, defaultDeploymentMutator),
		service:           newObjectEnsurer(support.client, support.logger, defaultServiceCreator, immutableObject(defaultDeploymentCreator)),
		workflowConfigMap: newObjectEnsurer(support.client, support.logger, workflowSpecConfigMapCreator, immutableObject(workflowSpecConfigMapCreator)),
	}
}

func newDevProfile(client client.Client, logger logr.Logger, workflow *operatorapi.KogitoServerlessWorkflow) ProfileReconciler {
	support := &stateSupport{
		logger: logger,
		client: client,
	}
	ensurers := newDevelopmentObjectEnsurers(support)

	handler := newReconciliationStateMachine(logger,
		&ensureDeployDevWorkflowReconcilerState{stateSupport: support, ensurers: ensurers},
		&verifyDeployDevWorkflowReconcilerState{stateSupport: support})
	// TODO: recover from failure state (try catch)

	profile := &developmentProfile{
		baseReconciler: newBaseProfileReconciler(support, handler, workflow),
	}
	return profile
}

func (d developmentProfile) GetProfile() Profile {
	return Development
}

type ensureDeployDevWorkflowReconcilerState struct {
	*stateSupport
	ensurers *developmentObjectEnsurers
}

func (d ensureDeployDevWorkflowReconcilerState) CanReconcile(workflow *operatorapi.KogitoServerlessWorkflow) bool {
	return workflow.Status.Condition == operatorapi.NoneConditionType ||
		workflow.Status.Condition == operatorapi.RunningConditionType
}

func (d ensureDeployDevWorkflowReconcilerState) Do(ctx context.Context, workflow *operatorapi.KogitoServerlessWorkflow) (ctrl.Result, []client.Object, error) {
	var objs []client.Object

	configMap, _, err := d.ensurers.workflowConfigMap.ensure(ctx, workflow)
	if err != nil {
		return ctrl.Result{Requeue: false}, objs, err
	}
	objs = append(objs, configMap)

	deployment, _, err := d.ensurers.deployment.ensure(ctx, workflow,
		naiveApplyImageDeploymentMutateVisitor(defaultKogitoServerlessWorkflowDevImage),
		mountWorkflowDefConfigMapMutateVisitor(configMap.(*v1.ConfigMap)))
	if err != nil {
		return ctrl.Result{Requeue: false}, objs, err
	}
	objs = append(objs, deployment)

	service, _, err := d.ensurers.service.ensure(ctx, workflow)
	if err != nil {
		return ctrl.Result{Requeue: false}, objs, err
	}
	objs = append(objs, service)

	// TODO: this should be done by the "onExit" handler in the state machine given that each state would have to transition
	if workflow.Status.Condition == operatorapi.NoneConditionType {
		workflow.Status.Condition = operatorapi.DeployingConditionType
		if _, err = d.performStatusUpdate(ctx, workflow); err != nil {
			return ctrl.Result{Requeue: false}, objs, err
		}
		return ctrl.Result{Requeue: false}, objs, nil
	}
	// let's make sure that the deployment is still running
	reflectDeployment := deployment.(*appsv1.Deployment)
	if !kubeutil.IsDeploymentAvailable(reflectDeployment) {
		workflow.Status.Reason = getDeploymentFailureReasonOrDefaultReason(reflectDeployment)
		workflow.Status.Condition = operatorapi.FailedConditionType
		if _, err = d.performStatusUpdate(ctx, workflow); err != nil {
			return ctrl.Result{Requeue: false}, objs, err
		}
	}

	return ctrl.Result{Requeue: false}, objs, nil
}

type verifyDeployDevWorkflowReconcilerState struct {
	*stateSupport
}

func (v verifyDeployDevWorkflowReconcilerState) CanReconcile(workflow *operatorapi.KogitoServerlessWorkflow) bool {
	return workflow.Status.Condition == operatorapi.DeployingConditionType
}

func (v verifyDeployDevWorkflowReconcilerState) Do(ctx context.Context, workflow *operatorapi.KogitoServerlessWorkflow) (ctrl.Result, []client.Object, error) {
	deployment := &appsv1.Deployment{}
	if err := v.client.Get(ctx, client.ObjectKeyFromObject(workflow), deployment); err != nil {
		// we should have the deployment by this time, so even if the error above is not found, we should halt.
		return ctrl.Result{Requeue: false}, nil, err
	}

	if kubeutil.IsDeploymentAvailable(deployment) {
		var err error
		if workflow.Status.Condition != operatorapi.RunningConditionType {
			workflow.Status.Condition = operatorapi.RunningConditionType
			_, err = v.performStatusUpdate(ctx, workflow)
		}
		return ctrl.Result{Requeue: false}, nil, err
	}

	if kubeutil.IsDeploymentProgressing(deployment) {
		var err error
		if workflow.Status.Condition != operatorapi.DeployingConditionType {
			workflow.Status.Condition = operatorapi.DeployingConditionType
			_, err = v.performStatusUpdate(ctx, workflow)
		}
		return ctrl.Result{Requeue: false}, nil, err
	}

	workflow.Status.Condition = operatorapi.FailedConditionType
	workflow.Status.Reason = getDeploymentFailureReasonOrDefaultReason(deployment)
	if _, err := v.performStatusUpdate(ctx, workflow); err != nil {
		return ctrl.Result{Requeue: false}, nil, err
	}

	return ctrl.Result{Requeue: false}, nil, nil

}

// getDeploymentFailureReasonOrDefaultReason for some reason the deployment is not available, but the replica failure state is false, so no apparent reason.
func getDeploymentFailureReasonOrDefaultReason(deployment *appsv1.Deployment) string {
	failure := kubeutil.GetDeploymentUnavailabilityReason(deployment)
	if len(failure) == 0 {
		failure = fmt.Sprintf("Workflow Deployment %s is unavailble for unknown reasons", deployment.Name)
	}
	return failure
}

// mountWorkflowDefConfigMapMutateVisitor mounts the given workflows definitions in the ConfigMap into the dev container
func mountWorkflowDefConfigMapMutateVisitor(cm *v1.ConfigMap) mutateVisitor {
	return func(object client.Object) controllerutil.MutateFn {
		return func() error {
			deployment := object.(*appsv1.Deployment)
			deployment.Spec.Template.Spec.Volumes = append(deployment.Spec.Template.Spec.Volumes, v1.Volume{
				Name: configMapWorkflowDefVolumeName,
				VolumeSource: v1.VolumeSource{
					ConfigMap: &v1.ConfigMapVolumeSource{LocalObjectReference: v1.LocalObjectReference{Name: cm.Name}},
				},
			})
			deployment.Spec.Template.Spec.Containers[0].VolumeMounts = append(deployment.Spec.Template.Spec.Containers[0].VolumeMounts,
				v1.VolumeMount{
					Name:      configMapWorkflowDefVolumeName,
					ReadOnly:  false,
					MountPath: configMapWorkflowDefMountPath,
				},
			)
			return nil
		}
	}
}
