/*
 * Copyright 2022 Red Hat, Inc. and/or its affiliates.
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

package controllers

import (
	"context"
	"fmt"
	apiv08 "github.com/davidesalerno/kogito-serverless-operator/api/v08"
	"github.com/davidesalerno/kogito-serverless-operator/builder"
	"github.com/davidesalerno/kogito-serverless-operator/platform"
	"github.com/davidesalerno/kogito-serverless-operator/utils"
	"github.com/ricardozanini/kogito-builder/api"
	"github.com/ricardozanini/kogito-builder/util/log"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// KogitoServerlessWorkflowReconciler reconciles a KogitoServerlessWorkflow object
type KogitoServerlessWorkflowReconciler struct {
	Client   client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=sw.kogito.kie.org,resources=kogitoserverlessworkflows,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=sw.kogito.kie.org,resources=kogitoserverlessworkflows/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=sw.kogito.kie.org,resources=kogitoserverlessworkflows/finalizers,verbs=update
//+kubebuilder:rbac:groups=sw.kogito.kie.org,resources=pods,verbs=get;watch;list

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// the KogitoServerlessWorkflow object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.2/pkg/reconcile
func (r *KogitoServerlessWorkflowReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrllog.FromContext(ctx)

	// Make sure the operator is allowed to act on namespace
	if ok, err := platform.IsOperatorAllowedOnNamespace(ctx, r.Client, req.Namespace); err != nil {
		return reconcile.Result{}, err
	} else if !ok {
		log.Info("Ignoring request because namespace is locked")
		return reconcile.Result{}, nil
	}

	// Fetch the Workflow instance
	workflow := &apiv08.KogitoServerlessWorkflow{}
	err := r.Client.Get(ctx, req.NamespacedName, workflow)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get KogitoServerlessWorkflow")
		return ctrl.Result{}, err
	}

	// Only process resources assigned to the operator
	if !platform.IsOperatorHandlerConsideringLock(ctx, r.Client, req.Namespace, workflow) {
		log.Info("Ignoring request because resource is not assigned to current operator")
		return reconcile.Result{}, nil
	}

	// Let's check that a Platform is defined for build the workflow
	platform := &apiv08.KogitoServerlessPlatform{}
	err = r.Client.Get(ctx, req.NamespacedName, platform)
	if err != nil {
		if !errors.IsNotFound(err) {
			return ctrl.Result{}, err
		}
		log.Info("Build not started because there is no platform in this namespace", "Namespace", req.Namespace)
		return ctrl.Result{}, nil
	}

	if workflow.Status.Condition == apiv08.NoneConditionType {
		buildable := builder.NewBuildable(r.Client, ctx)
		_, err = buildable.HandleWorkflowBuild(workflow.Name, req)
		workflow.Status.Condition = apiv08.BuildingConditionType
		r.Client.Update(ctx, workflow)
		return ctrl.Result{}, err
	}

	if workflow.Status.Condition == apiv08.BuildingConditionType {
		// Let's check that a Platform is defined for build the workflow
		buildSpec := &apiv08.KogitoServerlessBuildSpec{WorkflowId: workflow.Name}
		build := &apiv08.KogitoServerlessBuild{Spec: *buildSpec}
		err = r.Client.Get(ctx, req.NamespacedName, build)
		if err != nil {
			if !errors.IsNotFound(err) {
				return ctrl.Result{}, err
			}
			log.Info("Build not found for this workflow", "Workflow", req.Name)
			return ctrl.Result{}, nil
		}
	}

	return ctrl.Result{}, err
}

func buildEnqueueRequestsFromMapFunc(c client.Client, build *apiv08.KogitoServerlessBuild) []reconcile.Request {
	var requests []reconcile.Request
	if build.Status.BuildPhase != api.BuildPhaseSucceeded && build.Status.BuildPhase != api.BuildPhaseError {
		return requests
	}

	list := &apiv08.KogitoServerlessWorkflowList{}
	// Do global search in case of global operator (it may be using a global platform)
	var opts []client.ListOption
	if !platform.IsCurrentOperatorGlobal() {
		opts = append(opts, client.InNamespace(build.Namespace))
	}
	if err := c.List(context.Background(), list, opts...); err != nil {
		log.Error(err, "Failed to retrieve workflow list")
		return requests
	}

	for i := range list.Items {
		workflow := &list.Items[i]

		match, err := utils.SameOrMatch(build, workflow)
		if err != nil {
			log.Errorf(err, "Error matching workflow %q with kit %q", workflow.Name, build.Name)
			continue
		}
		if !match {
			continue
		}

		if workflow.Status.Condition == apiv08.BuildingConditionType ||
			workflow.Status.Condition == apiv08.RunningConditionType {
			log.Infof("Workflow %s ready, notify workflow: %s", build.Name, workflow.Name)
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: workflow.Namespace,
					Name:      workflow.Name,
				},
			})
		}
	}

	return requests
}

func platformEnqueueRequestsFromMapFunc(c client.Client, p *apiv08.KogitoServerlessPlatform) []reconcile.Request {
	var requests []reconcile.Request

	if p.Status.Phase == apiv08.PlatformPhaseReady {
		list := &apiv08.KogitoServerlessWorkflowList{}

		// Do global search in case of global operator (it may be using a global platform)
		var opts []client.ListOption
		if !platform.IsCurrentOperatorGlobal() {
			opts = append(opts, client.InNamespace(p.Namespace))
		}

		if err := c.List(context.Background(), list, opts...); err != nil {
			log.Error(err, "Failed to list integrations")
			return requests
		}

		for _, workflow := range list.Items {
			if workflow.Status.Condition == apiv08.WaitingForPlatformConditionType {
				log.Infof("Platform %s ready, wake-up workflow: %s", p.Name, workflow.Name)
				requests = append(requests, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Namespace: workflow.Namespace,
						Name:      workflow.Name,
					},
				})
			}
		}
	}
	return []reconcile.Request{}
}

// SetupWithManager sets up the controller with the Manager.
func (r *KogitoServerlessWorkflowReconciler) SetupWithManager(mgr ctrl.Manager) error {
	mgr.GetClient()
	return ctrl.NewControllerManagedBy(mgr).
		For(&apiv08.KogitoServerlessWorkflow{}).
		Watches(&source.Kind{Type: &apiv08.KogitoServerlessBuild{}}, handler.EnqueueRequestsFromMapFunc(func(c client.Object) []reconcile.Request {
			build, ok := c.(*apiv08.KogitoServerlessBuild)
			if !ok {
				log.Error(fmt.Errorf("type assertion failed: %v", c), "Failed to retrieve integration list")
				return []reconcile.Request{}
			}
			return buildEnqueueRequestsFromMapFunc(mgr.GetClient(), build)
		})).
		Watches(&source.Kind{Type: &apiv08.KogitoServerlessPlatform{}}, handler.EnqueueRequestsFromMapFunc(func(a client.Object) []reconcile.Request {
			platform, ok := a.(*apiv08.KogitoServerlessPlatform)
			if !ok {
				log.Error(fmt.Errorf("type assertion failed: %v", a), "Failed to retrieve integration list")
				return []reconcile.Request{}
			}
			return platformEnqueueRequestsFromMapFunc(mgr.GetClient(), platform)
		})).
		Complete(r)
}
