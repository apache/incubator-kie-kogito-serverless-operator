package api

import (
	"fmt"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	KnativeScheme    = "knative"
	KubernetesScheme = "kubernetes"
	OpenshiftScheme  = "openshift"

	// kubernetes groups
	kubernetesServices     = "kubernetes:services.v1"
	kubernetesPods         = "kubernetes:pods.v1"
	kubernetesDeployments  = "kubernetes:deployments.v1.apps"
	kubernetesStatefulSets = "kubernetes:statefulsets.v1.apps"
	kubernetesIngresses    = "kubernetes:ingresses.v1.networking.k8s.io"

	// knative groups
	knativeServices = "knative:services.v1.serving.knative.dev"

	// openshift groups
	openshiftRoutes            = "openshift:routes.v1.route.openshift.io"
	openshiftDeploymentConfigs = "openshift:deploymentconfigs.v1.apps.openshift.io"

	// CustomPortLabel well known label name to select a particular target port
	CustomPortLabel = "custom-port"

	// KubernetesDNSAddress use this output format with kubernetes services and pods to resolve to the corresponding
	// kubernetes DNS name. see: https://kubernetes.io/docs/concepts/services-networking/dns-pod-service/
	KubernetesDNSAddress = "KubernetesDNSAddress"

	// KubernetesIPAddress default format, resolves objects addresses to the corresponding cluster IP address.
	KubernetesIPAddress = "KubernetesIPAddress"
)

type ResourceUri struct {
	Scheme       string
	GVK          v1.GroupVersionKind
	Namespace    string
	Name         string
	CustomLabels map[string]string
}

type ResourceUriBuilder struct {
	uri *ResourceUri
}

func NewResourceUriBuilder(scheme string) ResourceUriBuilder {
	return ResourceUriBuilder{
		uri: &ResourceUri{
			Scheme:       scheme,
			GVK:          v1.GroupVersionKind{},
			CustomLabels: map[string]string{},
		},
	}
}

func (b ResourceUriBuilder) Kind(kind string) ResourceUriBuilder {
	b.uri.GVK.Kind = kind
	return b
}

func (b ResourceUriBuilder) Version(version string) ResourceUriBuilder {
	b.uri.GVK.Version = version
	return b
}

func (b ResourceUriBuilder) Group(group string) ResourceUriBuilder {
	b.uri.GVK.Group = group
	return b
}

func (b ResourceUriBuilder) Namespace(namespace string) ResourceUriBuilder {
	b.uri.Namespace = namespace
	return b
}

func (b ResourceUriBuilder) Name(name string) ResourceUriBuilder {
	b.uri.Name = name
	return b
}

func (b ResourceUriBuilder) CustomPort(customPort string) ResourceUriBuilder {
	b.uri.SetCustomPort(customPort)
	return b
}

func (b ResourceUriBuilder) WithLabel(labelName string, labelValue string) ResourceUriBuilder {
	b.uri.CustomLabels[labelName] = labelValue
	return b
}

func (b ResourceUriBuilder) Build() *ResourceUri {
	return b.uri
}

// ServiceCatalog is the entry point to resolve resource addresses given a ResourceUri.
type ServiceCatalog interface {
	// Query returns the address corresponding to the resource identified by the uri. In the case of services or pods,
	// the outputFormat can be used to determine the type of address to calculate.
	// If the outputFormat is KubernetesDNSAddress, the returned value for a service will be like this: http://my-service.my-namespace.svc.cluster.local:8080,
	// and the returned value for pod will be like this: http://10-244-1-135.my-namespace.pod.cluster.local:8080.
	// If the outputFormat is KubernetesIPAddress, the returned value for pods and services, and other resource types,
	// will be like this: http://10.245.1.132:8080
	Query(uri ResourceUri, outputFormat string) (string, error)
}

func (r *ResourceUri) AddLabel(name string, value string) *ResourceUri {
	if len(value) > 0 {
		r.CustomLabels[name] = value
	}
	return r
}

func (r *ResourceUri) GetLabel(name string) string {
	if len(name) > 0 {
		return r.CustomLabels[name]
	}
	return ""
}

func (r *ResourceUri) SetCustomPort(value string) *ResourceUri {
	return r.AddLabel(CustomPortLabel, value)
}

func (r *ResourceUri) GetCustomPort() string {
	return r.GetLabel(CustomPortLabel)
}

func (r *ResourceUri) String() string {
	if r == nil {
		return ""
	}
	gvk := appendWithDelimiter("", r.GVK.Kind, ".")
	gvk = appendWithDelimiter(gvk, r.GVK.Version, ".")
	gvk = appendWithDelimiter(gvk, r.GVK.Group, ".")
	uri := r.Scheme + ":" + gvk
	uri = appendWithDelimiter(uri, r.Namespace, "/")
	uri = appendWithDelimiter(uri, r.Name, "/")

	return appendWithDelimiter(uri, buildLabelsString(r.CustomLabels, "&"), "?")
}

func appendWithDelimiter(value string, toAppend string, delimiter string) string {
	if len(toAppend) > 0 {
		if len(value) > 0 {
			return fmt.Sprintf("%s%s%s", value, delimiter, toAppend)
		} else {
			return fmt.Sprintf("%s%s", value, toAppend)
		}
	}
	return value
}

func buildParam(name string, value string) string {
	return fmt.Sprintf("%s=%s", name, value)
}

func buildLabelsString(labels map[string]string, delimiter string) string {
	var labelsStr string
	for name, value := range labels {
		labelsStr = appendWithDelimiter(labelsStr, buildParam(name, value), delimiter)
	}
	return labelsStr
}
