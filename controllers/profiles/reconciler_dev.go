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

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	operatorapi "github.com/kiegroup/kogito-serverless-operator/api/v1alpha08"
)

var _ ProfileReconciler = &developmentProfile{}

type developmentProfile struct {
	baseReconciler
}

func newDevProfile(client client.Client, scheme *runtime.Scheme, workflow *operatorapi.KogitoServerlessWorkflow) ProfileReconciler {
	return &productionProfile{
		baseReconciler{
			workflow: workflow,
			scheme:   scheme,
			client:   client,
		},
	}
}

func (d developmentProfile) GetProfile() Profile {
	return Development
}

func (d developmentProfile) Reconcile(ctx context.Context) (ctrl.Result, error) {
	if err := d.initLogger(ctx); err != nil {
		return ctrl.Result{}, err
	}
	//TODO implement me
	panic("implement me")
}
