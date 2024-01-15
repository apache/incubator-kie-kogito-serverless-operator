// Copyright 2024 Apache Software Foundation (ASF)
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

package e2e

import (
	"bytes"
	"fmt"
	"math/rand"
	"os/exec"
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
		for _, pn := range strings.Split(string(output), " ") {
			verifyHealthStatusInPod(pn, ns)
			verifyDatabaseConnectionsHealthStatusInPod(pn, ns)
		}
	},
		Entry("defined in the workflow from an existing kubernetes service as a reference", test.GetSonataFlowE2EWorkflowPersistenceSampleDataDirectory("by_service")),
	)

})

func verifyDatabaseConnectionsHealthStatusInPod(name string, namespace string) {
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
			for _, c := range h.Checks {
				if c.Name == "Database connections health check" {
					Expect(c.Status).To(Equal(upStatus))
					return
				}
			}
			errs = fmt.Errorf("no database connection health status found; %w", errs)
		}
		errs = fmt.Errorf("%v; %w", err, errs)
	}
	Expect(errs).NotTo(HaveOccurred(), fmt.Sprintf("No container was found that could respond to the health endpoint %v", errs))
}
