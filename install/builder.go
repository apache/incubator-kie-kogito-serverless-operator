package install

import (
	"context"
	v08 "github.com/davidesalerno/kogito-serverless-operator/api/v08"
	"github.com/davidesalerno/kogito-serverless-operator/utils/kubernetes"
	"github.com/davidesalerno/kogito-serverless-operator/utils/resources"
	"github.com/ricardozanini/kogito-builder/client"
	"k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
)

// ResourceCustomizer can be used to inject code that changes the objects before they are created.
type ResourceCustomizer func(object ctrl.Object) ctrl.Object

// IdentityResourceCustomizer is a ResourceCustomizer that does nothing.
var IdentityResourceCustomizer = func(object ctrl.Object) ctrl.Object {
	return object
}

// Resources installs named resources from the project resource directory.
func Resources(ctx context.Context, c client.Client, namespace string, force bool, customizer ResourceCustomizer, names ...string) error {
	return ResourcesOrCollect(ctx, c, namespace, nil, force, customizer, names...)
}

func ResourcesOrCollect(ctx context.Context, c client.Client, namespace string, collection *kubernetes.Collection,
	force bool, customizer ResourceCustomizer, names ...string) error {
	for _, name := range names {
		if err := ResourceOrCollect(ctx, c, namespace, collection, force, customizer, name); err != nil {
			return err
		}
	}
	return nil
}

// Resource installs a single named resource from the project resource directory.
func Resource(ctx context.Context, c client.Client, namespace string, force bool, customizer ResourceCustomizer, name string) error {
	return ResourceOrCollect(ctx, c, namespace, nil, force, customizer, name)
}

func ResourceOrCollect(ctx context.Context, c client.Client, namespace string, collection *kubernetes.Collection,
	force bool, customizer ResourceCustomizer, name string) error {

	content, err := resources.ResourceAsString(name)
	if err != nil {
		return err
	}

	obj, err := kubernetes.LoadResourceFromYaml(c.GetScheme(), content)
	if err != nil {
		return err
	}

	return ObjectOrCollect(ctx, c, namespace, collection, force, customizer(obj))
}

func ObjectOrCollect(ctx context.Context, c client.Client, namespace string, collection *kubernetes.Collection, force bool, obj ctrl.Object) error {
	if collection != nil {
		// Adding to the collection before setting the namespace
		collection.Add(obj)
		return nil
	}

	obj.SetNamespace(namespace)

	if obj.GetObjectKind().GroupVersionKind().Kind == "PersistentVolumeClaim" {
		if err := c.Create(ctx, obj); err != nil && !errors.IsAlreadyExists(err) {
			return err
		}
	}

	if force {
		if _, err := kubernetes.ReplaceResource(ctx, c, obj); err != nil {
			return err
		}
		// For some resources, also reset the status
		if obj.GetObjectKind().GroupVersionKind().Kind == v08.KogitoServerlessPlatformKind {
			if err := c.Status().Update(ctx, obj); err != nil {
				return err
			}
		}
		return nil
	}

	// Just try to create them
	return c.Create(ctx, obj)
}

func installBuilderServiceAccountRolesOpenShift(ctx context.Context, c client.Client, namespace string) error {
	return ResourcesOrCollect(ctx, c, namespace, nil, true, IdentityResourceCustomizer,
		"/builder/builder-service-account.yaml",
		"/builder/builder-role.yaml",
		"/builder/builder-role-binding.yaml",
		"/builder/builder-role-openshift.yaml",
		"/builder/builder-role-binding-openshift.yaml",
	)
}

func installBuilderServiceAccountRolesKubernetes(ctx context.Context, c client.Client, namespace string) error {
	return ResourcesOrCollect(ctx, c, namespace, nil, true, IdentityResourceCustomizer,
		"/builder/builder-service-account.yaml",
		"/builder/builder-role.yaml",
		"/builder/builder-role-binding.yaml",
	)
}

// BuilderServiceAccountRoles installs the builder service account and related roles in the given namespace.
func BuilderServiceAccountRoles(ctx context.Context, c client.Client, namespace string, cluster v08.PlatformCluster) error {
	if cluster == v08.PlatformClusterOpenShift {
		if err := installBuilderServiceAccountRolesOpenShift(ctx, c, namespace); err != nil {
			return err
		}
	} else {
		if err := installBuilderServiceAccountRolesKubernetes(ctx, c, namespace); err != nil {
			return err
		}
	}
	return nil
}
