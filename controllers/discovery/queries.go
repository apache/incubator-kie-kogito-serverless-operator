package discovery

import (
	"context"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingV1 "k8s.io/api/networking/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const podTemplateHashLabel = "pod-template-hash"

// findService finds a service by name in the given namespace.
func findService(ctx context.Context, cli client.Client, namespace string, name string) (*corev1.Service, error) {
	service := &corev1.Service{}
	if err := cli.Get(ctx, buildObjectKey(namespace, name), service); err != nil {
		return nil, err
	}
	return service, nil
}

// findServiceByLabels finds a service by a set of matching labels in the given namespace.
func findServiceByLabels(ctx context.Context, cli client.Client, namespace string, labels map[string]string) (*corev1.ServiceList, error) {
	serviceList := &corev1.ServiceList{}
	if err := cli.List(ctx, serviceList, client.InNamespace(namespace), client.MatchingLabels(labels)); err != nil {
		return nil, err
	}
	return serviceList, nil
}

// findPod finds a pod by name in the given namespace.
func findPod(ctx context.Context, cli client.Client, namespace string, name string) (*corev1.Pod, error) {
	pod := &corev1.Pod{}
	if err := cli.Get(ctx, buildObjectKey(namespace, name), pod); err != nil {
		return nil, err
	}
	return pod, nil
}

// findPodAndReferenceServiceByPodLabels finds a pod by name in the given namespace at the same time it piggybacks it's
// reference service if any. The reference service is determined by using the same set of labels as the pod.
func findPodAndReferenceServiceByPodLabels(ctx context.Context, cli client.Client, namespace string, podName string) (*corev1.Pod, *corev1.Service, error) {
	if pod, err := findPod(ctx, cli, namespace, podName); err != nil {
		return nil, nil, err
	} else {
		queryLabels := pod.Labels
		// pod-template-hash is pod dependent, mustn't be considered.
		delete(queryLabels, podTemplateHashLabel)
		if len(queryLabels) > 0 {
			// check if we have a defined reference service
			if serviceList, err2 := findServiceByLabels(ctx, cli, namespace, queryLabels); err2 != nil {
				return nil, nil, err
			} else if len(serviceList.Items) > 0 {
				return pod, &serviceList.Items[0], nil
			}
		}
		return pod, nil, nil
	}
}

// findDeployment finds a deployment by name in the given namespace.
func findDeployment(ctx context.Context, cli client.Client, namespace string, name string) (*appsv1.Deployment, error) {
	deployment := &appsv1.Deployment{}
	if err := cli.Get(ctx, buildObjectKey(namespace, name), deployment); err != nil {
		return nil, err
	}
	return deployment, nil
}

// findStatefulSet finds a stateful set by name in the given namespace.
func findStatefulSet(ctx context.Context, cli client.Client, namespace string, name string) (*appsv1.StatefulSet, error) {
	statefulSet := &appsv1.StatefulSet{}
	if err := cli.Get(ctx, buildObjectKey(namespace, name), statefulSet); err != nil {
		return nil, err
	}
	return statefulSet, nil
}

// findIngress finds an ingress by name in the given namespace.
func findIngress(ctx context.Context, cli client.Client, namespace string, name string) (*networkingV1.Ingress, error) {
	ingress := &networkingV1.Ingress{}
	if err := cli.Get(ctx, buildObjectKey(namespace, name), ingress); err != nil {
		return nil, err
	}
	return ingress, nil
}

func buildObjectKey(namespace string, name string) client.ObjectKey {
	return client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}
}
