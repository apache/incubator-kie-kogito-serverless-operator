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

# runs the e2e locally
# You must have minikube installed

os=$(uname | awk '{print tolower($0)}')
minikube_ip=$(minikube ip)

if [ "${os}" == 'darwin' ]; then
    echo "Make sure to run in a separate terminal 'docker run --rm -it --network=host alpine ash -c \"apk add socat && socat TCP-LISTEN:5000,reuseaddr,fork TCP:\$(minikube ip):5000\"'"
    minikube_ip="localhost"
fi

minikube_registry="${minikube_ip}:5000"
export OPERATOR_IMAGE_NAME=${minikube_registry}/kogito-serverless-operator:0.0.1

make container-build BUILDER=docker IMG="${OPERATOR_IMAGE_NAME}"

if ! docker push "${OPERATOR_IMAGE_NAME}"; then
  echo "Failed to push image. Stopping test."
  exit 1
fi


make test-e2e
