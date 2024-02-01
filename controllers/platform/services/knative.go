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
	operatorapi "github.com/apache/incubator-kie-kogito-serverless-operator/api/v1alpha08"
	"k8s.io/utils/pointer"
)

func IsDataIndexEnabled(plf *operatorapi.SonataFlowPlatform) bool {
	if plf.Spec.Services != nil {
		if plf.Spec.Services.DataIndex != nil {
			return pointer.BoolDeref(plf.Spec.Services.DataIndex.Enabled, false)
		}
		return false
	}
	// Check if DataIndex is enabled in the platform status
	if plf.Status.ClusterPlatformRef != nil && plf.Status.ClusterPlatformRef.Services != nil && plf.Status.ClusterPlatformRef.Services.DataIndexRef != nil && len(plf.Status.ClusterPlatformRef.Services.DataIndexRef.Url) > 0 {
		return true
	}
	return false
}

func IsJobServiceEnabled(plf *operatorapi.SonataFlowPlatform) bool {
	if plf.Spec.Services != nil {
		if plf.Spec.Services.JobService != nil {
			return pointer.BoolDeref(plf.Spec.Services.JobService.Enabled, false)
		}
		return false
	}
	// Check if JobService is enabled in the platform status
	if plf.Status.ClusterPlatformRef != nil && plf.Status.ClusterPlatformRef.Services != nil && plf.Status.ClusterPlatformRef.Services.JobServiceRef != nil && len(plf.Status.ClusterPlatformRef.Services.JobServiceRef.Url) > 0 {
		return true
	}
	return false
}
