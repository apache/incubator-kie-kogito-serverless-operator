package platform

import (
	"context"
	"errors"
	v08 "github.com/davidesalerno/kogito-serverless-operator/api/v08"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
)

func NewWarmAction(reader ctrl.Reader) Action {
	return &warmAction{
		reader: reader,
	}
}

type warmAction struct {
	baseAction
	reader ctrl.Reader
}

func (action *warmAction) Name() string {
	return "warm"
}

func (action *warmAction) CanHandle(platform *v08.KogitoServerlessPlatform) bool {
	return platform.Status.Phase == v08.PlatformPhaseWarming
}

func (action *warmAction) Handle(ctx context.Context, platform *v08.KogitoServerlessPlatform) (*v08.KogitoServerlessPlatform, error) {
	// Check Kaniko warmer pod status
	pod := corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: platform.Namespace,
			Name:      platform.Name + "-cache",
		},
	}

	err := action.reader.Get(ctx, types.NamespacedName{Namespace: pod.Namespace, Name: pod.Name}, &pod)
	if err != nil {
		return nil, err
	}

	switch pod.Status.Phase {
	case corev1.PodSucceeded:
		action.L.Info("Kaniko cache successfully warmed up")
		platform.Status.Phase = v08.PlatformPhaseCreating
		return platform, nil
	case corev1.PodFailed:
		return nil, errors.New("failed to warm up Kaniko cache")
	default:
		action.L.Info("Waiting for Kaniko cache to warm up...")
		// Requeue
		return nil, nil
	}
}
