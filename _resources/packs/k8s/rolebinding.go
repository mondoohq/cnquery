package k8s

import (
	"errors"

	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
	rbacauthorizationv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func (k *mqlK8s) GetRolebindings() ([]interface{}, error) {
	return k8sResourceToMql(k.MotorRuntime, "rolebinding", func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (interface{}, error) {
		ts := obj.GetCreationTimestamp()

		manifest, err := core.JsonToDict(resource)
		if err != nil {
			return nil, err
		}

		roleBinding, ok := resource.(*rbacauthorizationv1.RoleBinding)
		if !ok {
			return nil, errors.New("not a k8s rolebinding")
		}

		subjects, err := core.JsonToDictSlice(roleBinding.Subjects)
		if err != nil {
			return nil, err
		}

		roleRef, err := core.JsonToDict(roleBinding.RoleRef)
		if err != nil {
			return nil, err
		}

		r, err := k.MotorRuntime.CreateResource("k8s.rbac.rolebinding",
			"id", objIdFromK8sObj(obj, objT),
			"uid", string(obj.GetUID()),
			"resourceVersion", obj.GetResourceVersion(),
			"name", obj.GetName(),
			"namespace", obj.GetNamespace(),
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

func (k *mqlK8sRbacRolebinding) id() (string, error) {
	return k.Id()
}

func (p *mqlK8sRbacRolebinding) init(args *resources.Args) (*resources.Args, K8sRbacRolebinding, error) {
	return initNamespacedResource[K8sRbacRolebinding](args, p.MotorRuntime, func(k K8s) ([]interface{}, error) { return k.Rolebindings() })
}

func (k *mqlK8sRbacRolebinding) GetAnnotations() (interface{}, error) {
	return k8sAnnotations(k.MqlResource())
}

func (k *mqlK8sRbacRolebinding) GetLabels() (interface{}, error) {
	return k8sLabels(k.MqlResource())
}
