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

package services

import (
	"context"

	operatorapi "github.com/apache/incubator-kie-kogito-serverless-operator/api/v1alpha08"
	"github.com/apache/incubator-kie-kogito-serverless-operator/container-builder/client"
	"github.com/apache/incubator-kie-kogito-serverless-operator/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

func GetSecretKeyValueString(ctx context.Context, client client.Client, secretName string, secretKey string, platform *operatorapi.SonataFlowPlatform) (string, error) {
	secret, err := client.CoreV1().Secrets(platform.Namespace).Get(ctx,
		secretName, metav1.GetOptions{})
	if err != nil {
		klog.V(log.E).InfoS("Error extracting secret: ", "namespace", platform.Namespace, "error", err)
		return "", err
	}

	return string(secret.Data[secretKey]), nil
}
