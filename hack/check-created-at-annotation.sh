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

# See https://github.com/operator-framework/operator-sdk/issues/6285
# and https://github.com/operator-framework/operator-sdk/pull/6419
# Since createdAt field is always updated when make bundle is run
# we check if the bundle/manifests/sonataflow-operator.clusterserviceversion.yaml is modified with git
# and then we check if is the change involve only the createdAt, if yes we suppress the error on the Github Action
# this fix could be remove when the operator sdk provides a flag to avoid this change that broke the Github action checks

changed_files=$(git status -s)
check_file=$(expr "$changed_files" == "M bundle/manifests/sonataflow-operator.clusterserviceversion.yaml")

if [[ "$check_file" == "0" ]] ; then
  check_lines=$(git diff HEAD --no-ext-diff --unified=0 --exit-code -a --no-prefix | grep "^\+")
  var="+++ bundle/manifests/sonataflow-operator.clusterserviceversion.yaml + createdAt:"
  if [[ $check_lines = $var* ]] ; then
     changed_files=''
  fi
else
  [[ -z "$changed_files" ]] ||  (printf "Generation has not been done on this PR. See modified files: \n$changed_files\n Did you run 'make generate-all' before sending the PR" && exit 1)
fi