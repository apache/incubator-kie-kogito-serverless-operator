/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

package knative

import (
	"k8s.io/client-go/rest"
)

type MonitoringAvailability struct {
	Prometheus bool
	Grafana    bool
}

const (
	prometheusGroup = "prometheuses.monitoring.coreos.com"
)

func GetMonitoringAvailability(cfg *rest.Config) (*MonitoringAvailability, error) {
	if cli, err := getDiscoveryClient(cfg); err != nil {
		return nil, err
	} else {
		apiList, err := cli.ServerGroups()
		if err != nil {
			return nil, err
		}
		result := new(MonitoringAvailability)
		for _, group := range apiList.Groups {
			if group.Name == prometheusGroup {
				result.Prometheus = true
			}
			if group.Name == knativeEventingGroup {
				result.Grafana = true
			}
		}
		return result, nil
	}
}
