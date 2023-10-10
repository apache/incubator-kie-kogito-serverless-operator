.PHONY: test
test:
	KUBEBUILDER_ASSETS=${KUBEBUILDER_ASSETS} go test $(shell go list ./... | grep -v /test/) -coverprofile cover.out
