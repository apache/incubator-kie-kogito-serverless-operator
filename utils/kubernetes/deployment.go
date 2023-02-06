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

package kubernetes

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
)

// IsDeploymentAvailable verifies if the Deployment conditions match the Available status
func IsDeploymentAvailable(deployment *appsv1.Deployment) bool {
	for _, condition := range deployment.Status.Conditions {
		if condition.Type == appsv1.DeploymentAvailable &&
			condition.Status == v1.ConditionTrue {
			return true
		}
	}
	return false
}

// IsDeploymentProgressing checks if the Deployment is progressing/scaling its replicas.
// Not progressing doesn't necessarily mean that the deployment is in a failure state.
// It might be running under the required replicas, always check IsDeploymentAvailable and GetDeploymentUnavailabilityReason to understand the real condition.
func IsDeploymentProgressing(deployment *appsv1.Deployment) bool {
	if !IsDeploymentAvailable(deployment) {
		// it's not available, so let's check if it's progressing
		for _, condition := range deployment.Status.Conditions {
			if condition.Type == appsv1.DeploymentProgressing &&
				condition.Status == v1.ConditionTrue {
				return true
			}
		}
	}

	// it might be either a failure or it's available
	return false
}

// GetDeploymentUnavailabilityReason returns a string explaining why the given deployment is unavailable. If empty, there's no replica failure.
// Note that the Deployment might be available, but a second replica failed to scale. Always check IsDeploymentAvailable.
func GetDeploymentUnavailabilityReason(deployment *appsv1.Deployment) string {
	for _, condition := range deployment.Status.Conditions {
		if condition.Type == appsv1.DeploymentReplicaFailure &&
			condition.Status == v1.ConditionTrue {
			return fmt.Sprintf("deployment %s unavailable: reason %s, message %s", deployment.Name, condition.Reason, condition.Message)
		}
	}
	return ""
}
