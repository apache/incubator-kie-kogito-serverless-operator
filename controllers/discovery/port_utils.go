package discovery

import (
	corev1 "k8s.io/api/core/v1"
)

const (
	httpProtocol  = "http"
	httpsProtocol = "https"
	webProtocol   = "web"
	securePort    = 443
	appSecurePort = 8443
)

func isSecurePort(port int) bool {
	return port == securePort || port == appSecurePort
}

// findServicePort returns the best suited ServicePort to connect to a service.
// The optional customPort can be used to determine which port should be used for the communication, when not set,
// the best suited port is returned. For this last, a secure port has precedence over a no-secure port.
func findServicePort(servicePorts []corev1.ServicePort, customPort string) *corev1.ServicePort {
	// customPort is provided and is configured?
	if len(customPort) > 0 {
		if result := findServicePortByName(servicePorts, customPort); result != nil {
			return result
		}
	}
	// has ssl port?
	if result := findServicePortByName(servicePorts, httpsProtocol); result != nil {
		return result
	}
	// has http port?
	if result := findServicePortByName(servicePorts, httpProtocol); result != nil {
		return result
	}
	// has web port?
	if result := findServicePortByName(servicePorts, webProtocol); result != nil {
		return result
	}
	// by definition a service must always have at least one port, get the first port.
	return &servicePorts[0]
}

func findServicePortByName(ports []corev1.ServicePort, name string) *corev1.ServicePort {
	for _, servicePort := range ports {
		if name == servicePort.Name {
			return &servicePort
		}
	}
	return nil
}

func isServicePortSecure(servicePort corev1.ServicePort) bool {
	return servicePort.Name == httpsProtocol || isSecurePort(int(servicePort.Port))
}

// findContainerPort returns the best suited PortPort to connect to a pod, or nil if the pod has no ports at all.
// The optional customPort can be used to determine which port should be used for the communication, when not set,
// the best suited port is returned. For this last, a secure port has precedence over a non-secure port.
func findContainerPort(containerPorts []corev1.ContainerPort, customPort string) *corev1.ContainerPort {
	// containers with no ports are permitted, we must check.
	if len(containerPorts) == 0 {
		return nil
	}
	// customPort is provided and configured?
	if len(customPort) > 0 {
		if result := findContainerPortByName(containerPorts, customPort); result != nil {
			return result
		}
	}
	// has ssl port?
	if result := findContainerPortByName(containerPorts, httpsProtocol); result != nil {
		return result
	}
	// has http port?
	if result := findContainerPortByName(containerPorts, httpProtocol); result != nil {
		return result
	}
	// has web port?
	if result := findContainerPortByName(containerPorts, webProtocol); result != nil {
		return result
	}
	// when defined, a ContainerPort must always have containerPort (Required value)
	return &containerPorts[0]
}

func findContainerPortByName(containerPorts []corev1.ContainerPort, name string) *corev1.ContainerPort {
	for _, containerPort := range containerPorts {
		if name == containerPort.Name {
			return &containerPort
		}
	}
	return nil
}

func findContainerByName(containers []corev1.Container, name string) *corev1.Container {
	for _, container := range containers {
		if name == container.Name {
			return &container
		}
	}
	return nil
}

func isContainerPortSecure(containerPort corev1.ContainerPort) bool {
	return containerPort.Name == httpsProtocol || isSecurePort(int(containerPort.ContainerPort))
}
