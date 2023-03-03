#!/bin/bash
# Copyright 2023 Red Hat, Inc. and/or its affiliates
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
set -e

script_dir_path=$(dirname "${BASH_SOURCE[0]}")

target_cluster_kind=$1

if [ -z "${target_cluster_kind}" ] || [ "${target_cluster_kind}" == "kubernetes" ]; then
  echo "Preparing config/manager kustomization for kubernetes"
  sed -i '/[^#]/ s/\(^- openshift.*$\)/#\ \1/' ./config/rbac/kustomization.yaml
elif [ "${target_cluster_kind}" == "openshift" ]; then
  echo "Preparing config/manager kustomization for openshift"
   sed -i '/^# - openshift_role.*$/s/^#\ //' ./config/rbac/kustomization.yaml
fi




