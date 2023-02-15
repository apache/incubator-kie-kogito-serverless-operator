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
	"time"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	operatorapi "github.com/kiegroup/kogito-serverless-operator/api/v1alpha08"
	kubeutil "github.com/kiegroup/kogito-serverless-operator/utils/kubernetes"
)

var _ ProfileReconciler = &developmentProfile{}

const (
	// TODO: read from the platform config, open a JIRA to track it down. Default tag MUST align with the current operator's version. See: https://issues.redhat.com/browse/KOGITO-8675
	defaultKogitoServerlessWorkflowDevImage = "quay.io/kiegroup/kogito-swf-builder-nightly:latest"
	configMapWorkflowDefVolumeNamePrefix    = "wd"
	configMapWorkflowDefMountPath           = "/home/kogito/serverless-workflow-project/src/main/resources/workflows"
	requeueAfterFailure                     = 3 * time.Minute
	requeueAfterFollowDeployment            = 10 * time.Second
	requeueAfterIsRunning                   = 3 * time.Minute
	// recoverDeploymentErrorRetries how many times the operator should try to recover from a failure before giving up
	recoverDeploymentErrorRetries = 3
	// recoverDeploymentErrorInterval interval between recovering from failures
	recoverDeploymentErrorInterval = 5 * time.Minute
)

type developmentProfile struct {
	baseReconciler
}

func (d developmentProfile) GetProfile() Profile {
	return Development
}

func newDevProfileReconciler(client client.Client, logger *logr.Logger) ProfileReconciler {
	support := &stateSupport{
		logger: logger,
		client: client,
	}
	ensurers := newDevelopmentObjectEnsurers(support)

	stateMachine := newReconciliationStateMachine(logger,
		&ensureRunningDevWorkflowReconciliationState{stateSupport: support, ensurers: ensurers},
		&followDeployDevWorkflowReconciliationState{stateSupport: support},
		&recoverFromFailureDevReconciliationState{stateSupport: support})

	profile := &developmentProfile{
		baseReconciler: newBaseProfileReconciler(support, stateMachine),
	}
	logger.Info("Reconciling in", "profile", profile.GetProfile())
	return profile
}

func newDevelopmentObjectEnsurers(support *stateSupport) *devProfileObjectEnsurers {
	return &devProfileObjectEnsurers{
		deployment:        newObjectEnsurer(support.client, support.logger, defaultDeploymentCreator),
		service:           newObjectEnsurer(support.client, support.logger, defaultServiceCreator),
		workflowConfigMap: newObjectEnsurer(support.client, support.logger, workflowSpecConfigMapCreator),
	}
}

type devProfileObjectEnsurers struct {
	deployment        *objectEnsurer
	service           *objectEnsurer
	workflowConfigMap *objectEnsurer
}

type ensureRunningDevWorkflowReconciliationState struct {
	*stateSupport
	ensurers *devProfileObjectEnsurers
}

func (e *ensureRunningDevWorkflowReconciliationState) CanReconcile(workflow *operatorapi.KogitoServerlessWorkflow) bool {
	return workflow.Status.Condition == operatorapi.NoneConditionType ||
		workflow.Status.Condition == operatorapi.RunningConditionType
}

func (e *ensureRunningDevWorkflowReconciliationState) Do(ctx context.Context, workflow *operatorapi.KogitoServerlessWorkflow) (ctrl.Result, []client.Object, error) {
	var objs []client.Object

	configMap, _, err := e.ensurers.workflowConfigMap.ensure(ctx, workflow, ensureWorkflowSpecConfigMapMutator(workflow))
	if err != nil {
		return ctrl.Result{Requeue: false}, objs, err
	}
	objs = append(objs, configMap)

	deployment, _, err := e.ensurers.deployment.ensure(ctx, workflow,
		defaultDeploymentMutateVisitor(workflow),
		naiveApplyImageDeploymentMutateVisitor(defaultKogitoServerlessWorkflowDevImage),
		mountWorkflowDefConfigMapMutateVisitor(configMap.(*v1.ConfigMap)))
	if err != nil {
		return ctrl.Result{RequeueAfter: requeueAfterFailure}, objs, err
	}
	objs = append(objs, deployment)

	service, _, err := e.ensurers.service.ensure(ctx, workflow, defaultServiceMutateVisitor(workflow))
	if err != nil {
		return ctrl.Result{RequeueAfter: requeueAfterFailure}, objs, err
	}
	objs = append(objs, service)

	// TODO (once we mature the implementation): these should be done by the "onExit" handler in the state machine given that each state would have to transition

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

type followDeployDevWorkflowReconciliationState struct {
	*stateSupport
}

func (f *followDeployDevWorkflowReconciliationState) CanReconcile(workflow *operatorapi.KogitoServerlessWorkflow) bool {
	return workflow.Status.Condition == operatorapi.DeployingConditionType
}

func (f *followDeployDevWorkflowReconciliationState) Do(ctx context.Context, workflow *operatorapi.KogitoServerlessWorkflow) (ctrl.Result, []client.Object, error) {
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

type recoverFromFailureDevReconciliationState struct {
	*stateSupport
}

func (r *recoverFromFailureDevReconciliationState) CanReconcile(workflow *operatorapi.KogitoServerlessWorkflow) bool {
	return workflow.Status.Condition == operatorapi.FailedConditionType
}

func (r *recoverFromFailureDevReconciliationState) Do(ctx context.Context, workflow *operatorapi.KogitoServerlessWorkflow) (ctrl.Result, []client.Object, error) {
	if workflow.Status.RecoverFailureAttempts > recoverDeploymentErrorRetries {
		r.logger.Info("Can't recover workflow from failure after maximum attempts", "attempts", workflow.Status.RecoverFailureAttempts)
		return ctrl.Result{Requeue: false}, nil, nil
	}

	// for now, a very basic attempt to recover by rolling out the deployment
	deployment := &appsv1.Deployment{}
	if err := r.client.Get(ctx, client.ObjectKeyFromObject(workflow), deployment); err != nil {
		// if the deployment is not there, let's try to reset the status condition and make the reconciliation fix the objects
		if errors.IsNotFound(err) {
			r.logger.Info("Tried to recover from failed state, no deployment found, trying to reset the workflow conditions")
			workflow.Status.RecoverFailureAttempts = 0
			workflow.Status.Condition = operatorapi.NoneConditionType
			if _, updateErr := r.performStatusUpdate(ctx, workflow); updateErr != nil {
				return ctrl.Result{Requeue: false}, nil, updateErr
			}
			return ctrl.Result{RequeueAfter: requeueAfterFailure}, nil, nil
		}
		return ctrl.Result{Requeue: false}, nil, err
	}

	// if the deployment is progressing we might have good news
	if kubeutil.IsDeploymentAvailable(deployment) {
		workflow.Status.RecoverFailureAttempts = 0
		workflow.Status.Condition = operatorapi.RunningConditionType
		if _, updateErr := r.performStatusUpdate(ctx, workflow); updateErr != nil {
			return ctrl.Result{Requeue: false}, nil, updateErr
		}
		return ctrl.Result{RequeueAfter: requeueAfterFailure}, nil, nil
	}

	// let's try rolling out the deployment
	if err := kubeutil.MarkDeploymentToRollout(deployment); err != nil {
		return ctrl.Result{Requeue: false}, nil, err
	}
	if err := r.client.Update(ctx, deployment); err != nil {
		return ctrl.Result{Requeue: false}, nil, err
	}

	workflow.Status.RecoverFailureAttempts += 1
	if _, err := r.performStatusUpdate(ctx, workflow); err != nil {
		return ctrl.Result{Requeue: false}, nil, err
	}
	return ctrl.Result{RequeueAfter: recoverDeploymentErrorInterval}, nil, nil
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

			volumes = append(volumes, v1.Volume{
				Name: configMapWorkflowDefVolumeNamePrefix,
				VolumeSource: v1.VolumeSource{
					ConfigMap: &v1.ConfigMapVolumeSource{
						LocalObjectReference: v1.LocalObjectReference{Name: cm.Name},
					},
				},
			})
			volumeMounts = append(volumeMounts,
				v1.VolumeMount{
					Name:      configMapWorkflowDefVolumeNamePrefix,
					ReadOnly:  true,
					MountPath: configMapWorkflowDefMountPath,
				},
			)

			deployment.Spec.Template.Spec.Volumes = make([]v1.Volume, 0)
			deployment.Spec.Template.Spec.Volumes = volumes
			deployment.Spec.Template.Spec.Containers[0].VolumeMounts = make([]v1.VolumeMount, 0)
			deployment.Spec.Template.Spec.Containers[0].VolumeMounts = volumeMounts

			return nil
		}
	}
}

// rolloutDeploymentIfCMChangedMutateVisitor forces a pod refresh if the workflow definition suffered any changes.
// This method can be used as an alternative to the Kubernetes ConfigMap refresher.
//
// See: https://kubernetes.io/docs/concepts/configuration/configmap/#mounted-configmaps-are-updated-automatically
func rolloutDeploymentIfCMChangedMutateVisitor(cmOperationResult controllerutil.OperationResult) mutateVisitor {
	return func(object client.Object) controllerutil.MutateFn {
		return func() error {
			if cmOperationResult == controllerutil.OperationResultUpdated {
				deployment := object.(*appsv1.Deployment)
				err := kubeutil.MarkDeploymentToRollout(deployment)
				return err
			}
			return nil
		}
	}
}
