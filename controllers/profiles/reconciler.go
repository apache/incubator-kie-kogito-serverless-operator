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

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"

	"github.com/kiegroup/kogito-serverless-operator/api"
	"github.com/kiegroup/kogito-serverless-operator/controllers/builder"
	"github.com/kiegroup/kogito-serverless-operator/controllers/platform"

	"k8s.io/client-go/rest"

	api2 "github.com/kiegroup/kogito-serverless-operator/container-builder/api"

	"github.com/kiegroup/kogito-serverless-operator/api/metadata"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	operatorapi "github.com/kiegroup/kogito-serverless-operator/api/v1alpha08"
	"github.com/kiegroup/kogito-serverless-operator/log"
)

// ProfileReconciler is the public interface to have access to this package and perform the actual reconciliation flow.
//
// There are a few concepts in this package that you need to understand before attempting to maintain it:
//
// 1. ProfileReconciler: it's the main interface that internal structs implement via the baseReconciler.
// Every profile must embed the baseReconciler.
//
// 2. stateSupport: is a struct with a few support objects passed around the reconciliation states like the client and recorder.
//
// 3. reconciliationStateMachine: is a struct within the ProfileReconciler that do the actual reconciliation.
// Each part of the reconciliation algorithm is a ReconciliationState that will be executed based on the ReconciliationState.CanReconcile call.
//
// 4. ReconciliationState: is where your business code should be focused on. Each state should react to a specific operatorapi.SonataFlowConditionType.
// The least conditions your state handles, the better.
// The ReconciliationState can provide specific code that will only be triggered if the workflow is in that specific condition.
//
// 5. objectCreator: are functions to create a specific Kubernetes object based on a given workflow instance. This function should return the desired default state.
//
// 6. mutateVisitor: is a function that states can pass to defaultObjectEnsurer that will be applied to a given live object during the reconciliation cycle.
// For example, if you wish to guarantee that an image in a specific container in the Deployment that you control and own won't change, make sure that your
// mutate function guarantees that.
//
// 7. defaultObjectEnsurer: is a struct for a given objectCreator to control the reconciliation and merge conditions to an object.
// A ReconciliationState may or may not have one or more ensurers. Depends on their role. There are states that just read objects, so no need to keep their desired state.
//
// See the already implemented reconciliation profiles to have a better understanding.
//
// While debugging, focus on the ReconciliationState(s), not in the profile implementation since the base algorithm is the same for every profile.
type ProfileReconciler interface {
	Reconcile(ctx context.Context, workflow *operatorapi.SonataFlow) (ctrl.Result, error)
	GetProfile() metadata.ProfileType
}

// stateSupport is the shared structure with common accessors used throughout the whole reconciliation profiles
type stateSupport struct {
	client   client.Client
	recorder record.EventRecorder
}

// performStatusUpdate updates the SonataFlow Status conditions
func (s stateSupport) performStatusUpdate(ctx context.Context, workflow *operatorapi.SonataFlow) (bool, error) {
	var err error
	workflow.Status.ObservedGeneration = workflow.Generation
	if err = s.client.Status().Update(ctx, workflow); err != nil {
		klog.V(log.E).ErrorS(err, "Failed to update Workflow status")
		return false, err
	}
	return true, err
}

// performStatusUpdate updates the SonataFlow Status conditions on the last version available
func (s stateSupport) getAndUpdateStatusWorkFlow(ctx context.Context, workflow *operatorapi.SonataFlow) (bool, error) {
	freshWorkFlow := operatorapi.SonataFlow{}

	err := s.client.Get(ctx, client.ObjectKeyFromObject(workflow), &freshWorkFlow)
	freshWorkFlow.Status = workflow.Status
	freshWorkFlow.Status.ObservedGeneration = workflow.Generation
	if err = s.client.Status().Update(ctx, &freshWorkFlow); err != nil {
		klog.V(log.E).ErrorS(err, "Failed to update Workflow status")
		s.recorder.Event(&freshWorkFlow, v1.EventTypeWarning, "SonataFlowStatusUpdateError", fmt.Sprintf("Error: %v", err))
		return false, err
	}
	return true, nil
}

func getAndUpdateStatusBuild(ctx context.Context, build *operatorapi.SonataFlowBuild, support *stateSupport) error {
	freshBuild := operatorapi.SonataFlowBuild{}
	err := support.client.Get(ctx, client.ObjectKeyFromObject(build), &freshBuild)
	freshBuild.Status = build.Status
	freshBuild.Name = build.Name
	if err = support.client.Status().Update(ctx, &freshBuild); err != nil {
		klog.V(log.E).ErrorS(err, "Failed to update Build status")
		support.recorder.Event(&freshBuild, v1.EventTypeWarning, "SonataFlowBuildStatusUpdateError", fmt.Sprintf("Error: %v", err))
		return err
	}
	return nil
}

// PostReconcile function to perform all the other operations required after the reconciliation - placeholder for null pattern usages
func (s stateSupport) PostReconcile(ctx context.Context, workflow *operatorapi.SonataFlow) error {
	//By default, we don't want to perform anything after the reconciliation, and so we will simply return no error
	return nil
}

// baseReconciler is the base structure used by every reconciliation profile.
// Use newBaseProfileReconciler to build a new reference.
type baseReconciler struct {
	*stateSupport
	reconciliationStateMachine *reconciliationStateMachine
	objects                    []client.Object
}

func newBaseProfileReconciler(support *stateSupport, stateMachine *reconciliationStateMachine) baseReconciler {
	return baseReconciler{
		stateSupport:               support,
		reconciliationStateMachine: stateMachine,
	}
}

// Reconcile does the actual reconciliation algorithm based on a set of ReconciliationState
func (b baseReconciler) Reconcile(ctx context.Context, workflow *operatorapi.SonataFlow) (ctrl.Result, error) {
	workflow.Status.Manager().InitializeConditions()
	result, objects, err := b.reconciliationStateMachine.do(ctx, workflow)
	if err != nil {
		return result, err
	}
	b.objects = objects
	klog.V(log.I).InfoS("Returning from reconciliation", "Result", result)
	return result, err
}

// ReconciliationState is an interface implemented internally by different reconciliation algorithms to perform the adequate logic for a given workflow profile
type ReconciliationState interface {
	// CanReconcile checks if this state can perform its reconciliation task
	CanReconcile(workflow *operatorapi.SonataFlow) bool
	// Do perform the reconciliation task. It returns the controller result, the objects updated, and an error if any.
	// Objects can be nil if the reconciliation state doesn't perform any updates in any Kubernetes object.
	Do(ctx context.Context, workflow *operatorapi.SonataFlow) (ctrl.Result, []client.Object, error)
	// PostReconcile performs the actions to perform after the reconciliation that are not mandatory
	PostReconcile(ctx context.Context, workflow *operatorapi.SonataFlow) error
}

// newReconciliationStateMachine builder for the reconciliationStateMachine
func newReconciliationStateMachine(states ...ReconciliationState) *reconciliationStateMachine {
	return &reconciliationStateMachine{
		states: states,
	}
}

// reconciliationStateMachine implements (sort of) the command pattern and delegate to a chain of ReconciliationState
// the actual task to reconcile in a given workflow condition
//
// TODO: implement state transition, so based on a given condition we do the status update which actively transition the object state
type reconciliationStateMachine struct {
	states []ReconciliationState
}

func (r *reconciliationStateMachine) do(ctx context.Context, workflow *operatorapi.SonataFlow) (ctrl.Result, []client.Object, error) {
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

// NewReconciler creates a new ProfileReconciler based on the given workflow and context.
func NewReconciler(client client.Client, config *rest.Config, recorder record.EventRecorder, workflow *operatorapi.SonataFlow) ProfileReconciler {
	return profileBuilder(workflow)(client, config, recorder)
}

/*
func getInnerBuild(build *operatorapi.SonataFlowBuild) (api2.ContainerBuildPhase, string) {
	containerBuild := &api2.ContainerBuild{}
	build.Status.GetInnerBuild(containerBuild)
	phaseCurrent := containerBuild.Status.Phase
	return phaseCurrent, containerBuild.Status.Error
}*/

func getNamespaceConfigMap(c context.Context, client client.Client, name string, namespace string) (v1.ConfigMap, error) {
	configMap := v1.ConfigMap{}
	configMapId := types.NamespacedName{Name: name, Namespace: namespace}
	if err := client.Get(c, configMapId, &configMap); err != nil {
		return configMap, err
	}
	return configMap, nil
}

func getActivePlatform(ctx context.Context, workflow *operatorapi.SonataFlow, client client.Client, stateSupport *stateSupport) (*operatorapi.SonataFlowPlatform, error) {
	activePlatform, err := platform.GetActivePlatform(ctx, client, workflow.Namespace)
	if err != nil {
		if errors.IsNotFound(err) {
			// Recovery isn't possible just signal on events and on log
			msg := "No active Platform for namespace %s so the workflow cannot be built."
			stateSupport.recorder.Event(workflow, v1.EventTypeWarning, "SonataFlowPlatformError", fmt.Sprintf(msg, workflow.Namespace))
			workflow.Status.Manager().MarkFalse(api.RunningConditionType, api.WaitingForPlatformReason, msg, workflow.Namespace)
			_, err = stateSupport.getAndUpdateStatusWorkFlow(ctx, workflow)
		}
		// Recovery isn't possible just signal on events and on log
		klog.V(log.E).ErrorS(err, "Failed to get active platform")
		stateSupport.recorder.Event(workflow, v1.EventTypeWarning, "SonataFlowStatusUpdateError 1", fmt.Sprintf("Failed to get active platform, error: %v", err))
	}
	return activePlatform, err
}

func restartBuild(ctx context.Context, stateSupport *stateSupport, workflow *operatorapi.SonataFlow, activePlatform *operatorapi.SonataFlowPlatform, build *operatorapi.SonataFlowBuild) {
	if activePlatform.Generation > workflow.Status.ObservedPlatformGeneration {
		workflow.Status.Manager().MarkFalse(api.BuiltConditionType, api.BuildFailedReason, "PlatformUpdated", workflow.Namespace)
		stateSupport.client.Status().Update(ctx, workflow)
	}
	// mark to restart
	build.Status.BuildPhase = operatorapi.BuildPhaseNone
	build.Status.BuildAttemptsAfterError = 0
	build.Status.InnerBuild = runtime.RawExtension{}
	restartErr := getAndUpdateStatusBuild(ctx, build, stateSupport)
	if restartErr != nil {
		buildManager := builder.NewSonataFlowBuildManager(ctx, stateSupport.client)
		freshBuild, restartErr := buildManager.GetOrCreateBuildWithPlatform(activePlatform, workflow)
		freshBuild.Status.BuildPhase = operatorapi.BuildPhaseNone
		freshBuild.Status.BuildAttemptsAfterError = 0
		freshBuild.Status.InnerBuild = runtime.RawExtension{}
		restartErr = getAndUpdateStatusBuild(ctx, freshBuild, stateSupport)
		if restartErr != nil {
			klog.V(log.E).ErrorS(restartErr, "Error on Restart Build updae status")
		}
	}
}

func handleMultipleBuildsAfterError(ctx context.Context, build *operatorapi.SonataFlowBuild, workflow *operatorapi.SonataFlow, stateSupport stateSupport, activePlatform operatorapi.SonataFlowPlatform) bool {
	if build.Status.BuildAttemptsAfterError == 0 {
		build.Status.BuildAttemptsAfterError = 1
	}

	msg := fmt.Sprintf("Build attempt number %v is in failed state", build.Status.BuildAttemptsAfterError)
	if build.Status.BuildAttemptsAfterError < activePlatform.Spec.Build.Template.BuildAttemptsAfterError {
		build.Status.BuildAttemptsAfterError = build.Status.BuildAttemptsAfterError + 1
		updateErr := stateSupport.client.Status().Update(ctx, build)
		klog.V(log.I).Info(msg)
		stateSupport.recorder.Event(workflow, v1.EventTypeWarning, "SonataFlowBuildError", msg)
		if updateErr != nil {
			klog.V(log.I).Info(fmt.Sprintf("Error updating Build: %v", updateErr))
			stateSupport.recorder.Event(workflow, v1.EventTypeWarning, "Error updating Build", fmt.Sprintf("Error updating Build: %v", updateErr))
		}
	} else {
		//We have surpassed the number of failed builds configured, we are going to change the condition to WaitingForChanges from the user
		msgFinal := fmt.Sprintf(" Workflow %s build is in failed state, stop to build after %v attempts and waiting to fix the problem. Try to fix the problem or delete the SonataFlowBuild to restart a new build cycle. error %s", workflow.Name, build.Status.BuildAttemptsAfterError, build.Status.Error)
		klog.V(log.I).Info(msgFinal)
		stateSupport.recorder.Event(workflow, v1.EventTypeWarning, "SonataFlowBuildError", msgFinal)
		workflow.Status.Manager().MarkFalse(api.BuiltConditionType, api.WaitingForWrongConfigurationReason, "WaitingForConfigChanges")
		workflow.Status.ObservedPlatformGeneration = activePlatform.Generation
		_, updateErr := stateSupport.getAndUpdateStatusWorkFlow(ctx, workflow)
		if updateErr != nil {
			klog.V(log.E).ErrorS(updateErr, "failed to update status update workflow")
		}
		return true
	}
	return false
}

func getInnerBuild(build *operatorapi.SonataFlowBuild) (api2.ContainerBuildPhase, string) {
	containerBuild := &api2.ContainerBuild{}
	build.Status.GetInnerBuild(containerBuild)
	phaseCurrent := containerBuild.Status.Phase
	return phaseCurrent, containerBuild.Status.Error
}

func getContextObjects(ctx context.Context, workflow *operatorapi.SonataFlow, support *stateSupport) (*operatorapi.SonataFlowPlatform, *operatorapi.SonataFlowBuild, v1.ConfigMap, error) {
	activePlatform, err := getActivePlatform(ctx, workflow, support.client, support)
	if err != nil {
		return nil, nil, v1.ConfigMap{}, err
	}
	// If there is an active platform we have got all the information to build but...
	// ...let's check before if we have got already a build!
	buildManager := builder.NewSonataFlowBuildManager(ctx, support.client)
	build, err := buildManager.GetOrCreateBuildWithPlatform(activePlatform, workflow)
	if err != nil {
		support.recorder.Event(workflow, v1.EventTypeWarning, "SonataFlowGetOrCreateBuildError", fmt.Sprintf("Error: %v", err))
		return activePlatform, nil, v1.ConfigMap{}, err
	}
	cm, err := getNamespaceConfigMap(ctx, support.client, "sonataflow-"+workflow.Name+"-builder", workflow.Namespace)
	if err != nil {
		return activePlatform, build, v1.ConfigMap{}, nil
	}
	return activePlatform, build, cm, nil
}
