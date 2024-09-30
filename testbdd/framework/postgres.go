/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

package framework

import (
	"fmt"

	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"

	framework "github.com/apache/incubator-kie-kogito-serverless-operator/bddframework/pkg/framework"
)

const (
	postgresDeploymentTimeoutMin = 5

	postgresName           = "postgres"
	postgresDeploymentName = postgresName

	postgresDeploymentPullImageSecretPrefix = postgresDeploymentName + "-dockercfg"
)

// WaitForPostgresDeploymentRunning waits for Postgres operator running
func WaitForPostgresDeploymentRunning(namespace string) error {
	return framework.WaitForOnOpenshift(namespace, "Postgres deployment running", postgresDeploymentTimeoutMin,
		func() (bool, error) {
			running, err := IsPostgresDeploymentRunning(namespace)
			if err != nil {
				return false, err
			}

			// If not running, make sure the image pull secret is present in pod
			// If not present, delete the pod to allow its reconstruction with correct pull secret
			// Note that this is specific to Openshift
			if !running && framework.IsOpenshift() {
				podList, err := framework.GetPodsWithLabels(namespace, map[string]string{"name": postgresDeploymentName})
				if err != nil {
					framework.GetLogger(namespace).Error(err, "Error while trying to retrieve Postgres pod")
					return false, nil
				}
				for _, pod := range podList.Items {
					if !framework.CheckPodHasImagePullSecretWithPrefix(&pod, postgresDeploymentPullImageSecretPrefix) {
						// Delete pod as it has been misconfigured (missing pull secret)
						framework.GetLogger(namespace).Info("Postgres pod does not have the image pull secret needed. Deleting it to renew it.")
						err := framework.DeleteObject(&pod)
						if err != nil {
							framework.GetLogger(namespace).Error(err, "Error while trying to delete Postgres pod")
							return false, nil
						}
					}
				}
			}
			return running, nil
		})
}

// IsPostgresDeploymentRunning returns whether Postgres deployment is running
func IsPostgresDeploymentRunning(namespace string) (bool, error) {
	exists, err := PostgresDeploymentExists(namespace)
	if err != nil {
		if exists {
			return false, nil
		}
		return false, err
	}

	return exists, nil
}

// PostgresDeploymentExists returns whether Postgres deployment exists and is running. If it is existing but not running, it returns true and an error
func PostgresDeploymentExists(namespace string) (bool, error) {
	framework.GetLogger(namespace).Debug("Checking Postgres", "Deployment", postgresDeploymentName, "Namespace", namespace)

	postgresDeployment := &v1.Deployment{}
	namespacedName := types.NamespacedName{Namespace: namespace, Name: postgresDeploymentName} // done to reuse the framework function
	if exists, err := framework.GetObjectWithKey(namespacedName, postgresDeployment); err != nil {
		return false, fmt.Errorf("error while trying to look for Deploment %s: %v ", postgresDeploymentName, err)
	} else if !exists {
		return false, nil
	}

	if postgresDeployment.Status.AvailableReplicas == 0 {
		return true, fmt.Errorf("postgres seems to be created in the namespace '%s', but there's no available pods replicas deployed ", namespace)
	}

	if postgresDeployment.Status.AvailableReplicas != 1 {
		return false, fmt.Errorf("unexpected number of pods for Postgres. Expected %d but got %d ", 1, postgresDeployment.Status.AvailableReplicas)
	}

	return true, nil
}
