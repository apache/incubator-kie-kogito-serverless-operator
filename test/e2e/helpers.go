package e2e

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/apache/incubator-kie-kogito-serverless-operator/test"
	"github.com/apache/incubator-kie-kogito-serverless-operator/test/utils"

	//nolint:golint
	//nolint:revive
	. "github.com/onsi/ginkgo/v2"

	//nolint:golint
	//nolint:revive
	. "github.com/onsi/gomega"
)

type health struct {
	Status string `json:"status"`
}

var (
	upStatus string = "UP"
)

func verifyHealthStatusInPod(name string, namespace string) {
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
			Expect(h.Status).To(Equal(upStatus))
			return
		}
		if len(errs.Error()) > 0 {
			errs = fmt.Errorf("%v; %w", err, errs)
		} else {
			errs = err
		}
	}
	Expect(errs).NotTo(HaveOccurred(), fmt.Sprintf("No container was found that could respond to the health endpoint %v", errs))

}

func getHealthStatusInContainer(podName string, containerName string, ns string) (*health, error) {
	h := health{}
	cmd := exec.Command("kubectl", "exec", "-t", podName, "-n", ns, "-c", containerName, "--", "curl", "-s", "localhost:8080/q/health")
	output, err := utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred())
	err = json.Unmarshal(output, &h)
	if err != nil {
		return nil, fmt.Errorf("failed to execute curl command against health endpoint in container %s:%v with output %s", containerName, err, output)
	}
	return &h, nil
}
func verifyWorkflowIsInRunningStateInNamespace(workflowName string, ns string) bool {
	cmd := exec.Command("kubectl", "get", "workflow", workflowName, "-n", ns, "-o", "jsonpath={.status.conditions[?(@.type=='Running')].status}")
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

func verifyWorkflowIsInRunningState(workflowName string, targetNamespace string) bool {
	return verifyWorkflowIsInRunningStateInNamespace(workflowName, targetNamespace)
}

func verifyWorkflowIsAddressable(workflowName string, targetNamespace string) bool {
	cmd := exec.Command("kubectl", "get", "workflow", workflowName, "-n", targetNamespace, "-o", "jsonpath={.status.address.url}")
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

const (
	minikubePlatform  = "minikube"
	openshiftPlatform = "openshift"
)

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
