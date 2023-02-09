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
	"strings"
	"time"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	operatorapi "github.com/kiegroup/kogito-serverless-operator/api/v1alpha08"
	"github.com/kiegroup/kogito-serverless-operator/utils"
	kubeutil "github.com/kiegroup/kogito-serverless-operator/utils/kubernetes"
)

var _ ProfileReconciler = &developmentProfile{}

const (
	// TODO: read from the platform config, open a JIRA to track it down. Default tag MUST align with the current operator's version
	defaultKogitoServerlessWorkflowDevImage = "quay.io/kiegroup/kogito-swf-builder-nightly:latest"
	configMapWorkflowDefVolumeNamePrefix    = "wd"
	configMapWorkflowDefMountPath           = "/home/kogito/serverless-workflow-project/src/main/resources/workflows"
	requeueAfterFailure                     = 3 * time.Minute
	requeueAfterFollowDeployment            = 10 * time.Second
	requeueAfterIsRunning                   = 3 * time.Minute
)

type developmentProfile struct {
	baseReconciler
}

func (d developmentProfile) GetProfile() Profile {
	return Development
}

func newDevProfileReconciler(client client.Client, logger *logr.Logger, workflow *operatorapi.KogitoServerlessWorkflow) ProfileReconciler {
	support := &stateSupport{
		logger: logger,
		client: client,
	}
	ensurers := newDevelopmentObjectEnsurers(support)

	handler := newReconciliationStateMachine(logger,
		&ensureRunningDevWorkflowReconcilerState{stateSupport: support, ensurers: ensurers},
		&followDeployDevWorkflowReconcilerState{stateSupport: support})
	// TODO: recover from failure state (try catch)

	profile := &developmentProfile{
		baseReconciler: newBaseProfileReconciler(support, handler, workflow),
	}
	return profile
}

func newDevelopmentObjectEnsurers(support *stateSupport) *developmentObjectEnsurers {
	return &developmentObjectEnsurers{
		deployment:        newObjectEnsurer(support.client, support.logger, defaultDeploymentCreator),
		service:           newObjectEnsurer(support.client, support.logger, defaultServiceCreator),
		workflowConfigMap: newObjectEnsurer(support.client, support.logger, workflowSpecConfigMapCreator),
	}
}

type developmentObjectEnsurers struct {
	deployment        *objectEnsurer
	service           *objectEnsurer
	workflowConfigMap *objectEnsurer
}

type ensureRunningDevWorkflowReconcilerState struct {
	*stateSupport
	ensurers *developmentObjectEnsurers
}

func (e *ensureRunningDevWorkflowReconcilerState) CanReconcile(workflow *operatorapi.KogitoServerlessWorkflow) bool {
	return workflow.Status.Condition == operatorapi.NoneConditionType ||
		workflow.Status.Condition == operatorapi.RunningConditionType
}

func (e *ensureRunningDevWorkflowReconcilerState) Do(ctx context.Context, workflow *operatorapi.KogitoServerlessWorkflow) (ctrl.Result, []client.Object, error) {
	var objs []client.Object

	configMap, cmResult, err := e.ensurers.workflowConfigMap.ensure(ctx, workflow, ensureWorkflowSpecConfigMapMutator(workflow))
	if err != nil {
		return ctrl.Result{Requeue: false}, objs, err
	}
	objs = append(objs, configMap)

	deployment, _, err := e.ensurers.deployment.ensure(ctx, workflow,
		defaultDeploymentMutateVisitor(workflow),
		naiveApplyImageDeploymentMutateVisitor(defaultKogitoServerlessWorkflowDevImage),
		mountWorkflowDefConfigMapMutateVisitor(configMap.(*v1.ConfigMap)),
		rolloutDeploymentIfCMChangedMutateVisitor(cmResult))
	if err != nil {
		return ctrl.Result{RequeueAfter: requeueAfterFailure}, objs, err
	}
	objs = append(objs, deployment)

	service, _, err := e.ensurers.service.ensure(ctx, workflow, defaultServiceMutateVisitor(workflow))
	if err != nil {
		return ctrl.Result{RequeueAfter: requeueAfterFailure}, objs, err
	}
	objs = append(objs, service)

	// TODO: these should be done by the "onExit" handler in the state machine given that each state would have to transition

	if workflow.Status.Condition == operatorapi.NoneConditionType {
		workflow.Status.Condition = operatorapi.DeployingConditionType
		if _, err = e.performStatusUpdate(ctx, workflow); err != nil {
			return ctrl.Result{RequeueAfter: requeueAfterFailure}, objs, err
		}
		return ctrl.Result{RequeueAfter: requeueAfterFollowDeployment}, objs, nil
	}
	// This conditional is not necessary since the reconciliation state guarantees that, but it's here to make the code easier to understand.
	if workflow.Status.Condition == operatorapi.RunningConditionType {
		// let's make sure that the deployment is still running
		reflectDeployment := deployment.(*appsv1.Deployment)
		if !kubeutil.IsDeploymentAvailable(reflectDeployment) {
			workflow.Status.Reason = getDeploymentFailureReasonOrDefaultReason(reflectDeployment)
			workflow.Status.Condition = operatorapi.FailedConditionType
			if _, err = e.performStatusUpdate(ctx, workflow); err != nil {
				return ctrl.Result{RequeueAfter: requeueAfterFailure}, objs, err
			}
		}
	}

	return ctrl.Result{RequeueAfter: requeueAfterIsRunning}, objs, nil
}

type followDeployDevWorkflowReconcilerState struct {
	*stateSupport
}

func (f *followDeployDevWorkflowReconcilerState) CanReconcile(workflow *operatorapi.KogitoServerlessWorkflow) bool {
	return workflow.Status.Condition == operatorapi.DeployingConditionType
}

func (f *followDeployDevWorkflowReconcilerState) Do(ctx context.Context, workflow *operatorapi.KogitoServerlessWorkflow) (ctrl.Result, []client.Object, error) {
	deployment := &appsv1.Deployment{}
	if err := f.client.Get(ctx, client.ObjectKeyFromObject(workflow), deployment); err != nil {
		// we should have the deployment by this time, so even if the error above is not found, we should halt.
		return ctrl.Result{RequeueAfter: requeueAfterFailure}, nil, err
	}

	if kubeutil.IsDeploymentAvailable(deployment) {
		if workflow.Status.Condition != operatorapi.RunningConditionType {
			workflow.Status.Condition = operatorapi.RunningConditionType
			if _, err := f.performStatusUpdate(ctx, workflow); err != nil {
				return ctrl.Result{RequeueAfter: requeueAfterFailure}, nil, err
			}
		}
		return ctrl.Result{}, nil, nil
	}

	if kubeutil.IsDeploymentProgressing(deployment) {
		if workflow.Status.Condition != operatorapi.DeployingConditionType {
			workflow.Status.Condition = operatorapi.DeployingConditionType
			if _, err := f.performStatusUpdate(ctx, workflow); err != nil {
				return ctrl.Result{RequeueAfter: requeueAfterFailure}, nil, err
			}
		}
		return ctrl.Result{RequeueAfter: requeueAfterFollowDeployment}, nil, nil
	}

	workflow.Status.Condition = operatorapi.FailedConditionType
	workflow.Status.Reason = getDeploymentFailureReasonOrDefaultReason(deployment)
	_, err := f.performStatusUpdate(ctx, workflow)
	return ctrl.Result{RequeueAfter: requeueAfterFailure}, nil, err
}

// getDeploymentFailureReasonOrDefaultReason gets the replica failure reason.
// MUST be called after checking that the Deployment is NOT available.
// If it's no reason, the Deployment state has no apparent reason to be in failed state.
func getDeploymentFailureReasonOrDefaultReason(deployment *appsv1.Deployment) string {
	failure := kubeutil.GetDeploymentUnavailabilityReason(deployment)
	if len(failure) == 0 {
		failure = fmt.Sprintf("Workflow Deployment %s is unavailable for unknown reasons", deployment.Name)
	}
	return failure
}

// mountWorkflowDefConfigMapMutateVisitor mounts the given ConfigMap workflows definitions into the dev container
func mountWorkflowDefConfigMapMutateVisitor(cm *v1.ConfigMap) mutateVisitor {
	return func(object client.Object) controllerutil.MutateFn {
		return func() error {
			deployment := object.(*appsv1.Deployment)
			volumes := make([]v1.Volume, 0)
			volumeMounts := make([]v1.VolumeMount, 0)
			permission := kubeutil.ConfigMapAllPermissions

			for file := range cm.Data {
				if !strings.HasSuffix(file, kogitoWorkflowJSONFileExt) {
					break
				}
				volumeName := configMapWorkflowDefVolumeNamePrefix + "-" + utils.RemoveKnownExtension(file, kogitoWorkflowJSONFileExt)
				volumes = append(volumes, v1.Volume{
					Name: volumeName,
					VolumeSource: v1.VolumeSource{
						ConfigMap: &v1.ConfigMapVolumeSource{
							LocalObjectReference: v1.LocalObjectReference{Name: cm.Name},
							Items: []v1.KeyToPath{{
								Key:  file,
								Path: file,
							}},
							DefaultMode: &permission,
						},
					},
				})
				volumeMounts = append(volumeMounts,
					v1.VolumeMount{
						Name:      volumeName,
						ReadOnly:  false,
						MountPath: configMapWorkflowDefMountPath + "/" + file,
						SubPath:   file,
					},
				)
			}

			deployment.Spec.Template.Spec.Volumes = make([]v1.Volume, 0)
			deployment.Spec.Template.Spec.Volumes = volumes
			deployment.Spec.Template.Spec.Containers[0].VolumeMounts = make([]v1.VolumeMount, 0)
			deployment.Spec.Template.Spec.Containers[0].VolumeMounts = volumeMounts

			return nil
		}
	}
}

func rolloutDeploymentIfCMChangedMutateVisitor(cmOperationResult controllerutil.OperationResult) mutateVisitor {
	return func(object client.Object) controllerutil.MutateFn {
		return func() error {
			if cmOperationResult == controllerutil.OperationResultUpdated {
				deployment := object.(*appsv1.Deployment)
				//logger.Info("Workflow definition changed, deployment must restart to reflect changes", "deployment", deployment.Name, "namespace", deployment.Namespace)
				err := kubeutil.MarkDeploymentToRollout(deployment)
				return err
			}
			return nil
		}
	}
}
