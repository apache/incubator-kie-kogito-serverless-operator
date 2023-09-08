// Copyright 2022 Red Hat, Inc. and/or its affiliates
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
	"os/exec"
	"path/filepath"
	"time"

	"github.com/kiegroup/kogito-serverless-operator/test"
	"github.com/kiegroup/kogito-serverless-operator/test/utils"

	//nolint:golint
	//nolint:revive
	. "github.com/onsi/ginkgo/v2"

	//nolint:golint
	//nolint:revive
	. "github.com/onsi/gomega"
)

var _ = Describe("SonataFlow Operator - no webhooks", Serial, func() {

	Describe("ensure that Operator and Operand(s) can run in restricted namespaces - no webhooks", Ordered, func() {
		It("Prepare the environment - no webhooks", func() {
			var controllerPodName string
			operatorImageName, err := utils.GetOperatorImageName()
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			By("installing CRDs")
			cmd := exec.Command("make", "install-no-webhooks")
			_, err = utils.Run(cmd)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			By("deploying the controller-manager")
			cmd = exec.Command("make", "deploy-no-webhooks", fmt.Sprintf("IMG=%s", operatorImageName))

			outputMake, err := utils.Run(cmd)
			fmt.Println(string(outputMake))
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			By("validating that the controller-manager pod is running as expected - no webhooks")
			verifyControllerUp := func() error {
				var podOutput []byte
				var err error

				if utils.IsDebugEnabled() {
					err = utils.OutputAllPods()
					err = utils.OutputAllEvents(namespace)
				}

				// Get pod name
				cmd = exec.Command("kubectl", "get",
					"pods", "-l", "control-plane=controller-manager",
					"-o", "go-template={{ range .items }}{{ if not .metadata.deletionTimestamp }}{{ .metadata.name }}"+
						"{{ \"\\n\" }}{{ end }}{{ end }}",
					"-n", namespace,
				)
				podOutput, err = utils.Run(cmd)
				fmt.Println(string(podOutput))
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				podNames := utils.GetNonEmptyLines(string(podOutput))
				if len(podNames) != 1 {
					return fmt.Errorf("expect 1 controller pods running, but got %d", len(podNames))
				}
				controllerPodName = podNames[0]
				ExpectWithOffset(2, controllerPodName).Should(ContainSubstring("controller-manager"))

				// Validate pod status
				cmd = exec.Command("kubectl", "get",
					"pods", controllerPodName, "-o", "jsonpath={.status.phase}",
					"-n", namespace,
				)
				status, err := utils.Run(cmd)
				fmt.Println(string(status))
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				if string(status) != "Running" {
					return fmt.Errorf("controller pod in %s status", status)
				}
				return nil
			}
			EventuallyWithOffset(1, verifyControllerUp, time.Minute, time.Second).Should(Succeed())
		})

		// ============= SONATAFLOW OPERATOR TESTS =============

		projectDir, _ := utils.GetProjectDir()

		It("should create a basic platform for Minikube - no webhooks", func() {
			By("creating an instance of the SonataFlowPlatform")
			EventuallyWithOffset(1, func() error {
				cmd := exec.Command("kubectl", "apply", "-f", filepath.Join(projectDir,
					getSonataFlowPlatformFilename()), "-n", namespace)
				_, err := utils.Run(cmd)
				return err
			}, time.Minute, time.Second).Should(Succeed())
		})

		It("should successfully deploy the Greeting Workflow in prod mode and verify if it's running - no webhooks", func() {
			By("creating external resources DataInputSchema configMap - no webhooks")
			EventuallyWithOffset(1, func() error {
				cmd := exec.Command("kubectl", "apply", "-f", filepath.Join(projectDir,
					"test/testdata/"+test.SonataFlowGreetingsDataInputSchemaConfig), "-n", namespace)
				_, err := utils.Run(cmd)
				return err
			}, time.Minute, time.Second).Should(Succeed())

			By("creating an instance of the SonataFlow Operand(CR) - no webhooks")
			EventuallyWithOffset(1, func() error {
				cmd := exec.Command("kubectl", "apply", "-f", filepath.Join(projectDir,
					"test/testdata/"+test.SonataFlowGreetingsWithDataInputSchemaCR), "-n", namespace)
				_, err := utils.Run(cmd)
				return err
			}, time.Minute, time.Second).Should(Succeed())

			By("check the workflow is in running state - no webhooks")
			EventuallyWithOffset(1, func() bool { return verifyWorkflowIsInRunningState("greeting") }, 15*time.Minute, 30*time.Second).Should(BeTrue())

			EventuallyWithOffset(1, func() error {
				cmd := exec.Command("kubectl", "delete", "-f", filepath.Join(projectDir,
					"test/testdata/"+test.SonataFlowGreetingsWithDataInputSchemaCR), "-n", namespace)
				_, err := utils.Run(cmd)
				return err
			}, time.Minute, time.Second).Should(Succeed())
		})

		It("should successfully deploy the orderprocessing workflow in devmode and verify if it's running - no webhooks", func() {

			By("creating an instance of the SonataFlow Workflow in DevMode - no webhooks")
			EventuallyWithOffset(1, func() error {
				cmd := exec.Command("kubectl", "apply", "-f", filepath.Join(projectDir,
					test.GetSonataFlowE2eOrderProcessingFolder()), "-n", namespace)
				_, err := utils.Run(cmd)
				return err
			}, time.Minute, time.Second).Should(Succeed())

			By("check the workflow is in running state - no webhooks")
			EventuallyWithOffset(1, func() bool { return verifyWorkflowIsInRunningState("orderprocessing") }, 15*time.Minute, 30*time.Second).Should(BeTrue())

			cmdLog := exec.Command("kubectl", "logs", "orderprocessing", "-n", namespace)
			if responseLog, errLog := utils.Run(cmdLog); errLog == nil {
				GinkgoWriter.Println(fmt.Sprintf("devmode podlog %s", responseLog))
			}

			By("check that the workflow is addressable - no webhooks")
			EventuallyWithOffset(1, func() bool { return verifyWorkflowIsAddressable("orderprocessing") }, 5*time.Minute, 30*time.Second).Should(BeTrue())

			EventuallyWithOffset(1, func() error {
				cmd := exec.Command("kubectl", "delete", "-f", filepath.Join(projectDir,
					test.GetSonataFlowE2eOrderProcessingFolder()), "-n", namespace)
				_, err := utils.Run(cmd)
				return err
			}, time.Minute, time.Second).Should(Succeed())
		})

		AfterAll(func() {
			By("removing manager namespace - no webhooks")
			cmd := exec.Command("make", "undeploy")
			_, _ = utils.Run(cmd)
			By("uninstalling CRDs")
			cmd = exec.Command("make", "uninstall")
			_, _ = utils.Run(cmd)
		})
	})
})
