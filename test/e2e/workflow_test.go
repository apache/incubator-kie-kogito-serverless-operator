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
	"fmt"
	"math/rand"
	"os/exec"
	"path/filepath"
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

const (
	workflowAppLabel = "sonataflow.org/workflow-app"
)

var _ = Describe("Workflow Non-Persistence Use Cases :: ", Label("flows-ephemeral"), Ordered, func() {

	var targetNamespace string
	BeforeEach(func() {
		targetNamespace = fmt.Sprintf("test-%d", rand.Intn(randomIntRange)+1)
		err := kubectlCreateNamespace(targetNamespace)
		Expect(err).NotTo(HaveOccurred())
	})
	AfterEach(func() {
		// Remove resources in test namespace
		if !CurrentSpecReport().Failed() && len(targetNamespace) > 0 {
			cmd := exec.Command("kubectl", "delete", "sonataflow", "--all", "-n", targetNamespace, "--wait")
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())
			err = kubectlDeleteNamespace(targetNamespace)
			Expect(err).NotTo(HaveOccurred())
		}
	})

	Describe("Ensure basic workflow deployments", func() {

		It("should successfully deploy the Simple Workflow in GitOps profile and verify if it's running", func() {
			By("creating an instance of the SonataFlow Operand(CR)")
			sonataFlowCrFile := test.GetPathFromE2EDirectory("workflows", prebuiltWorkflows.Greetings.Name, "01-sonataflow.org_v1alpha08_sonataflow.yaml")
			EventuallyWithOffset(1, func() error {
				return kubectlApplyFileOnCluster(sonataFlowCrFile, targetNamespace)
			}, 3*time.Minute, time.Second).Should(Succeed())

			By("Replacing the image with a prebuilt one and rollout")
			EventuallyWithOffset(1, func() error {
				return kubectlPatchSonataFlowImageAndRollout(targetNamespace, prebuiltWorkflows.Greetings.Name, prebuiltWorkflows.Greetings.Tag)
			}, 3*time.Minute, time.Second).Should(Succeed())

			By("check the workflow is in running state")
			EventuallyWithOffset(1, func() bool { return verifyWorkflowIsInRunningState(prebuiltWorkflows.Greetings.Name, targetNamespace) }, 15*time.Minute, 30*time.Second).Should(BeTrue())

			EventuallyWithOffset(1, func() error {
				return kubectlDeleteFileOnCluster(sonataFlowCrFile, targetNamespace)
			}, 3*time.Minute, time.Second).Should(Succeed())
		})

		It("should successfully deploy the Greeting Workflow in preview profile and verify if it's running", func() {
			By("creating external resources DataInputSchema configMap")
			EventuallyWithOffset(1, func() error {
				cmd := exec.Command("kubectl", "apply", "-f", test.GetPathFromDataDirectory(test.SonataFlowGreetingsDataInputSchemaConfig), "-n", targetNamespace)
				_, err := utils.Run(cmd)
				return err
			}, 3*time.Minute, time.Second).Should(Succeed())

			sonataFlowYaml := test.GetPathFromDataDirectory(test.SonataFlowGreetingsWithDataInputSchemaCR)
			By("creating an instance of the SonataFlow Operand(CR)")
			EventuallyWithOffset(1, func() error {
				cmd := exec.Command("kubectl", "apply", "-f", sonataFlowYaml, "-n", targetNamespace)
				_, err := utils.Run(cmd)
				return err
			}, 3*time.Minute, time.Second).Should(Succeed())

			By("check the workflow is in running state")
			EventuallyWithOffset(1, func() bool { return verifyWorkflowIsInRunningState("greeting", targetNamespace) }, 15*time.Minute, 30*time.Second).Should(BeTrue())

			EventuallyWithOffset(1, func() error {
				cmd := exec.Command("kubectl", "delete", "-f", sonataFlowYaml, "-n", targetNamespace)
				_, err := utils.Run(cmd)
				return err
			}, 3*time.Minute, time.Second).Should(Succeed())
		})

		It("should successfully deploy the orderprocessing workflow in devmode and verify if it's running", func() {
			orderProcessingFolder := test.GetPathFromE2EDirectory("order-processing")
			By("creating an instance of the SonataFlow Workflow in DevMode")
			EventuallyWithOffset(1, func() error {
				cmd := exec.Command("kubectl", "apply", "-f", orderProcessingFolder, "-n", targetNamespace)
				_, err := utils.Run(cmd)
				return err
			}, 3*time.Minute, time.Second).Should(Succeed())

			By("check the workflow is in running state")
			EventuallyWithOffset(1, func() bool { return verifyWorkflowIsInRunningState("orderprocessing", targetNamespace) }, 10*time.Minute, 30*time.Second).Should(BeTrue())

			cmdLog := exec.Command("kubectl", "logs", "orderprocessing", "-n", targetNamespace)
			if responseLog, errLog := utils.Run(cmdLog); errLog == nil {
				GinkgoWriter.Println(fmt.Sprintf("devmode podlog %s", responseLog))
			}

			By("check that the workflow is addressable")
			EventuallyWithOffset(1, func() bool { return verifyWorkflowIsAddressable("orderprocessing", targetNamespace) }, 10*time.Minute, 30*time.Second).Should(BeTrue())

			EventuallyWithOffset(1, func() error {
				cmd := exec.Command("kubectl", "delete", "-f", orderProcessingFolder, "-n", targetNamespace)
				_, err := utils.Run(cmd)
				return err
			}, 3*time.Minute, time.Second).Should(Succeed())
		})

	})
})

var _ = Describe("Workflow Persistence Use Cases :: ", Label("flows-persistence"), Ordered, func() {

	const (
		dbConnectionName = "Database connections health check"
		defaultDataCheck = "<default>"
	)
	var (
		ns string
	)

	BeforeEach(func() {
		ns = fmt.Sprintf("test-%d", rand.Intn(randomIntRange)+1)
		cmd := exec.Command("kubectl", "create", "namespace", ns)
		_, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred())
	})
	AfterEach(func() {
		if len(ns) > 0 {
			cmd := exec.Command("kubectl", "delete", "sonataflow", "--all", "-n", ns, "--wait")
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())
			cmd = exec.Command("kubectl", "delete", "namespace", ns, "--wait")
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())
		}

	})

	DescribeTable("when deploying a SonataFlow CR with PostgreSQL persistence", func(testcaseDir string, withPersistence bool, waitKSinkInjection bool) {
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

		By("Replacing the image with a prebuilt one and rollout")
		EventuallyWithOffset(1, func() error {
			if withPersistence {
				return kubectlPatchSonataFlowImageAndRollout(ns, prebuiltWorkflows.CallBackPersistence.Name, prebuiltWorkflows.CallBackPersistence.Tag)
			}
			return kubectlPatchSonataFlowImageAndRollout(ns, prebuiltWorkflows.CallBack.Name, prebuiltWorkflows.CallBack.Tag)
		}, 3*time.Minute, time.Second).Should(Succeed())

		By("Wait for SonataFlow CR to complete deployment")
		// wait for service deployments to be ready
		EventuallyWithOffset(1, func() bool {
			cmd = exec.Command("kubectl", "wait", "pod", "-n", ns, "-l", workflowAppLabel, "--for", "condition=Ready", "--timeout=5s")
			out, err := utils.Run(cmd)
			if err != nil {
				return false
			}
			GinkgoWriter.Printf("%s\n", string(out))
			if !waitKSinkInjection {
				return true
			}
			GinkgoWriter.Println("waitForPodRestartCompletion")
			waitForPodRestartCompletion(workflowAppLabel, ns)
			GinkgoWriter.Println("waitForPodRestartCompletion done")
			return true
		}, 5*time.Minute, 5*time.Second).Should(BeTrue())

		By("Evaluate status of the workflow's pod database connection health endpoint")
		cmd = exec.Command("kubectl", "get", "pod", "-l", workflowAppLabel, "-n", ns, "-ojsonpath={.items[*].metadata.name}")
		output, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred())
		EventuallyWithOffset(1, func() bool {
			for _, pn := range strings.Split(string(output), " ") {
				h, err := getHealthFromPod(pn, ns)
				if err != nil {
					continue
				}
				Expect(h.Status).To(Equal(upStatus), "Pod health is not UP")
				if withPersistence {
					connectionCheckFound := false
					for _, c := range h.Checks {
						if c.Name == dbConnectionName {
							Expect(c.Status).To(Equal(upStatus), "Pod's database connection is not UP")
							Expect(c.Data[defaultDataCheck]).To(Equal(upStatus), "Pod's 'default' database data is not UP")
							connectionCheckFound = true
						}
					}
					Expect(connectionCheckFound).To(Equal(true), "Connection health check not found, but the wofkflow has persistence")
					return true
				} else {
					connectionCheckFound := false
					for _, c := range h.Checks {
						if c.Name == dbConnectionName {
							connectionCheckFound = true
						}
					}
					Expect(connectionCheckFound).To(Equal(false), "Connection health check was found, but the workflow don't have persistence")
					return true
				}
			}
			return false
		}, 4*time.Minute).Should(BeTrue())
		// Persistence initialization checks
		cmd = exec.Command("kubectl", "get", "pod", "-l", workflowAppLabel, "-n", ns, "-ojsonpath={.items[*].metadata.name}")
		output, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred())
		podName := string(output)
		cmd = exec.Command("kubectl", "logs", podName, "-n", ns)
		output, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred())
		logs := string(output)
		if withPersistence {
			By("Validate that the workflow persistence was properly initialized")
			Expect(logs).Should(ContainSubstring("Flyway Community Edition"))
			Expect(logs).Should(ContainSubstring("Database: jdbc:postgresql://postgres.%s:5432", ns))
			result := verifySchemaMigration(logs, prebuiltWorkflows.CallBackPersistence.Name)
			GinkgoWriter.Println(fmt.Sprintf("verifySchemaMigration: %v", result))
			Expect(result).Should(BeTrue())
			Expect(logs).Should(ContainSubstring("Profile prod activated"))
		} else {
			By("Validate that the workflow has no persistence")
			Expect(logs).ShouldNot(ContainSubstring("Flyway Community Edition"))
			Expect(logs).ShouldNot(ContainSubstring(fmt.Sprintf(`Creating schema "%s"`, prebuiltWorkflows.CallBack.Name)))
			Expect(logs).Should(ContainSubstring("Profile prod activated"))
		}
	},
		Entry("defined in the workflow from an existing kubernetes service as a reference", test.GetPathFromE2EDirectory("workflows", "persistence", "by_service"), true, false),
		Entry("defined in the workflow and from the sonataflow platform", test.GetPathFromE2EDirectory("workflows", "persistence", "from_platform_overwritten_by_service"), true, false),
		Entry("defined from the sonataflow platform as reference and with DI and JS", test.GetPathFromE2EDirectory("workflows", "persistence", "from_platform_with_di_and_js_services"), true, true),
		Entry("defined from the sonataflow platform as reference and without DI and JS", test.GetPathFromE2EDirectory("workflows", "persistence", "from_platform_without_di_and_js_services"), true, false),
		Entry("defined from the sonataflow platform as reference but not required by the workflow", test.GetPathFromE2EDirectory("workflows", "persistence", "from_platform_with_no_persistence_required"), false, false),
	)

	Describe("basic workflow monitoring", func() {
		projectDir, _ := utils.GetProjectDir()

		It("should create servicemonitor for workflow deployed as k8s deployment when monitoring enabled in platform", func() {
			By("creating external resources DataInputSchema configMap")
			By("Deploy the SonataFlowPlatform CR")
			var manifests []byte
			EventuallyWithOffset(1, func() error {
				var err error
				cmd := exec.Command("kubectl", "kustomize", filepath.Join(projectDir,
					"test/e2e/testdata/workflows/prometheus", "k8s_deployment"))
				manifests, err = utils.Run(cmd)
				return err
			}, time.Minute, time.Second).Should(Succeed())
			cmd := exec.Command("kubectl", "create", "-n", ns, "-f", "-")
			cmd.Stdin = bytes.NewBuffer(manifests)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())
			workflowName := "callbackstatetimeouts"
			By("check the workflow is in running state")
			EventuallyWithOffset(1, func() bool { return verifyWorkflowIsInRunningState(workflowName, ns) }, 15*time.Minute, 30*time.Second).Should(BeTrue())

			By("Retrieve the name of the running pod for the workflow")
			labels := fmt.Sprintf("sonataflow.org/workflow-app=%s,sonataflow.org/workflow-namespace=%s", workflowName, ns)
			podName := waitForPodRestartCompletion(labels, ns)

			By("check service monitor has been created")
			EventuallyWithOffset(1, func() bool {
				cmd := exec.Command("kubectl", "get", "servicemonitor", workflowName, "-n", ns)
				_, err := utils.Run(cmd)
				if err != nil {
					GinkgoWriter.Println(fmt.Errorf("failed to get servicemonitor: %v", err))
					return false
				}
				return true
			}, 1*time.Minute, 5).Should(BeTrue())

			By("trigger a new workflow instance")
			var workflowId string
			EventuallyWithOffset(1, func() bool {
				curlCmd := fmt.Sprintf("curl -X POST -H 'Content-Type: application/json' -H 'Accept: application/json' http://%s/%s", workflowName, workflowName)
				cmd := exec.Command("kubectl", "exec", podName, "-c", "workflow", "-n", ns, "--", "/bin/bash", "-c", curlCmd)
				resp, err := utils.Run(cmd)
				if err != nil {
					GinkgoWriter.Println(fmt.Errorf("failed to trigger workflow instance: %v", err))
					return false
				}
				GinkgoWriter.Println(fmt.Errorf("Response: %v", string(resp)))
				if id, err := getWorkflowId(string(resp)); err != nil {
					GinkgoWriter.Println(err)
					return false
				} else {
					workflowId = id
					GinkgoWriter.Println("Workflow id found:", id)
					return true
				}
			}, 2*time.Minute, 5).Should(BeTrue())

			By("check the new workflow instance has finished")
			checkStr := fmt.Sprintf("callback-state-timeouts: %s has finalized", workflowId)
			EventuallyWithOffset(1, func() bool {
				cmdLog := exec.Command("kubectl", "logs", podName, "-c", "workflow", "-n", ns)
				responseLog, err := utils.Run(cmdLog)
				if err == nil {
					if strings.Contains(string(responseLog), checkStr) {
						return true
					}
					GinkgoWriter.Println(fmt.Errorf("did not find matched string in the log"))

				} else {
					GinkgoWriter.Println(fmt.Errorf("failed to get logs from workflow pod: %v", err))
				}
				return false
			}, 1*time.Minute, 5).Should(BeTrue())

			By("check prometheus server has workflow instance metrics")
			EventuallyWithOffset(1, func() bool {
				curlCmd := fmt.Sprintf("curl http://prometheus-operated.default:9090/api/v1/query --data-urlencode 'query=kogito_process_instance_duration_seconds_count{job=\"%s\",namespace=\"%s\"}'", workflowName, ns)
				GinkgoWriter.Println(curlCmd)
				cmd := exec.Command("kubectl", "exec", podName, "-c", "workflow", "-n", ns, "--", "/bin/bash", "-c", curlCmd)
				resp, err := utils.Run(cmd)
				if err != nil {
					GinkgoWriter.Println(fmt.Errorf("failed to get metrics from prometheus server: %v", err))
					return false
				}
				if val, err := getMetricValue(string(resp)); err != nil {
					GinkgoWriter.Println(err)
					return false
				} else {
					GinkgoWriter.Println("metric value found:", val)
					return val == "1"
				}
			}, 1*time.Minute, 5).Should(BeTrue())
		})

		It("should create servicemonitor for workflow deployed as Knative service when monitoring enabled in platform", func() {
			By("creating external resources DataInputSchema configMap")
			By("Deploy the SonataFlowPlatform CR")
			var manifests []byte
			EventuallyWithOffset(1, func() error {
				var err error
				cmd := exec.Command("kubectl", "kustomize", filepath.Join(projectDir,
					"test/e2e/testdata/workflows/prometheus", "knative_service"))
				manifests, err = utils.Run(cmd)
				return err
			}, time.Minute, time.Second).Should(Succeed())
			cmd := exec.Command("kubectl", "create", "-n", ns, "-f", "-")
			cmd.Stdin = bytes.NewBuffer(manifests)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())
			workflowName := "callbackstatetimeouts"
			By("check the workflow is in running state")
			EventuallyWithOffset(1, func() bool { return verifyWorkflowIsInRunningState(workflowName, ns) }, 15*time.Minute, 30*time.Second).Should(BeTrue())

			By("Retrieve the name of the running pod for data index service")
			labels := "sonataflow.org/service=sonataflow-platform-data-index-service"
			podName := waitForPodRestartCompletion(labels, ns)

			By("check service monitor has been created")
			EventuallyWithOffset(1, func() bool {
				cmd := exec.Command("kubectl", "get", "servicemonitor", workflowName, "-n", ns)
				_, err := utils.Run(cmd)
				if err != nil {
					GinkgoWriter.Println(fmt.Errorf("failed to get servicemonitor: %v", err))
					return false
				}
				return true
			}, 1*time.Minute, 5).Should(BeTrue())

			By("trigger a new workflow instance")
			var workflowId string
			EventuallyWithOffset(1, func() bool {
				// To trigger workflow instances with Knative service, we need to use the full service name
				curlCmd := fmt.Sprintf("curl -X POST -H 'Content-Type: application/json' -H 'Accept: application/json' http://%s.%s.svc.cluster.local/%s", workflowName, ns, workflowName)
				cmd := exec.Command("kubectl", "exec", podName, "-c", "data-index-service", "-n", ns, "--", "/bin/bash", "-c", curlCmd)
				resp, err := utils.Run(cmd)
				if err != nil {
					GinkgoWriter.Println(fmt.Errorf("failed to trigger workflow instance: %v", err))
					return false
				}
				GinkgoWriter.Println(fmt.Errorf("Response: %v", string(resp)))
				if id, err := getWorkflowId(string(resp)); err != nil {
					GinkgoWriter.Println(err)
					return false
				} else {
					workflowId = id
					GinkgoWriter.Println("Workflow instance created:", id)
					return true
				}
			}, 2*time.Minute, 5).Should(BeTrue())

			By("check the workflow pod has started for the new workflow instance")
			checkStr := fmt.Sprintf("callback-state-timeouts: %s has started", workflowId)
			EventuallyWithOffset(1, func() bool {
				name, err := getPodNameAfterWorkflowInstCreation(workflowName, ns)
				if err != nil {
					GinkgoWriter.Println(err)
					return false
				}
				GinkgoWriter.Println("Workflow pod found:", name)
				cmdLog := exec.Command("kubectl", "logs", name, "-c", "workflow", "-n", ns)
				responseLog, err := utils.Run(cmdLog)
				if err == nil {
					if strings.Contains(string(responseLog), checkStr) {
						return true
					}
					GinkgoWriter.Println(fmt.Errorf("did not find matched string in the log"))
				} else {
					GinkgoWriter.Println(fmt.Errorf("failed to get logs from workflow pod: %v", err))
				}
				return false
			}, 5*time.Minute, 5).Should(BeTrue())

			By("check prometheus server has workflow instance metrics")
			EventuallyWithOffset(1, func() bool {
				curlCmd := fmt.Sprintf("curl http://prometheus-operated.default:9090/api/v1/query --data-urlencode 'query=kogito_process_instance_started_total{service=\"%s-metrics\",namespace=\"%s\"}'", workflowName, ns)
				GinkgoWriter.Println(curlCmd)
				cmd := exec.Command("kubectl", "exec", podName, "-c", "data-index-service", "-n", ns, "--", "/bin/bash", "-c", curlCmd)
				resp, err := utils.Run(cmd)
				if err != nil {
					GinkgoWriter.Println(fmt.Errorf("failed to get metrics from prometheus server: %v", err))
					return false
				}
				if val, err := getMetricValue(string(resp)); err != nil {
					GinkgoWriter.Println(err)
					return false
				} else {
					GinkgoWriter.Println("metric value found:", val)
					return val == "1"
				}
			}, 105*time.Minute, 5).Should(BeTrue())
		})
	})
})
