package catalog

import (
	"context"
	"fmt"
	"github.com/kiegroup/kogito-serverless-operator/controllers/catalog/api"
	"github.com/kiegroup/kogito-serverless-operator/controllers/catalog/knative"
	"github.com/kiegroup/kogito-serverless-operator/controllers/catalog/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type sonataFlowServiceCatalog struct {
	kubernetesCatalog api.ServiceCatalog
	knativeCatalog    api.ServiceCatalog
	openshiftCatalog  api.ServiceCatalog
}

// New returns a new ServiceCatalog configured to resolve kubernetes, knative, and openshift resource addresses.
func New(ctx context.Context, cli client.Client) api.ServiceCatalog {
	return &sonataFlowServiceCatalog{
		kubernetesCatalog: kubernetes.New(ctx, cli),
		knativeCatalog:    knative.New(ctx, cli),
	}
}

func (c *sonataFlowServiceCatalog) Query(uri api.ResourceUri, outputFormat string) (string, error) {
	switch uri.Scheme {
	case api.KubernetesScheme:
		return c.kubernetesCatalog.Query(uri, outputFormat)
	case api.KnativeScheme:
		return "", fmt.Errorf("knative service discovery is not yet implemened")
	case api.OpenshiftScheme:
		return "", fmt.Errorf("openshift service discovery is not yet implemented")
	default:
		return "", fmt.Errorf("unknonw scheme was provided for service discovery: %s", uri.Scheme)
	}
}
