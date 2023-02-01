/*
 * Copyright 2022 Red Hat, Inc. and/or its affiliates.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package utils

import (
	"bytes"
	"io/ioutil"
	"log"

	"k8s.io/apimachinery/pkg/util/yaml"

	apiv08 "github.com/kiegroup/kogito-serverless-operator/api/v1alpha08"
)

func GetKogitoServerlessWorkflow(path string) (*apiv08.KogitoServerlessWorkflow, error) {

	ksw := &apiv08.KogitoServerlessWorkflow{}
	yamlFile, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatalf("yamlFile.Get err   #%v ", err)
		return nil, err
	}
	// Important: Here we are reading the CR deployment file from a given path and creating a &apiv08.KogitoServerlessWorkflow struct
	err = yaml.NewYAMLOrJSONDecoder(bytes.NewReader(yamlFile), 100).Decode(ksw)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
		return nil, err
	}
	log.Printf("Successfully read KSW  #%v ", ksw)
	return ksw, err
}
