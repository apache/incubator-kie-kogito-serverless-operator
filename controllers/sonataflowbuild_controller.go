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

package controllers

import (
	"context"
	"fmt"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/kiegroup/kogito-serverless-operator/container-builder/util/log"

	buildv1 "github.com/openshift/api/build/v1"
	imgv1 "github.com/openshift/api/image/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/kiegroup/kogito-serverless-operator/utils"

	operatorapi "github.com/kiegroup/kogito-serverless-operator/api/v1alpha08"
	"github.com/kiegroup/kogito-serverless-operator/controllers/builder"
)

// SonataFlowBuildReconciler reconciles a SonataFlowBuild object
type SonataFlowBuildReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
	Config   *rest.Config
}

const (
	requeueAfterForNewBuild     = 10 * time.Second
	requeueAfterForBuildRunning = 30 * time.Second
)

// +kubebuilder:rbac:groups=sonataflow.org,resources=sonataflowbuilds,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=sonataflow.org,resources=sonataflowbuilds/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=sonataflow.org,resources=sonataflowbuilds/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// the SonataFlowBuild object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.1/pkg/reconcile
func (r *SonataFlowBuildReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrllog.FromContext(ctx)
	build := &operatorapi.SonataFlowBuild{}
	err := r.Client.Get(ctx, req.NamespacedName, build)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		errMsg := "Failed to get the SonataFlowBuild"
		log.Error(err, errMsg)
		r.Recorder.Event(build, corev1.EventTypeWarning, "SonataFlowBuildError", fmt.Sprintf("Error: %v", err))
		return ctrl.Result{}, err
	}

	buildManager, err := builder.NewBuildManager(ctx, r.Client, r.Config, build.Name, build.Namespace)
	if err != nil {
		errMsg := "Failed to get create a build manager to handle the workflow build"
		log.Error(err, errMsg)
		r.Recorder.Event(build, corev1.EventTypeWarning, "SonataFlowBuildManagerError", fmt.Sprintf("Error: %v", err))
		return ctrl.Result{}, err
	}

	//phase := build.Status.BuildPhase
	if build.Status.BuildPhase == operatorapi.BuildPhaseNone {
		if err = buildManager.Schedule(build); err != nil {
			r.Recorder.Event(build, corev1.EventTypeWarning, "SonataFlowBuildManagerScheduleError", fmt.Sprintf("Error: %v", err))
			return ctrl.Result{}, err
		}
		_, err := r.manageStatusUpdate(ctx, build)
		return ctrl.Result{RequeueAfter: requeueAfterForNewBuild}, err
		// TODO: this smells, why not just else? review in the future: https://issues.redhat.com/browse/KOGITO-8785
	} else if build.Status.BuildPhase != operatorapi.BuildPhaseSucceeded && build.Status.BuildPhase != operatorapi.BuildPhaseError && build.Status.BuildPhase != operatorapi.BuildPhaseFailed {
		beforeReconcilePhase := build.Status.BuildPhase
		if err = buildManager.Reconcile(build); err != nil {
			r.Recorder.Event(build, corev1.EventTypeWarning, "SonataFlowBuildManagerReconcileError", fmt.Sprintf("Error: %v", err))
			return ctrl.Result{}, err
		}
		if beforeReconcilePhase != build.Status.BuildPhase {
			_, err := r.manageStatusUpdate(ctx, build)
			if err != nil {
				r.Recorder.Event(build, corev1.EventTypeWarning, "SonataFlowStatusUpdateError 4", fmt.Sprintf("Error: %v", err))
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{RequeueAfter: requeueAfterForBuildRunning}, nil
	}
	return ctrl.Result{}, nil
}

func (r *SonataFlowBuildReconciler) manageStatusUpdate(ctx context.Context, instance *operatorapi.SonataFlowBuild) (bool, error) {

	/*
		err = r.client.Status().Patch(ctx, ctx.Object, client.MergeFrom(cleanObj), &client.PatchOptions{FieldManager: r.name})
			if err != nil && !kerrors.IsNotFound(err) {
				// If it was a NotFound error, the object was probably already deleted so just ignore the error and return the existing result.
				return ctx.result, errors.Wrap(err, "error patching status")
			}


	*/

	freshBuild := operatorapi.SonataFlowBuild{}
	err := r.Get(ctx, client.ObjectKeyFromObject(instance), &freshBuild)
	log.Info(fmt.Sprintf("Freshbuild %v", freshBuild))
	freshBuild.Status = instance.Status
	if freshBuild.Name != "" {
		if err = r.Status().Update(ctx, &freshBuild); err != nil {
			log.Error(err, "Failed to update Build status")
			r.Recorder.Event(&freshBuild, corev1.EventTypeWarning, "SonataFlowStatusUpdateError 3", fmt.Sprintf("Error: %v", err))
			return false, err
		} else {
			r.Recorder.Event(instance, corev1.EventTypeNormal, "Updated", fmt.Sprintf("Updated buildphase to  %s", instance.Status.BuildPhase))
		}
	} else {
		return false, nil
	}
	return true, nil

}

// SetupWithManager sets up the controller with the Manager.
func (r *SonataFlowBuildReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if utils.IsOpenShift() {
		return ctrl.NewControllerManagedBy(mgr).
			For(&operatorapi.SonataFlowBuild{}).
			Owns(&buildv1.BuildConfig{}).
			Owns(&imgv1.ImageStream{}).
			Watches(&operatorapi.SonataFlowPlatform{}, handler.EnqueueRequestsFromMapFunc(func(c context.Context, a client.Object) []reconcile.Request {
				platform, ok := a.(*operatorapi.SonataFlowPlatform)
				if !ok {
					log.Error(fmt.Errorf("type assertion failed: %v", a), "Failed to retrieve platform list")
					return []reconcile.Request{}
				}
				return platformEnqueueRequestsFromMapFunc(mgr.GetClient(), platform, mgr.GetEventRecorderFor("build-controller"))
			})).
			Watches(&corev1.ConfigMap{}, handler.EnqueueRequestsFromMapFunc(func(c context.Context, a client.Object) []reconcile.Request {
				configMap, ok := a.(*corev1.ConfigMap)
				if !ok {
					log.Error(fmt.Errorf("type assertion failed: %v", a), "Failed to retrieve platform list")
					return []reconcile.Request{}
				}
				return configMapEnqueueRequestsFromMapFunc(mgr.GetClient(), configMap, mgr.GetEventRecorderFor("build-controller"))
			})).
			Complete(r)
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorapi.SonataFlowBuild{}).
		Watches(&operatorapi.SonataFlowPlatform{}, handler.EnqueueRequestsFromMapFunc(func(c context.Context, a client.Object) []reconcile.Request {
			platform, ok := a.(*operatorapi.SonataFlowPlatform)
			if !ok {
				log.Error(fmt.Errorf("type assertion failed: %v", a), "Failed to retrieve platform list")
				return []reconcile.Request{}
			}
			return platformEnqueueRequestsFromMapFunc(mgr.GetClient(), platform, mgr.GetEventRecorderFor("build-controller"))
		})).
		Watches(&corev1.ConfigMap{}, handler.EnqueueRequestsFromMapFunc(func(c context.Context, a client.Object) []reconcile.Request {
			configMap, ok := a.(*corev1.ConfigMap)
			if !ok {
				log.Error(fmt.Errorf("type assertion failed: %v", a), "Failed to retrieve platform list")
				return []reconcile.Request{}
			}
			return configMapEnqueueRequestsFromMapFunc(mgr.GetClient(), configMap, mgr.GetEventRecorderFor("build-controller"))
		})).
		Complete(r)
}
