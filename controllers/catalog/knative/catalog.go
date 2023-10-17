package knative

import (
	"context"
	"fmt"
	"github.com/kiegroup/kogito-serverless-operator/controllers/catalog/api"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type KnServiceCatalog struct {
	Context context.Context
	Client  client.Client
}

func New(ctx context.Context, cli client.Client) KnServiceCatalog {
	return KnServiceCatalog{
		Context: ctx,
		Client:  cli,
	}
}

func (c KnServiceCatalog) Query(uri api.ResourceUri, outputFormat string) (string, error) {
	return "", fmt.Errorf("knative service discovery is not yet implemened")
}
