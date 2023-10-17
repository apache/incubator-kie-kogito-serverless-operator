package kubernetes

import (
	"context"
	"fmt"
	"github.com/kiegroup/kogito-serverless-operator/controllers/catalog/api"
	"github.com/kiegroup/kogito-serverless-operator/controllers/catalog/utils"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ServiceKind = "services"
	PodKind     = "pods"
)

type K8SServiceCatalog struct {
	Context context.Context
	Client  client.Client
}

func New(ctx context.Context, cli client.Client) K8SServiceCatalog {
	return K8SServiceCatalog{
		Context: ctx,
		Client:  cli,
	}
}

func (c K8SServiceCatalog) Query(uri api.ResourceUri, outputFormat string) (string, error) {
	switch uri.GVK.Kind {
	case ServiceKind:
		return c.resolveServiceQuery(uri, outputFormat)
	case PodKind:
		return c.resolvePodQuery(uri, outputFormat)
	default:
		return "", fmt.Errorf("resolution of kind: %s is not yet implemented", uri.GVK.Kind)
	}
}

func (c K8SServiceCatalog) resolveServiceQuery(uri api.ResourceUri, outputFormat string) (string, error) {
	if service, err := FindService(c.Context, c.Client, uri.Namespace, uri.Name); err != nil {
		return "", err
	} else if serviceUri, err2 := utils.ResolveServiceUri(*service, uri.GetCustomPort(), outputFormat); err2 != nil {
		return "", err2
	} else {
		return serviceUri, nil
	}
}

func (c K8SServiceCatalog) resolvePodQuery(uri api.ResourceUri, outputFormat string) (string, error) {
	if pod, service, err := FindPodAndReferenceServiceByPodLabels(c.Context, c.Client, uri.Namespace, uri.Name); err != nil {
		return "", err
	} else {
		if service != nil {
			if serviceUri, err2 := utils.ResolveServiceUri(*service, uri.GetCustomPort(), outputFormat); err2 != nil {
				return "", err2
			} else {
				return serviceUri, nil
			}
		} else {
			if podUri, err3 := utils.ResolvePodUri(pod, "", uri.GetCustomPort(), outputFormat); err3 != nil {
				return "", err3
			} else {
				return podUri, nil
			}
		}
	}
}
