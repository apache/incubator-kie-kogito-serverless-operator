package platform

import (
	"context"
	v08 "github.com/davidesalerno/kogito-serverless-operator/api/v08"
	"github.com/davidesalerno/kogito-serverless-operator/builder"
	"github.com/davidesalerno/kogito-serverless-operator/constants"
	"github.com/ricardozanini/kogito-builder/api"
	"github.com/ricardozanini/kogito-builder/client"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewInitializeAction returns a action that initializes the platform configuration when not provided by the user.
func NewInitializeAction() Action {
	return &initializeAction{}
}

type initializeAction struct {
	baseAction
}

func (action *initializeAction) Name() string {
	return "initialize"
}

func (action *initializeAction) CanHandle(platform *v08.KogitoServerlessPlatform) bool {
	return platform.Status.Phase == "" || platform.Status.Phase == v08.PlatformPhaseDuplicate
}

func (action *initializeAction) Handle(ctx context.Context, platform *v08.KogitoServerlessPlatform) (*v08.KogitoServerlessPlatform, error) {
	duplicate, err := action.isPrimaryDuplicate(ctx, platform)
	if err != nil {
		return nil, err
	}
	if duplicate {
		// another platform already present in the namespace
		if platform.Status.Phase != v08.PlatformPhaseDuplicate {
			platform := platform.DeepCopy()
			platform.Status.Phase = v08.PlatformPhaseDuplicate

			return platform, nil
		}

		return nil, nil
	}

	if err = ConfigureDefaults(ctx, action.client, platform, true); err != nil {
		return nil, err
	}
	// nolint: staticcheck
	if platform.Status.BuildPlatform.PublishStrategy == api.PlatformBuildPublishStrategyKaniko {
		cacheEnabled := platform.Status.BuildPlatform.IsOptionEnabled(builder.KanikoBuildCacheEnabled)
		//If KanikoCache is enabled
		if cacheEnabled {
			// Create the persistent volume claim used by the Kaniko cache
			action.L.Info("Create persistent volume claim")
			err := createPersistentVolumeClaim(ctx, action.client, platform)
			if err != nil {
				return nil, err
			}
			// Create the Kaniko warmer pod that caches the base image into the Camel K builder volume
			action.L.Info("Create Kaniko cache warmer pod")
			err = createKanikoCacheWarmerPod(ctx, action.client, platform)
			if err != nil {
				return nil, err
			}
			platform.Status.Phase = v08.PlatformPhaseWarming
		} else {
			// Skip the warmer pod creation
			platform.Status.Phase = v08.PlatformPhaseCreating
		}
	} else {
		platform.Status.Phase = v08.PlatformPhaseCreating
	}
	platform.Status.Version = constants.VERSION

	return platform, nil
}

func createPersistentVolumeClaim(ctx context.Context, client client.Client, platform *v08.KogitoServerlessPlatform) error {
	volumeSize, err := resource.ParseQuantity("1Gi")
	if err != nil {
		return err
	}
	// nolint: staticcheck
	pvcName := constants.DEFAULT_KANIKOCACHE_PVC_NAME
	if persistentVolumeClaim, found := platform.Status.BuildPlatform.PublishStrategyOptions[builder.KanikoPVCName]; found {
		pvcName = persistentVolumeClaim
	}

	pvc := &corev1.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "PersistentVolumeClaim",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: platform.Namespace,
			Name:      pvcName,
			Labels: map[string]string{
				"app": "camel-k",
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: volumeSize,
				},
			},
		},
	}

	err = client.Create(ctx, pvc)
	// Skip the error in case the PVC already exists
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return err
	}

	return nil
}

func (action *initializeAction) isPrimaryDuplicate(ctx context.Context, thisPlatform *v08.KogitoServerlessPlatform) (bool, error) {
	if IsSecondary(thisPlatform) {
		// Always reconcile secondary platforms
		return false, nil
	}
	platforms, err := ListPrimaryPlatforms(ctx, action.client, thisPlatform.Namespace)
	if err != nil {
		return false, err
	}
	for _, p := range platforms.Items {
		p := p // pin
		if p.Name != thisPlatform.Name && IsActive(&p) {
			return true, nil
		}
	}

	return false, nil
}
