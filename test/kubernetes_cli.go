package test

import (
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/kiegroup/kogito-serverless-operator/api/v1alpha08"
)

// NewKogitoClientBuilder creates a new fake.ClientBuilder with the right scheme references
func NewKogitoClientBuilder() *fake.ClientBuilder {
	s := scheme.Scheme
	utilruntime.Must(v1alpha08.AddToScheme(s))
	return fake.NewClientBuilder().WithScheme(s)
}
