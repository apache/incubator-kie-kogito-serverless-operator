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

package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/apache/incubator-kie-kogito-serverless-operator/test"
	"github.com/apache/incubator-kie-kogito-serverless-operator/test/utils"

	//nolint:golint
	//nolint:revive
	. "github.com/onsi/ginkgo/v2"

	//nolint:golint
	//nolint:revive
	. "github.com/onsi/gomega"
)

// sonataflow_operator_namespace store the ns where the Operator and Operand will be executed
const sonataflow_operator_namespace = "sonataflow-operator-system"

var _ = Describe("SonataFlow Operator", Ordered, func() {

	var targetNamespace string
	BeforeEach(func() {
		targetNamespace = fmt.Sprintf("test-%d", rand.Intn(1024)+1)
		cmd := exec.Command("kubectl", "create", "namespace", targetNamespace)
		_, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred())
	})
	AfterEach(func() {
		// Remove resources in test namespace
		if !CurrentGinkgoTestDescription().Failed && len(targetNamespace) > 0 {
			cmd := exec.Command("kubectl", "delete", "namespace", targetNamespace, "--wait")
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())
		}
	})

	Describe("ensure that Operator and Operand(s) can run in restricted namespaces", func() {
		projectDir, _ := utils.GetProjectDir()

		It("should successfully deploy the Simple Workflow in prod ops mode and verify if it's running", func() {
			By("creating an instance of the SonataFlow Operand(CR)")
			EventuallyWithOffset(1, func() error {
				cmd := exec.Command("kubectl", "apply", "-f", filepath.Join(projectDir,
					"test/testdata/"+test.SonataFlowSimpleOpsYamlCR), "-n", sonataflow_operator_namespace)
				_, err := utils.Run(cmd)
				return err
			}, 2*time.Minute, time.Second).Should(Succeed())

			By("check the workflow is in running state")
			EventuallyWithOffset(1, func() bool { return verifyWorkflowIsInRunningState("simple", targetNamespace) }, 15*time.Minute, 30*time.Second).Should(BeTrue())

			EventuallyWithOffset(1, func() error {
				cmd := exec.Command("kubectl", "delete", "-f", filepath.Join(projectDir,
					"test/testdata/"+test.SonataFlowSimpleOpsYamlCR), "-n", sonataflow_operator_namespace)
				_, err := utils.Run(cmd)
				return err
			}, 2*time.Minute, time.Second).Should(Succeed())
		})

		It("should successfully deploy the Greeting Workflow in prod mode and verify if it's running", func() {
			By("creating external resources DataInputSchema configMap")
			EventuallyWithOffset(1, func() error {
				cmd := exec.Command("kubectl", "apply", "-f", filepath.Join(projectDir,
					"test/testdata/"+test.SonataFlowGreetingsDataInputSchemaConfig), "-n", sonataflow_operator_namespace)
				_, err := utils.Run(cmd)
				return err
			}, 2*time.Minute, time.Second).Should(Succeed())

			By("creating an instance of the SonataFlow Operand(CR)")
			EventuallyWithOffset(1, func() error {
				cmd := exec.Command("kubectl", "apply", "-f", filepath.Join(projectDir,
					"test/testdata/"+test.SonataFlowGreetingsWithDataInputSchemaCR), "-n", sonataflow_operator_namespace)
				_, err := utils.Run(cmd)
				return err
			}, 2*time.Minute, time.Second).Should(Succeed())

			By("check the workflow is in running state")
			EventuallyWithOffset(1, func() bool { return verifyWorkflowIsInRunningState("greeting", targetNamespace) }, 15*time.Minute, 30*time.Second).Should(BeTrue())

			EventuallyWithOffset(1, func() error {
				cmd := exec.Command("kubectl", "delete", "-f", filepath.Join(projectDir,
					"test/testdata/"+test.SonataFlowGreetingsWithDataInputSchemaCR), "-n", sonataflow_operator_namespace)
				_, err := utils.Run(cmd)
				return err
			}, 2*time.Minute, time.Second).Should(Succeed())
		})

		It("should successfully deploy the orderprocessing workflow in devmode and verify if it's running", func() {

			By("creating an instance of the SonataFlow Workflow in DevMode")
			EventuallyWithOffset(1, func() error {
				cmd := exec.Command("kubectl", "apply", "-f", filepath.Join(projectDir,
					test.GetSonataFlowE2eOrderProcessingFolder()), "-n", sonataflow_operator_namespace)
				_, err := utils.Run(cmd)
				return err
			}, 2*time.Minute, time.Second).Should(Succeed())

			By("check the workflow is in running state")
			EventuallyWithOffset(1, func() bool { return verifyWorkflowIsInRunningState("orderprocessing", targetNamespace) }, 10*time.Minute, 30*time.Second).Should(BeTrue())

			cmdLog := exec.Command("kubectl", "logs", "orderprocessing", "-n", sonataflow_operator_namespace)
			if responseLog, errLog := utils.Run(cmdLog); errLog == nil {
				GinkgoWriter.Println(fmt.Sprintf("devmode podlog %s", responseLog))
			}

			By("check that the workflow is addressable")
			EventuallyWithOffset(1, func() bool { return verifyWorkflowIsAddressable("orderprocessing", targetNamespace) }, 10*time.Minute, 30*time.Second).Should(BeTrue())

			EventuallyWithOffset(1, func() error {
				cmd := exec.Command("kubectl", "delete", "-f", filepath.Join(projectDir,
					test.GetSonataFlowE2eOrderProcessingFolder()), "-n", sonataflow_operator_namespace)
				_, err := utils.Run(cmd)
				return err
			}, 2*time.Minute, time.Second).Should(Succeed())
		})

	})

})

var _ = Describe("Validate the persistence ", Ordered, func() {

	BeforeAll(func() {

		operatorImageName, err := utils.GetOperatorImageName()
		ExpectWithOffset(1, err).NotTo(HaveOccurred())

		By("deploying the controller-manager")
		cmd := exec.Command("make", "deploy", fmt.Sprintf("IMG=%s", operatorImageName))

		outputMake, err := utils.Run(cmd)
		fmt.Println(string(outputMake))
		ExpectWithOffset(1, err).NotTo(HaveOccurred())

		By("validating that the controller-manager pod is running as expected")

		By("Wait for SonatatFlowPlatform CR to complete deployment")
		// wait for service deployments to be ready
		EventuallyWithOffset(1, func() error {
			cmd = exec.Command("kubectl", "wait", "pod", "-n", sonataflow_operator_namespace, "-l", "control-plane=controller-manager", "--for", "condition=Ready", "--timeout=5s")
			_, err = utils.Run(cmd)
			return err
		}, 10*time.Minute, 5).Should(Succeed())
	})

	AfterAll(func() {
		By("removing manager namespace")
		cmd := exec.Command("make", "undeploy")
		_, _ = utils.Run(cmd)
	})

	var (
		ns string
	)

	BeforeEach(func() {
		ns = fmt.Sprintf("test-%d", rand.Intn(1024)+1)
		cmd := exec.Command("kubectl", "create", "namespace", ns)
		_, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred())
	})
	AfterEach(func() {
		// Remove platform CR if it exists
		if len(ns) > 0 {
			cmd := exec.Command("kubectl", "delete", "namespace", ns, "--wait")
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())
		}

	})

	DescribeTable("when deploying a SonataFlow CR with PostgreSQL persistence", func(testcaseDir string) {
		By("Deploy the CR")
		var manifests []byte
		EventuallyWithOffset(1, func() error {
			var err error
			cmd := exec.Command("kubectl", "kustomize", testcaseDir)
			manifests, err = utils.Run(cmd)
			return err
		}, time.Minute, time.Second).Should(Succeed())
		cmd := exec.Command("kubectl", "create", "-n", ns, "-f", "-")
		cmd.Stdin = bytes.NewBuffer(manifests)
		_, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred())
		By("Wait for SonatatFlow CR to complete deployment")
		// wait for service deployments to be ready
		EventuallyWithOffset(1, func() error {
			cmd = exec.Command("kubectl", "wait", "pod", "-n", ns, "-l", "sonataflow.org/workflow-app=callbackstatetimeouts", "--for", "condition=Ready", "--timeout=5s")
			out, err := utils.Run(cmd)
			GinkgoWriter.Printf("%s\n", string(out))
			return err
		}, 10*time.Minute, 5).Should(Succeed())

		By("Evaluate status of the workflow's pod database connection health endpoint")
		cmd = exec.Command("kubectl", "get", "pod", "-l", "sonataflow.org/workflow-app=callbackstatetimeouts", "-n", ns, "-ojsonpath={.items[*].metadata.name}")
		output, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred())
		EventuallyWithOffset(1, func() bool {
			for _, pn := range strings.Split(string(output), " ") {
				h, err := getHealthFromPod(pn, ns)
				if err != nil {
					continue
				}
				Expect(h.Status).To(Equal(upStatus), "Pod health is not UP")
				for _, c := range h.Checks {
					if c.Name == "Database connections health check" {
						Expect(c.Status).To(Equal(upStatus), "Pod's database connection is not UP")
						return true
					}
				}
			}
			return false
		}, 10*time.Minute).Should(BeTrue())
	},
		Entry("defined in the workflow from an existing kubernetes service as a reference", test.GetSonataFlowE2EWorkflowPersistenceSampleDataDirectory("by_service")),
	)

})

type health struct {
	Status string  `json:"status"`
	Checks []check `json:"checks"`
}

type check struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

var (
	upStatus string = "UP"
)

func getHealthFromPod(name, namespace string) (*health, error) {
	// iterate over all containers to find the one that responds to the HTTP health endpoint
	Expect(name).NotTo(BeEmpty(), "pod name is empty")
	cmd := exec.Command("kubectl", "get", "pod", name, "-n", namespace, "-o", `jsonpath={.spec.containers[*].name}`)
	output, err := utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred())
	var errs error
	for _, cname := range strings.Split(string(output), " ") {
		var h *health
		h, err = getHealthStatusInContainer(name, cname, namespace)
		if err == nil {
			return h, nil
		}
		errs = fmt.Errorf("%v; %w", err, errs)
	}
	return nil, errs
}

func getHealthStatusInContainer(podName string, containerName string, namespace string) (*health, error) {
	h := health{}
	cmd := exec.Command("kubectl", "exec", "-t", podName, "-n", namespace, "-c", containerName, "--", "curl", "-s", "localhost:8080/q/health")
	output, err := utils.Run(cmd)
	GinkgoWriter.Printf("%s\n", string(output))
	Expect(err).NotTo(HaveOccurred())
	err = json.Unmarshal(output, &h)
	if err != nil {
		return nil, fmt.Errorf("failed to execute curl command against health endpoint in container %s:%v", containerName, err)
	}
	return &h, nil
}
func verifyWorkflowIsInRunningStateInNamespace(workflowName string, namespace string) bool {
	cmd := exec.Command("kubectl", "get", "workflow", workflowName, "-n", namespace, "-o", "jsonpath={.status.conditions[?(@.type=='Running')].status}")
	response, err := utils.Run(cmd)
	if err != nil {
		GinkgoWriter.Println(fmt.Errorf("failed to check if greeting workflow is running: %v", err))
		return false
	}
	GinkgoWriter.Println(fmt.Sprintf("Got response %s", response))

	if len(strings.TrimSpace(string(response))) == 0 {
		GinkgoWriter.Println(fmt.Errorf("empty response %v", err))
		return false
	}
	status, err := strconv.ParseBool(string(response))
	if err != nil {
		GinkgoWriter.Println(fmt.Errorf("failed to parse result %v", err))
		return false
	}
	return status
}

func verifyWorkflowIsInRunningState(workflowName string) bool {
	return verifyWorkflowIsInRunningStateInNamespace(workflowName, sonataflow_operator_namespace)
}

func verifyWorkflowIsAddressable(workflowName string) bool {
	cmd := exec.Command("kubectl", "get", "workflow", workflowName, "-n", sonataflow_operator_namespace, "-o", "jsonpath={.status.address.url}")
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
