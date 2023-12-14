ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

## Tool binaries
CLIENT_GEN ?= $(GOBIN)/client-gen

## Tool versions
CLIENT_TOOLS_VERSION ?= v0.27.3

## Default Image Name
DEFAULT_APISERVER_IMAGE_NAME ?= dev.local/sonataflow-workflowproj-apiserver
DEFAULT_APISERVER_IMAGE_TAG	 ?= latest

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

.PHONY: build-image
image_name?=$(DEFAULT_APISERVER_IMAGE_NAME)
image_tag?=$(DEFAULT_APISERVER_IMAGE_TAG)
build-image:
	@echo "Building API Server binaries"
	@export GOOS=linux; go build .
	@mv workflowproj-apiserver ./artifacts/simple-image/
	@echo "Building API Server Image ${image_name}:${image_tag}"
	@docker build -t ${image_name}:${image_tag} ./artifacts/simple-image

.PHONY: kustomize-overrides
image_name?=$(DEFAULT_APISERVER_IMAGE_NAME)
image_tag?=$(DEFAULT_APISERVER_IMAGE_TAG)
kustomize-overrides:
	@echo "Overriding controller image to ${image_name}:${image_tag}"
	@cd artifacts/examples/base && kustomize edit set image workflowproj-apiserver=${image_name}:${image_tag} && cd - > /dev/null
	@cd artifacts/examples/overlays/openshift && kustomize edit set image workflowproj-apiserver=${image_name}:${image_tag} && cd - > /dev/null

.PHONY: deploy-example-minikube
image_name?=$(DEFAULT_APISERVER_IMAGE_NAME)
image_tag?=$(DEFAULT_APISERVER_IMAGE_TAG)
deploy-example-minikube: build-image kustomize-overrides
	@echo "Adding image ${image_name}:${image_tag} to Minikube registry"
	@minikube image load ${image_name}:${image_tag}
	@echo "Deploying API Server"
	@kubectl apply -k artifacts/examples/base

.PHONY: undeploy-example-minikube
image_name?=$(DEFAULT_APISERVER_IMAGE_NAME)
image_tag?=$(DEFAULT_APISERVER_IMAGE_TAG)
undeploy-example-minikube:
	@echo "Deleting Kubernetes Objects"
	@kubectl delete -k artifacts/examples/base --ignore-not-found=true
	@echo "Removing image ${image_name}:${image_tag} from Minikube registry"
	@minikube image rm ${image_name}:${image_tag}

.PHONY: run-operator-locally
run-operator-locally:
	@cd .. && ./hack/local/run-operator.sh
