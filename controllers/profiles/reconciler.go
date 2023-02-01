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

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/kiegroup/kogito-serverless-operator/api/metadata"
	operatorapi "github.com/kiegroup/kogito-serverless-operator/api/v1alpha08"
)

type Profile string

var _ deploymentHandler = &baseReconciler{}
var _ serviceHandler = &baseReconciler{}

const (
	Development    Profile = "dev"
	Production     Profile = "prod"
	defaultProfile         = Production
)

var profileBuilders = map[Profile]reconcilerBuilder{
	Production:  newProductionProfile,
	Development: newDevProfile,
}

type ProfileReconciler interface {
	Reconcile(ctx context.Context) (ctrl.Result, error)
	GetProfile() Profile
}

type reconcilerBuilder func(client client.Client, scheme *runtime.Scheme, workflow *operatorapi.KogitoServerlessWorkflow) ProfileReconciler

func NewReconciler(client client.Client, scheme *runtime.Scheme, workflow *operatorapi.KogitoServerlessWorkflow) ProfileReconciler {
	return profileBuilder(workflow)(client, scheme, workflow)
}

func profileBuilder(workflow *operatorapi.KogitoServerlessWorkflow) reconcilerBuilder {
	profile := workflow.Annotations[metadata.Profile]
	if len(profile) == 0 {
		return profileBuilders[defaultProfile]
	}
	if _, ok := profileBuilders[Profile(profile)]; !ok {
		return profileBuilders[defaultProfile]
	}
	return profileBuilders[Profile(profile)]
}

type baseReconciler struct {
	workflow *operatorapi.KogitoServerlessWorkflow
	scheme   *runtime.Scheme
	client   client.Client
	logger   logr.Logger
}

func (b baseReconciler) initLogger(ctx context.Context) error {
	var err error
	b.logger, err = logr.FromContext(ctx)
	return err
}

func (b baseReconciler) performStatusUpdate(ctx context.Context) (bool, error) {
	var err error
	b.workflow.Status.Applied = b.workflow.Spec
	if err = b.client.Status().Update(ctx, b.workflow); err != nil {
		b.logger.Error(err, "Failed to update Workflow status")
		return false, err
	}
	return true, err
}

func (b baseReconciler) EnsureDeployment(ctx context.Context, image string) (*reconcile.Result, error) {
	handler := &defaultDeploymentHandler{
		workflow: b.workflow,
		scheme:   b.scheme,
		client:   b.client,
	}
	return handler.EnsureDeployment(ctx, image)
}

func (b baseReconciler) EnsureService() (*reconcile.Result, error) {
	handler := &defaultServiceHandler{
		workflow: b.workflow,
		scheme:   b.scheme,
		client:   b.client,
	}
	return handler.EnsureService()
}
