package test

import (
	"bytes"
	"os"

	"github.com/kiegroup/container-builder/util/log"

	"k8s.io/apimachinery/pkg/util/yaml"

	operatorapi "github.com/kiegroup/kogito-serverless-operator/api/v1alpha08"
)

func GetKogitoServerlessWorkflow(path string, namespace string) *operatorapi.KogitoServerlessWorkflow {
	ksw := &operatorapi.KogitoServerlessWorkflow{}
	yamlFile, err := os.ReadFile(path)
	if err != nil {
		log.Errorf(err, "yamlFile.Get err   #%v ", err)
		panic(err)
	}
	// Important: Here we are reading the CR deployment file from a given path and creating a &operatorapi.KogitoServerlessWorkflow struct
	err = yaml.NewYAMLOrJSONDecoder(bytes.NewReader(yamlFile), 100).Decode(ksw)
	if err != nil {
		log.Errorf(err, "Unmarshal: #%v", err)
		panic(err)
	}
	log.Debugf("Successfully read KSW  #%v ", ksw)
	ksw.Namespace = namespace
	return ksw
}

func GetKogitoServerlessPlatform(path string) *operatorapi.KogitoServerlessPlatform {
	ksp := &operatorapi.KogitoServerlessPlatform{}
	yamlFile, err := os.ReadFile(path)
	if err != nil {
		log.Errorf(err, "yamlFile.Get err #%v ", err)
		panic(err)
	}
	// Important: Here we are reading the CR deployment file from a given path and creating a &operatorapi.KogitoServerlessPlatform struct
	err = yaml.NewYAMLOrJSONDecoder(bytes.NewReader(yamlFile), 100).Decode(ksp)
	if err != nil {
		log.Errorf(err, "Unmarshal: %v", err)
		panic(err)
	}
	log.Debugf("Successfully read KSP  #%v ", ksp)
	return ksp
}

func GetKogitoServerlessPlatformInReadyPhase(path string, namespace string) *operatorapi.KogitoServerlessPlatform {
	ksp := GetKogitoServerlessPlatform(path)
	ksp.Status.Phase = operatorapi.PlatformPhaseReady
	ksp.Namespace = namespace
	return ksp
}
