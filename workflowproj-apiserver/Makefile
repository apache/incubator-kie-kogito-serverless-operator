ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

## Tool binaries
CLIENT_GEN ?= $(GOBIN)/client-gen

## Tool versions
CLIENT_TOOLS_VERSION ?= v0.27.3

.PHONY: client-gen
client-gen: $(CLIENT_GEN) ## Download client-gen locally if necessary.
$(CLIENT_GEN): $(GOBIN)
	test -s $(GOBIN)/client-gen || GO111MODULE=on go install k8s.io/code-generator/cmd/client-gen@$(CLIENT_GEN_VERSION)

.PHONY: generate
generate: client-gen ## Generate code
	$(CLIENT_GEN) --go-header-file="hack/boilerplate.go.txt" --clientset-name="clientset" --input-base="github.com/apache/incubator-kie-kogito-serverless-operator/workflowproj-apiserver/pkg/apis" --input="workflowproj/v1alpha08" --output-package="github.com/apache/incubator-kie-kogito-serverless-operator/workflowproj-apiserver/pkg/generated"

.PHONY: test
test:
	go test $(shell go list ./... | grep -v /test/) -coverprofile cover.out
