package platform

import (
	"context"
	"fmt"
	"github.com/kiegroup/container-builder/client"
	"github.com/kiegroup/container-builder/util/defaults"
	v08 "github.com/kiegroup/kogito-serverless-operator/api/v1alpha08"
	"github.com/kiegroup/kogito-serverless-operator/builder"
	"github.com/kiegroup/kogito-serverless-operator/constants"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func createKanikoCacheWarmerPod(ctx context.Context, client client.Client, platform *v08.KogitoServerlessPlatform) error {
	// The pod will be scheduled to nodes that are selected by the persistent volume
	// node affinity spec, if any, as provisioned by the persistent volume claim storage
	// class provisioner.
	// See:
	// - https://kubernetes.io/docs/concepts/storage/persistent-volumes/#node-affinity
	// - https://kubernetes.io/docs/concepts/storage/volumes/#local
	// nolint: staticcheck
	pvcName := constants.DEFAULT_KANIKOCACHE_PVC_NAME
	if persistentVolumeClaim, found := platform.Status.BuildPlatform.PublishStrategyOptions[builder.KanikoPVCName]; found {
		pvcName = persistentVolumeClaim
	}

	var warmerImage string
	if image, found := platform.Status.BuildPlatform.PublishStrategyOptions[builder.KanikoWarmerImage]; found {
		warmerImage = image
	} else {
		warmerImage = fmt.Sprintf("%s:v%s", builder.KanikoDefaultWarmerImageName, defaults.KanikoVersion)
	}

	pod := corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: platform.Namespace,
			Name:      platform.Name + "-cache",
			Labels: map[string]string{
				"camel.apache.org/component": "kaniko-warmer",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "warm-kaniko-cache",
					Image: warmerImage,
					Args: []string{
						"--cache-dir=" + builder.KanikoCacheDir,
						"--image=" + platform.Status.BuildPlatform.BaseImage,
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "kaniko-cache",
							MountPath: builder.KanikoCacheDir,
						},
					},
				},
			},
			// Create the cache directory otherwise Kaniko warmer skips caching silently
			InitContainers: []corev1.Container{
				{
					Name:            "create-kaniko-cache",
					Image:           "busybox",
					ImagePullPolicy: corev1.PullIfNotPresent,
					Command:         []string{"/bin/sh", "-c"},
					Args:            []string{"mkdir -p " + builder.KanikoCacheDir + "&& chmod -R a+rwx " + builder.KanikoCacheDir},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "kaniko-cache",
							MountPath: builder.KanikoCacheDir,
						},
					},
				},
			},
			RestartPolicy: corev1.RestartPolicyOnFailure,
			Volumes: []corev1.Volume{
				{
					Name: "kaniko-cache",
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: pvcName,
						},
					},
				},
			},
		},
	}

	err := client.Delete(ctx, &pod)
	if err != nil && !k8serrors.IsNotFound(err) {
		return errors.Wrap(err, "cannot delete Kaniko warmer pod")
	}

	err = client.Create(ctx, &pod)
	if err != nil {
		return errors.Wrap(err, "cannot create Kaniko warmer pod")
	}

	return nil
}
