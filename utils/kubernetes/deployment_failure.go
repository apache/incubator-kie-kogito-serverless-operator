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
	"context"
	"fmt"

	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

/*
Possible failure causes:

1. Image not available (ImagePullFailure, ImagePullBackOff)
- Nothing to do, signal an event that the image is wrong (fwd deployment error message)

2. Exceeding resource limits (Pod Stucks on Pending) - verify ReplicaSet
- Try to scale it back to 1

3. Application failure, usually "CrashLoopBackOff"
- Show the pod's log error in the event

4. Ensure properties configMap is correctly defined ("RunContainerError")
- Nothing to do, ensurer always do that

5. Ensure that the mount volumes are there ("RunContainerError")
- Nothing to do, ensurer always do that

*/

var _ DeploymentUnavailabilityReader = &deploymentUnavailabilityReader{}

// DeploymentUnavailabilityReader implementations find the reason behind a deployment failure
type DeploymentUnavailabilityReader interface {
	// ReasonMessage returns the reason message in string format fetched from the culprit resource. For example, a Container status.
	ReasonMessage() (string, error)
}

// DeploymentTroubleshooter creates a new DeploymentUnavailabilityReader for finding out why a deployment failed
func DeploymentTroubleshooter(client client.Client, deployment *v1.Deployment, container string) DeploymentUnavailabilityReader {
	return &deploymentUnavailabilityReader{
		c:          client,
		deployment: deployment,
		container:  container,
	}
}

type deploymentUnavailabilityReader struct {
	c          client.Client
	deployment *v1.Deployment
	container  string
}

// ReasonMessage tries to find a human-readable reason message for why the deployment is not available or in a failed state.
// This implementation fetches the given container status for this information.
//
// Future implementations might look in other objects for a more specific reason.
// Additionally, future work might involve returning a typed Reason, so controllers may take actions depending on what have happened.
func (d deploymentUnavailabilityReader) ReasonMessage() (string, error) {
	podList := &corev1.PodList{}
	// ideally we should get the latest replicaset, then the pods.
	// problem is that we don't have a reliable field to get this information,
	// it's in a message within the deployment status
	// since this use case is only to show the deployment problem for user's troubleshooting, it's ok showing all of them.
	// additionally, we are using a unique label identifier for matching
	opts := &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(d.deployment.Spec.Selector.MatchLabels),
		/*
			FieldSelector: fields.AndSelectors(
				fields.OneTermNotEqualSelector("status.phase", "Running"),
				fields.OneTermNotEqualSelector("status.phase", "Succeeded")),

		*/
		Namespace: d.deployment.Namespace,
	}

	if err := d.c.List(context.TODO(), podList, opts); err != nil {
		return "", err
	}

	for _, pod := range podList.Items {
		if pod.Status.Phase == corev1.PodRunning || pod.Status.Phase == corev1.PodSucceeded {
			continue
		}
		for _, container := range pod.Status.ContainerStatuses {
			if container.Name == d.container {
				if !container.Ready {
					if container.State.Waiting != nil {
						return fmt.Sprintf("ContainerNotReady: (%s) %s", container.State.Waiting.Reason, container.State.Waiting.Message), nil
					}
					if container.State.Terminated != nil {
						return fmt.Sprintf("ContainerNotReady: (%s) %s", container.State.Terminated.Reason, container.State.Terminated.Message), nil
					}
				}
			}
		}
	}

	return "", nil
}
