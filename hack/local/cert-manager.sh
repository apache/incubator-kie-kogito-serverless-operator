#!/usr/bin/env bash
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

CERT_MANAGER_INSTALLER="https://github.com/cert-manager/cert-manager/releases/download/v1.12.0/cert-manager.yaml"

# Verify if there is any cert-manager operator installed
function verify() {
    # get any pod that has the status different than Running
    kubectl -n cert-manager rollout status deploy/cert-manager-webhook --timeout=120s
    result_running=$(exec kubectl get pods --namespace cert-manager --field-selector=status.phase==Running --no-headers | wc -l | tr -d ' ')
    if [ "${result_running}" != 3 ]; then
        echo "cert manager is not ready, please verify."
        kubectl get pods --namespace cert-manager
    fi
    echo "ready"
}

function clean_up() {
    i="--ignore-not-found=true"
    kubectl delete $i mutatingwebhookconfiguration.admissionregistration.k8s.io/cert-manager-webhook
    kubectl delete $i validatingwebhookconfiguration.admissionregistration.k8s.io/cert-manager-webhook
}

case "$1" in
    "install")
        # make sure there is no previous webhooks installed:
        clean_up
        kubectl apply -f ${CERT_MANAGER_INSTALLER}
        max_attempts=10
        counter=0
        while [ $counter -lt $max_attempts ]; do
            is_ready=$(verify)
            if [ "${is_ready}" == "ready" ]; then
                break
            else
                counter=$((counter+1))
                # wait 3 seconds
                sleep 3
            fi
        done
        ;;
    "uninstall")
        kubectl delete -f ${CERT_MANAGER_INSTALLER} | true
        clean_up
        ;;
    "verify")
        verify
        ;;
    *)
        echo "Option not recognized, allowed values are [un]install and verify."
        exit 1
        ;;
esac
