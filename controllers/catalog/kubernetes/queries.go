package kubernetes

import (
	"context"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingV1 "k8s.io/api/networking/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const podTemplateHashLabel = "pod-template-hash"

// FindService finds a service by name in the given namespace.
func FindService(ctx context.Context, cli client.Client, namespace string, name string) (*corev1.Service, error) {
	service := &corev1.Service{}
	if err := cli.Get(ctx, buildObjectKey(namespace, name), service); err != nil {
		return nil, err
	}
	return service, nil
}

// FindServiceByLabels finds a service by a set of matching labels in the given namespace.
func FindServiceByLabels(ctx context.Context, cli client.Client, namespace string, labels map[string]string) (*corev1.ServiceList, error) {
	serviceList := &corev1.ServiceList{}
	if err := cli.List(ctx, serviceList, client.InNamespace(namespace), client.MatchingLabels(labels)); err != nil {
		return nil, err
	}
	return serviceList, nil
}

// FindPod finds a pod by name in the given namespace.
func FindPod(ctx context.Context, cli client.Client, namespace string, name string) (*corev1.Pod, error) {
	pod := &corev1.Pod{}
	if err := cli.Get(ctx, buildObjectKey(namespace, name), pod); err != nil {
		return nil, err
	}
	return pod, nil
}

// FindPodAndReferenceServiceByPodLabels finds a pod by name in the given namespace at the same time it piggybacks it's
// reference service if any. The reference service is determined by using the same set of labels as the pod.
func FindPodAndReferenceServiceByPodLabels(ctx context.Context, cli client.Client, namespace string, podName string) (*corev1.Pod, *corev1.Service, error) {
	if pod, err := FindPod(ctx, cli, namespace, podName); err != nil {
		return nil, nil, err
	} else {
		queryLabels := pod.Labels
		// pod-template-hash is pod dependent, mustn't be considered.
		delete(queryLabels, podTemplateHashLabel)
		if len(queryLabels) > 0 {
			// check if we have a defined reference service
			if serviceList, err2 := FindServiceByLabels(ctx, cli, namespace, queryLabels); err2 != nil {
				return nil, nil, err
			} else if len(serviceList.Items) > 0 {
				return pod, &serviceList.Items[0], nil
			}
		}
		return pod, nil, nil
	}
}

// FindDeployment finds a deployment by name in the given namespace.
func FindDeployment(ctx context.Context, cli client.Client, namespace string, name string) (*appsv1.Deployment, error) {
	deployment := &appsv1.Deployment{}
	if err := cli.Get(ctx, buildObjectKey(namespace, name), deployment); err != nil {
		return nil, err
	}
	return deployment, nil
}

// FindStatefulSet finds a stateful set by name in the given namespace.
func FindStatefulSet(ctx context.Context, cli client.Client, namespace string, name string) (*appsv1.StatefulSet, error) {
	statefulSet := &appsv1.StatefulSet{}
	if err := cli.Get(ctx, buildObjectKey(namespace, name), statefulSet); err != nil {
		return nil, err
	}
	return statefulSet, nil
}

// FindIngress finds an ingress by name in the given namespace.
func FindIngress(ctx context.Context, cli client.Client, namespace string, name string) (*networkingV1.Ingress, error) {
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
