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

	"sigs.k8s.io/controller-runtime/pkg/client"

	operatorapi "github.com/kiegroup/kogito-serverless-operator/api/v1alpha08"
	"github.com/kiegroup/kogito-serverless-operator/platform"
)

// objectCreationCondition is the func that will be executed before object creation to verify if we have to create or not this object.
// It is optional, and it will skip the object creation if it returns false
type objectCreationCondition func(ctx context.Context, client client.Client, workflow *operatorapi.KogitoServerlessWorkflow) (bool, error)

// onlyForOpenShiftClusters returns true only if the used Platform is running on an OpenShift cluster
func onlyForOpenShiftClusters(ctx context.Context, client client.Client, workflow *operatorapi.KogitoServerlessWorkflow) (bool, error) {
	// Fetch the Platform build with the information we need for the build
	pl, err := platform.GetActivePlatform(ctx, client, workflow.Namespace)
	if err != nil {
		return false, err
	}
	return pl.Spec.Cluster == operatorapi.PlatformClusterOpenShift, nil
}

// onlyForKubernetesClusters returns true only if the used Platform is running on a Kubernetes cluster (i.e. Minikube, KIND, K3S, GKE, AKS, EKS etc...)
func onlyForKubernetesClusters(ctx context.Context, client client.Client, workflow *operatorapi.KogitoServerlessWorkflow) (bool, error) {
	// Fetch the Platform build with the information we need for the build
	pl, err := platform.GetActivePlatform(ctx, client, workflow.Namespace)
	if err != nil {
		return false, err
	}
	return pl.Spec.Cluster == operatorapi.PlatformClusterKubernetes, nil
}
