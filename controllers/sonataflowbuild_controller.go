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
	"reflect"
	"time"

	"k8s.io/klog/v2"

	buildv1 "github.com/openshift/api/build/v1"
	imgv1 "github.com/openshift/api/image/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kiegroup/kogito-serverless-operator/utils"

	operatorapi "github.com/kiegroup/kogito-serverless-operator/api/v1alpha08"
	"github.com/kiegroup/kogito-serverless-operator/controllers/builder"
	"github.com/kiegroup/kogito-serverless-operator/log"
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
	build := &operatorapi.SonataFlowBuild{}
	err := r.Client.Get(ctx, req.NamespacedName, build)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		klog.V(log.E).ErrorS(err, "Failed to get the SonataFlowBuild")
		return ctrl.Result{}, err
	}

	phase := build.Status.BuildPhase
	buildManager, err := builder.NewBuildManager(ctx, r.Client, r.Config, build.Name, build.Namespace)
	if err != nil {
		klog.V(log.E).ErrorS(err, "Failed to get create a build manager to handle the workflow build")
		return ctrl.Result{}, err
	}

	if phase == operatorapi.BuildPhaseNone {
		if err = buildManager.Schedule(build); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: requeueAfterForNewBuild}, r.manageStatusUpdate(ctx, build)
		// TODO: this smells, why not just else? review in the future: https://issues.redhat.com/browse/KOGITO-8785
	} else if phase != operatorapi.BuildPhaseSucceeded && phase != operatorapi.BuildPhaseError && phase != operatorapi.BuildPhaseFailed {
		beforeReconcileStatus := build.Status.DeepCopy()
		if err = buildManager.Reconcile(build); err != nil {
			return ctrl.Result{}, err
		}
		if !reflect.DeepEqual(build.Status.InnerBuild, beforeReconcileStatus.InnerBuild) {
			if err = r.manageStatusUpdate(ctx, build); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{RequeueAfter: requeueAfterForBuildRunning}, nil
	}

	return ctrl.Result{}, nil
}

func (r *SonataFlowBuildReconciler) manageStatusUpdate(ctx context.Context, instance *operatorapi.SonataFlowBuild) error {
	err := r.Status().Update(ctx, instance)
	if err == nil {
		r.Recorder.Event(instance, corev1.EventTypeNormal, "Updated", fmt.Sprintf("Updated buildphase to  %s", instance.Status.BuildPhase))
	}
	return err
}

// SetupWithManager sets up the controller with the Manager.
func (r *SonataFlowBuildReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if utils.IsOpenShift() {
		return ctrl.NewControllerManagedBy(mgr).
			For(&operatorapi.SonataFlowBuild{}).
			Owns(&buildv1.BuildConfig{}).
			Owns(&imgv1.ImageStream{}).
			Complete(r)
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorapi.SonataFlowBuild{}).
		Complete(r)
}
