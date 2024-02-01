// Copyright 2024 Apache Software Foundation (ASF)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package common

import (
	"context"

	operatorapi "github.com/apache/incubator-kie-kogito-serverless-operator/api/v1alpha08"
	"github.com/apache/incubator-kie-kogito-serverless-operator/controllers/knative"
	"github.com/apache/incubator-kie-kogito-serverless-operator/log"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	KnativeBundleVolume = "kne-bundle-volume"
)

var _ KnativeEventingHandler = &knativeObjectManager{}

type knativeObjectManager struct {
	sinkBinding ObjectEnsurerWithPlatform
	trigger     ObjectsEnsurerWithPlatform
	platform    *operatorapi.SonataFlowPlatform
	*StateSupport
}

func NewKnativeEventingHandler(support *StateSupport, pl *operatorapi.SonataFlowPlatform) KnativeEventingHandler {
	return &knativeObjectManager{
		sinkBinding:  NewObjectEnsurerWithPlatform(support.C, SinkBindingCreator),
		trigger:      NewObjectsEnsurerWithPlatform(support.C, TriggersCreator),
		platform:     pl,
		StateSupport: support,
	}
}

type KnativeEventingHandler interface {
	Ensure(ctx context.Context, workflow *operatorapi.SonataFlow) ([]client.Object, error)
}

func (k knativeObjectManager) Ensure(ctx context.Context, workflow *operatorapi.SonataFlow) ([]client.Object, error) {
	var objs []client.Object

	knativeAvail, err := knative.GetKnativeAvailability(k.Cfg)
	if err != nil {
		klog.V(log.I).InfoS("Error checking Knative Eventing: %v", err)
		return nil, err
	}
	if !knativeAvail.Eventing {
		klog.V(log.I).InfoS("Knative Eventing is not installed")
	} else {
		// create sinkBinding and trigger
		sinkBinding, _, err := k.sinkBinding.Ensure(ctx, workflow, k.platform)
		if err != nil {
			return objs, err
		} else if sinkBinding != nil {
			objs = append(objs, sinkBinding)
		}

		triggers := k.trigger.Ensure(ctx, workflow, k.platform)
		for _, trigger := range triggers {
			if trigger.Error != nil {
				return objs, trigger.Error
			}
			objs = append(objs, trigger.Object)
		}
	}
	return objs, nil
}

func moveKnativeVolumeToEnd(vols []corev1.Volume) {
	for i := 0; i < len(vols)-1; i++ {
		if vols[i].Name == KnativeBundleVolume {
			vols[i], vols[i+1] = vols[i+1], vols[i]
		}
	}
}

func moveKnativeVolumeMountToEnd(mounts []corev1.VolumeMount) {
	for i := 0; i < len(mounts)-1; i++ {
		if mounts[i].Name == KnativeBundleVolume {
			mounts[i], mounts[i+1] = mounts[i+1], mounts[i]
		}
	}
}

// Knative Sinkbinding injects K_SINK env, a volume and volumn mount. The volume and volume mount
// must be in the end of the array to avoid repeadly restarting of the workflow pod
func restoreKnativeVolumeAndVolumeMount(deployment *appsv1.Deployment) {
	moveKnativeVolumeToEnd(deployment.Spec.Template.Spec.Volumes)
	for i := 0; i < len(deployment.Spec.Template.Spec.Containers); i++ {
		moveKnativeVolumeMountToEnd(deployment.Spec.Template.Spec.Containers[i].VolumeMounts)
	}
}

func preserveKnativeVolumeMount(object *appsv1.Deployment) {
	var kneVol *corev1.Volume = nil
	for _, v := range object.Spec.Template.Spec.Volumes {
		if v.Name == KnativeBundleVolume {
			kneVol = &v
		}
	}
	if kneVol != nil {
		object.Spec.Template.Spec.Volumes = []corev1.Volume{*kneVol}
	} else {
		object.Spec.Template.Spec.Volumes = nil
	}
	for i := range object.Spec.Template.Spec.Containers {
		var kneVolMount *corev1.VolumeMount = nil
		for _, mount := range object.Spec.Template.Spec.Containers[i].VolumeMounts {
			if mount.Name == KnativeBundleVolume {
				kneVolMount = &mount
			}
		}
		if kneVolMount == nil {
			object.Spec.Template.Spec.Containers[i].VolumeMounts = nil
		} else {
			object.Spec.Template.Spec.Containers[i].VolumeMounts = []corev1.VolumeMount{*kneVolMount}
		}
	}
}
