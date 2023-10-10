// Copyright 2023 Red Hat, Inc. and/or its affiliates
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package e2e

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/kiegroup/kogito-serverless-operator/test"
	"github.com/kiegroup/kogito-serverless-operator/test/utils"

	//nolint:golint
	//nolint:revive
	. "github.com/onsi/ginkgo/v2"
)

// namespace store the ns where the Operator and Operand will be executed
const namespace = "sonataflow-operator-system"

const (
	minikubePlatform  = "minikube"
	openshiftPlatform = "openshift"
)

func verifyWorkflowIsInRunningState(workflowName string) bool {
	cmd := exec.Command("kubectl", "get", "workflow", workflowName, "-n", namespace, "-o", "jsonpath={.status.conditions[?(@.type=='Running')].status}")
	if response, err := utils.Run(cmd); err != nil {
		GinkgoWriter.Println(fmt.Errorf("failed to check if greeting workflow is running: %v", err))
		return false
	} else {
		GinkgoWriter.Println(fmt.Sprintf("Got response %s", response))

		if len(strings.TrimSpace(string(response))) > 0 {
			status, err := strconv.ParseBool(string(response))
			if err != nil {
				GinkgoWriter.Println(fmt.Errorf("failed to parse result %v", err))
				return false
			}
			return status
		}
		return false
	}
}

func verifyWorkflowIsAddressable(workflowName string) bool {
	cmd := exec.Command("kubectl", "get", "workflow", workflowName, "-n", namespace, "-o", "jsonpath={.status.address.url}")
	if response, err := utils.Run(cmd); err != nil {
		GinkgoWriter.Println(fmt.Errorf("failed to check if greeting workflow is running: %v", err))
		return false
	} else {
		GinkgoWriter.Println(fmt.Sprintf("Got response %s", response))
		if len(strings.TrimSpace(string(response))) > 0 {
			_, err := url.ParseRequestURI(string(response))
			if err != nil {
				GinkgoWriter.Println(fmt.Errorf("failed to parse result %v", err))
				return false
			}
			// The response is a valid URL so the test is passed
			return true
		}
		return false
	}
}

func getSonataFlowPlatformFilename() string {
	if getClusterPlatform() == openshiftPlatform {
		return test.GetPlatformOpenshiftE2eTest()
	}
	return test.GetPlatformMinikubeE2eTest()
}

func getClusterPlatform() string {
	if v, ok := os.LookupEnv("CLUSTER_PLATFORM"); ok {
		return v
	}
	return minikubePlatform
}
