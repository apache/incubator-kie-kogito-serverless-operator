package utils

import (
	"context"
	"github.com/kiegroup/container-builder/util/log"
	"github.com/kiegroup/kogito-serverless-operator/constants"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetBuilderCommonConfigMap retrieves the config map with the builder common configuration information
func GetBuilderCommonConfigMap(client client.Client) (corev1.ConfigMap, error) {

	namespace, found := os.LookupEnv("POD_NAMESPACE")

	if !found {
		return corev1.ConfigMap{}, errors.New("ConfigMap not found")
	}

	existingConfigMap := corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      constants.BUILDER_CM_NAME,
			Namespace: namespace,
		},
		Data: map[string]string{},
	}

	err := client.Get(context.TODO(), types.NamespacedName{Name: constants.BUILDER_CM_NAME, Namespace: namespace}, &existingConfigMap)
	if err != nil {
		log.Error(err, "reading configmap")
		return corev1.ConfigMap{}, err
	} else {
		return existingConfigMap, nil
	}
}
