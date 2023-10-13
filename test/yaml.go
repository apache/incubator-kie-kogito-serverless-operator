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

package test

import (
	"bytes"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/apache/incubator-kie-kogito-serverless-operator/api"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"

	"github.com/apache/incubator-kie-kogito-serverless-operator/log"

	operatorapi "github.com/apache/incubator-kie-kogito-serverless-operator/api/v1alpha08"
)

const (
	sonataFlowOrderProcessingFolder           = "order-processing"
	sonataFlowSampleYamlCR                    = "sonataflow.org_v1alpha08_sonataflow.yaml"
	SonataFlowGreetingsWithDataInputSchemaCR  = "sonataflow.org_v1alpha08_sonataflow_greetings_datainput.yaml"
	SonataFlowGreetingsWithStaticResourcesCR  = "sonataflow.org_v1alpha08_sonataflow-metainf.yaml"
	SonataFlowSimpleOpsYamlCR                 = "sonataflow.org_v1alpha08_sonataflow-simpleops.yaml"
	SonataFlowGreetingsDataInputSchemaConfig  = "v1_configmap_greetings_datainput.yaml"
	SonataFlowGreetingsStaticFilesConfig      = "v1_configmap_greetings_staticfiles.yaml"
	sonataFlowPlatformYamlCR                  = "sonataflow.org_v1alpha08_sonataflowplatform.yaml"
	sonataFlowPlatformWithCacheMinikubeYamlCR = "sonataflow.org_v1alpha08_sonataflowplatform_withCache_minikube.yaml"
	sonataFlowPlatformForOpenshift            = "sonataflow.org_v1alpha08_sonataflowplatform_openshift.yaml"
	sonataFlowBuilderConfig                   = "sonataflow-operator-builder-config_v1_configmap.yaml"
	sonataFlowBuildSucceed                    = "sonataflow.org_v1alpha08_sonataflowbuild.yaml"

	e2eSamples    = "test/testdata/"
	manifestsPath = "bundle/manifests/"
)

var projectDir = ""

func GetSonataFlow(testFile, namespace string) *operatorapi.SonataFlow {
	ksw := &operatorapi.SonataFlow{}

	GetKubernetesResource(testFile, ksw)
	klog.V(log.D).InfoS("Successfully read KSW", "ksw", spew.Sprint(ksw))
	ksw.Namespace = namespace
	return ksw
}

func GetKubernetesResource(testFile string, resource client.Object) {
	yamlFile, err := os.ReadFile(path.Join(getTestDataDir(), testFile))
	if err != nil {
		klog.V(log.E).ErrorS(err, "yamlFile.Get")
		panic(err)
	}

	// Important: Here we are reading the CR deployment file from a given path and creating an &operatorapi.SonataFlow struct
	err = yaml.NewYAMLOrJSONDecoder(bytes.NewReader(yamlFile), 100).Decode(resource)
	if err != nil {
		klog.V(log.E).ErrorS(err, "Unmarshal")
		panic(err)
	}
}

func getSonataFlowPlatform(testFile string) *operatorapi.SonataFlowPlatform {
	ksp := &operatorapi.SonataFlowPlatform{}
	yamlFile, err := os.ReadFile(path.Join(getTestDataDir(), testFile))
	if err != nil {
		klog.V(log.E).ErrorS(err, "yamlFile.Get")
		panic(err)
	}
	// Important: Here we are reading the CR deployment file from a given path and creating a &operatorapi.SonataFlowPlatform struct
	err = yaml.NewYAMLOrJSONDecoder(bytes.NewReader(yamlFile), 100).Decode(ksp)
	if err != nil {
		klog.V(log.E).ErrorS(err, "Unmarshal")
		panic(err)
	}
	klog.V(log.D).InfoS("Successfully read KSP", "ksp", ksp)
	ksp.Status.Manager().InitializeConditions()
	return ksp
}

func GetSonataFlowPlatformInReadyPhase(path string, namespace string) *operatorapi.SonataFlowPlatform {
	ksp := getSonataFlowPlatform(path)
	ksp.Status.Manager().MarkTrue(api.SucceedConditionType)
	ksp.Namespace = namespace
	return ksp
}

func GetNewEmptySonataFlowBuild(name, namespace string) *operatorapi.SonataFlowBuild {
	return &operatorapi.SonataFlowBuild{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: operatorapi.SonataFlowBuildSpec{
			BuildTemplate: operatorapi.BuildTemplate{
				Resources: corev1.ResourceRequirements{},
				Arguments: []string{},
			},
		},
		Status: operatorapi.SonataFlowBuildStatus{},
	}
}

// GetLocalSucceedSonataFlowBuild gets a local (testdata dir ref to caller) SonataFlowBuild with Succeed status equals to true.
func GetLocalSucceedSonataFlowBuild(name, namespace string) *operatorapi.SonataFlowBuild {
	yamlFile, err := os.ReadFile(path.Join(getTestDataDir(), sonataFlowBuildSucceed))
	if err != nil {
		klog.ErrorS(err, "Yaml file not found on local testdata dir")
		panic(err)
	}
	build := &operatorapi.SonataFlowBuild{}
	if err = yaml.NewYAMLOrJSONDecoder(bytes.NewReader(yamlFile), 255).Decode(build); err != nil {
		klog.ErrorS(err, "Failed to unmarshal SonataFlowBuild")
		panic(err)
	}
	build.Name = name
	build.Namespace = namespace
	return build
}

func GetSonataFlowBuilderConfig(namespace string) *corev1.ConfigMap {
	cm := &corev1.ConfigMap{}
	yamlFile, err := os.ReadFile(path.Join(getBundleDir(), sonataFlowBuilderConfig))
	if err != nil {
		klog.V(log.E).ErrorS(err, "yamlFile.Get")
		panic(err)
	}
	err = yaml.NewYAMLOrJSONDecoder(bytes.NewReader(yamlFile), 100).Decode(cm)
	if err != nil {
		klog.V(log.E).ErrorS(err, "Unmarshal")
		panic(err)
	}
	cm.Namespace = namespace
	return cm
}

func GetBaseSonataFlow(namespace string) *operatorapi.SonataFlow {
	return GetSonataFlow(sonataFlowSampleYamlCR, namespace)
}

func GetBaseSonataFlowWithDevProfile(namespace string) *operatorapi.SonataFlow {
	workflow := GetBaseSonataFlow(namespace)
	workflow.Annotations["sonataflow.org/profile"] = "dev"
	return workflow
}

func GetBaseSonataFlowWithProdProfile(namespace string) *operatorapi.SonataFlow {
	workflow := GetBaseSonataFlow(namespace)
	workflow.Annotations["sonataflow.org/profile"] = "prod"
	return workflow
}

func GetBaseSonataFlowWithProdOpsProfile(namespace string) *operatorapi.SonataFlow {
	workflow := GetSonataFlow(SonataFlowSimpleOpsYamlCR, namespace)
	return workflow
}

func GetBasePlatformInReadyPhase(namespace string) *operatorapi.SonataFlowPlatform {
	return GetSonataFlowPlatformInReadyPhase(sonataFlowPlatformYamlCR, namespace)
}

func GetBasePlatformWithBaseImageInReadyPhase(namespace string) *operatorapi.SonataFlowPlatform {
	platform := GetBasePlatform()
	platform.Namespace = namespace
	platform.Status.Manager().MarkTrue(api.SucceedConditionType)
	platform.Spec.Build.Config.BaseImage = "quay.io/customx/custom-swf-builder:24.8.17"
	return platform
}

func GetBasePlatformWithDevBaseImageInReadyPhase(namespace string) *operatorapi.SonataFlowPlatform {
	platform := GetBasePlatform()
	platform.Namespace = namespace
	platform.Status.Manager().MarkTrue(api.SucceedConditionType)
	platform.Spec.DevMode.BaseImage = "quay.io/customgroup/custom-swf-builder-nightly:42.43.7"
	return platform
}

func GetBasePlatform() *operatorapi.SonataFlowPlatform {
	return getSonataFlowPlatform(sonataFlowPlatformYamlCR)
}

func GetPlatformMinikubeE2eTest() string {
	return e2eSamples + sonataFlowPlatformWithCacheMinikubeYamlCR
}

func GetPlatformOpenshiftE2eTest() string {
	return e2eSamples + sonataFlowPlatformForOpenshift
}

func GetSonataFlowE2eOrderProcessingFolder() string {
	return e2eSamples + sonataFlowOrderProcessingFolder
}

// getTestDataDir gets the testdata directory containing every sample out there from test/testdata.
// It should be used for every testing unit within the module.
func getTestDataDir() string {
	return path.Join(getProjectDir(), "test", "testdata")
}

func getBundleDir() string {
	return path.Join(getProjectDir(), manifestsPath)
}

func getProjectDir() string {
	// we only have to do this once
	if len(projectDir) > 0 {
		return projectDir
	}
	// file is the current caller relative filename (this file yaml.go) from GOPATH/src
	_, filename, _, _ := runtime.Caller(0)
	// remove the filename and the "test" directory
	filename = filepath.Dir(filepath.Dir(filename))
	wd, _ := os.Getwd()
	for {
		if strings.HasSuffix(wd, filename) {
			break
		}
		wd = filepath.Dir(wd)
	}
	projectDir = wd
	return projectDir
}
