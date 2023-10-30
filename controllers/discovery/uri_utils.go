package discovery

import (
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"strings"
)

func resolveServiceUri(service corev1.Service, customPort string, outputFormat string) (string, error) {
	var port int
	var protocol string
	var host string
	var err error = nil

	switch service.Spec.Type {
	case corev1.ServiceTypeExternalName:
		// ExternalName may not work properly with SSL:
		// https://kubernetes.io/docs/concepts/services-networking/service/#externalname
		protocol = httpProtocol
		host = service.Spec.ExternalName
		port = 80
	case corev1.ServiceTypeClusterIP:
		protocol, host, port = resolveClusterIPOrTypeNodeServiceUriParams(service, customPort)
	case corev1.ServiceTypeNodePort:
		protocol, host, port = resolveClusterIPOrTypeNodeServiceUriParams(service, customPort)
	case corev1.ServiceTypeLoadBalancer:
		err = fmt.Errorf("%s type is not yet supported", service.Spec.Type)
	default:
		err = fmt.Errorf("%s type is not yet supported", service.Spec.Type)
	}
	if err != nil {
		return "", err
	}
	if outputFormat == KubernetesDNSAddress {
		return buildKubernetesServiceDNSUri(protocol, service.Namespace, service.Name, port), nil
	} else {
		return buildURI(protocol, host, port), nil
	}
}

// resolveClusterIPOrTypeNodeServiceUriParams returns the uri parameters for a service of type ClusterIP or TypeNode.
// The optional customPort can be used to determine which port should be used for the communication, when not set,
// the best suited port is returned. For this last, a secure port has precedence over a no-secure port.
func resolveClusterIPOrTypeNodeServiceUriParams(service corev1.Service, customPort string) (protocol string, host string, port int) {
	servicePort := findServicePort(service.Spec.Ports, customPort)
	if isServicePortSecure(*servicePort) {
		protocol = httpsProtocol
	} else {
		protocol = httpProtocol
	}
	host = service.Spec.ClusterIP
	port = int(servicePort.Port)
	return protocol, host, port
}

func resolvePodUri(pod *corev1.Pod, customContainer, customPort string, outputFormat string) (string, error) {
	if podIp := pod.Status.PodIP; len(podIp) == 0 {
		return "", fmt.Errorf("pod: %s in namespace: %s, has no allocated address", pod.Name, pod.Namespace)
	} else {
		var container *corev1.Container
		if len(customContainer) > 0 {
			container = findContainerByName(pod.Spec.Containers, customContainer)
		}
		if container == nil {
			container = &pod.Spec.Containers[0]
		}
		if containerPort := findContainerPort(container.Ports, customPort); containerPort == nil {
			return "", fmt.Errorf("no container port was found for pod: %s in namespace: %s", pod.Name, pod.Namespace)
		} else {
			protocol := httpProtocol
			if isSecure := isContainerPortSecure(*containerPort); isSecure {
				protocol = httpsProtocol
			}
			if outputFormat == KubernetesDNSAddress {
				return buildKubernetesPodDNSUri(protocol, pod.Namespace, podIp, int(containerPort.ContainerPort)), nil
			} else {
				return buildURI(protocol, podIp, int(containerPort.ContainerPort)), nil
			}
		}
	}
}

func buildURI(scheme string, host string, port int) string {
	return fmt.Sprintf("%s://%s:%v", scheme, host, port)
}

func buildKubernetesServiceDNSUri(scheme string, namespace string, name string, port int) string {
	return fmt.Sprintf("%s://%s.%s.svc.cluster.local:%v", scheme, name, namespace, port)
}

func buildKubernetesPodDNSUri(scheme string, namespace string, podIP string, port int) string {
	hyphenedIp := strings.Replace(podIP, ".", "-", -1)
	return fmt.Sprintf("%s://%s.%s.pod.cluster.local:%v", scheme, hyphenedIp, namespace, port)
}
