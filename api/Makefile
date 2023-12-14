ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

## Tool binaries
CLIENT_GEN ?= $(GOBIN)/client-gen
CONVERSION_GEN ?= $(GOBIN)/conversion-gen

## Tool versions
CLIENT_GEN_VERSION ?= v0.27.3
CONVERSION_GEN_VERSION ?= v0.27.3

.PHONY: client-gen
client-gen: $(CLIENT_GEN) ## Download client-gen locally if necessary.
$(CLIENT_GEN): $(GOBIN)
	test -s $(GOBIN)/client-gen || GO111MODULE=on go install k8s.io/code-generator/cmd/client-gen@$(CLIENT_GEN_VERSION)

.PHONY: conversion-gen
conversion-gen: $(CONVERSION_GEN)
$(CONVERSION_GEN): $(GOBIN)
	test -s $(GOBIN)/conversion-gen || GO111MODULE=on go install k8s.io/code-generator/cmd/conversion-gen@$(CONVERSION_GEN_VERSION)

.PHONY: generate
generate: client-gen conversion-gen ## Generate code
	$(CLIENT_GEN) --go-header-file="../hack/boilerplate.go.txt" --clientset-name="clientset" --input-base="github.com/apache/incubator-kie-kogito-serverless-operator/api" --input="sonataflow/v1alpha08" --output-package="github.com/apache/incubator-kie-kogito-serverless-operator/api/generated"
	#$(CONVERSION_GEN) --go-header-file="../hack/boilerplate.go.txt" -O zz_generated.conversion --output-base="github.com/apache/incubator-kie-kogito-serverless-operator/api" --input-dirs="github.com/apache/incubator-kie-kogito-serverless-operator/api"

.PHONY: test
test:
	go test $(shell go list ./... | grep -v /test/) -coverprofile cover.out
