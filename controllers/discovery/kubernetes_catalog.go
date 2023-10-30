package discovery

import (
	"context"
	"fmt"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	serviceKind = "services"
	podKind     = "pods"
)

type k8sServiceCatalog struct {
	Context context.Context
	Client  client.Client
}

func newK8SServiceCatalog(ctx context.Context, cli client.Client) k8sServiceCatalog {
	return k8sServiceCatalog{
		Context: ctx,
		Client:  cli,
	}
}

func (c k8sServiceCatalog) Query(uri ResourceUri, outputFormat string) (string, error) {
	switch uri.GVK.Kind {
	case serviceKind:
		return c.resolveServiceQuery(uri, outputFormat)
	case podKind:
		return c.resolvePodQuery(uri, outputFormat)
	default:
		return "", fmt.Errorf("resolution of kind: %s is not yet implemented", uri.GVK.Kind)
	}
}

func (c k8sServiceCatalog) resolveServiceQuery(uri ResourceUri, outputFormat string) (string, error) {
	if service, err := findService(c.Context, c.Client, uri.Namespace, uri.Name); err != nil {
		return "", err
	} else if serviceUri, err2 := resolveServiceUri(*service, uri.GetCustomPort(), outputFormat); err2 != nil {
		return "", err2
	} else {
		return serviceUri, nil
	}
}

func (c k8sServiceCatalog) resolvePodQuery(uri ResourceUri, outputFormat string) (string, error) {
	if pod, service, err := findPodAndReferenceServiceByPodLabels(c.Context, c.Client, uri.Namespace, uri.Name); err != nil {
		return "", err
	} else {
		if service != nil {
			if serviceUri, err2 := resolveServiceUri(*service, uri.GetCustomPort(), outputFormat); err2 != nil {
				return "", err2
			} else {
				return serviceUri, nil
			}
		} else {
			if podUri, err3 := resolvePodUri(pod, "", uri.GetCustomPort(), outputFormat); err3 != nil {
				return "", err3
			} else {
				return podUri, nil
			}
		}
	}
}
