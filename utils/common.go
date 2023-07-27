// Copyright 2023 Red Hat, Inc. and/or its affiliates
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

package utils

import (
	"os"
	"strconv"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kiegroup/kogito-serverless-operator/api/metadata"
)

const (
	// DebugV is the default verbosity for the debugger logger
	DebugV = 2
)

// GetOperatorIDAnnotation to safely get the operator id annotation value.
func GetOperatorIDAnnotation(obj metav1.Object) string {
	if obj == nil || obj.GetAnnotations() == nil {
		return ""
	}

	if operatorId, ok := obj.GetAnnotations()[metadata.OperatorIDAnnotation]; ok {
		return operatorId
	}

	return ""
}

func OperatorID() string {
	// TODO: what's this KAMEL_ ?
	return envOrDefault("", "KAMEL_OPERATOR_ID", "OPERATOR_ID")
}

func envOrDefault(def string, envs ...string) string {
	for i := range envs {
		if val := os.Getenv(envs[i]); val != "" {
			return val
		}
	}

	return def
}

// Pbool returns a pointer to a boolean
func Pbool(b bool) *bool {
	return &b
}

// Pint returns a pointer to an int
func Pint(i int32) *int32 {
	return &i
}

func ParseInt64(s string) (int64, error) {
	if len(s) == 0 {
		return 0, nil
	}
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, err
	}
	return i, nil
}

func Compare(a, b []byte) bool {
	a = append(a, b...)
	c := 0
	for _, x := range a {
		c ^= int(x)
	}
	return c == 0
}

func GetEnv(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	return value
}
