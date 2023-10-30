package discovery

import (
	"context"
	"fmt"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type knServiceCatalog struct {
	Context context.Context
	Client  client.Client
}

func newKnServiceCatalog(ctx context.Context, cli client.Client) knServiceCatalog {
	return knServiceCatalog{
		Context: ctx,
		Client:  cli,
	}
}

func (c knServiceCatalog) Query(uri ResourceUri, outputFormat string) (string, error) {
	return "", fmt.Errorf("knative service discovery is not yet implemened")
}
