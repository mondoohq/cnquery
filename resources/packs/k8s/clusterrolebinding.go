package k8s

import (
	"errors"

	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
	rbacauthorizationv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func (k *mqlK8s) GetClusterrolebindings() ([]interface{}, error) {
	return k8sResourceToMql(k.MotorRuntime, "clusterrolebindings", func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (interface{}, error) {
		ts := obj.GetCreationTimestamp()

		manifest, err := core.JsonToDict(resource)
		if err != nil {
			return nil, err
		}

		clusterRoleBinding, ok := resource.(*rbacauthorizationv1.ClusterRoleBinding)
		if !ok {
			return nil, errors.New("not a k8s clusterrolebinding")
		}

		subjects, err := core.JsonToDictSlice(clusterRoleBinding.Subjects)
		if err != nil {
			return nil, err
		}

		roleRef, err := core.JsonToDict(clusterRoleBinding.RoleRef)
		if err != nil {
			return nil, err
		}

		r, err := k.MotorRuntime.CreateResource("k8s.rbac.clusterrolebinding",
			"id", objIdFromK8sObj(obj, objT),
			"uid", string(obj.GetUID()),
			"resourceVersion", obj.GetResourceVersion(),
			"name", obj.GetName(),
			"kind", objT.GetKind(),
			"created", &ts.Time,
			"manifest", manifest,
			"subjects", subjects,
			"roleRef", roleRef,
		)
		if err != nil {
			return nil, err
		}
		r.MqlResource().Cache.Store("_resource", &resources.CacheEntry{Data: resource})
		return r, nil
	})
}

func (k *mqlK8sRbacClusterrolebinding) id() (string, error) {
	return k.Id()
}

func (p *mqlK8sRbacClusterrolebinding) init(args *resources.Args) (*resources.Args, K8sRbacClusterrolebinding, error) {
	return initResource[K8sRbacClusterrolebinding](args, p.MotorRuntime, func(k K8s) ([]interface{}, error) { return k.Clusterrolebindings() })
}

func (k *mqlK8sRbacClusterrolebinding) GetAnnotations() (interface{}, error) {
	return k8sAnnotations(k.MqlResource())
}

func (k *mqlK8sRbacClusterrolebinding) GetLabels() (interface{}, error) {
	return k8sLabels(k.MqlResource())
}
