package k8s

import (
	"errors"

	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
	rbacauthorizationv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func (k *mqlK8s) GetRoles() ([]interface{}, error) {
	return k8sResourceToMql(k.MotorRuntime, "roles", func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (interface{}, error) {
		ts := obj.GetCreationTimestamp()

		manifest, err := core.JsonToDict(resource)
		if err != nil {
			return nil, err
		}

		role, ok := resource.(*rbacauthorizationv1.Role)
		if !ok {
			return nil, errors.New("not a k8s role")
		}

		rules, err := core.JsonToDictSlice(role.Rules)
		if err != nil {
			return nil, err
		}

		r, err := k.MotorRuntime.CreateResource("k8s.rbac.role",
			"id", objIdFromK8sObj(obj, objT),
			"uid", string(obj.GetUID()),
			"resourceVersion", obj.GetResourceVersion(),
			"name", obj.GetName(),
			"namespace", obj.GetNamespace(),
			"kind", objT.GetKind(),
			"created", &ts.Time,
			"manifest", manifest,
			"rules", rules,
		)
		if err != nil {
			return nil, err
		}
		r.MqlResource().Cache.Store("_resource", &resources.CacheEntry{Data: resource})
		return r, nil
	})
}

func (k *mqlK8sRbacRole) id() (string, error) {
	return k.Id()
}

func (p *mqlK8sRbacRole) init(args *resources.Args) (*resources.Args, K8sRbacRole, error) {
	return initNamespacedResource[K8sRbacRole](args, p.MotorRuntime, func(k K8s) ([]interface{}, error) { return k.Roles() })
}

func (k *mqlK8sRbacRole) GetAnnotations() (interface{}, error) {
	return k8sAnnotations(k.MqlResource())
}

func (k *mqlK8sRbacRole) GetLabels() (interface{}, error) {
	return k8sLabels(k.MqlResource())
}
