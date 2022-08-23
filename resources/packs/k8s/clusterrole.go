package k8s

import (
	"errors"

	"go.mondoo.io/mondoo/resources"
	"go.mondoo.io/mondoo/resources/packs/core"
	rbacauthorizationv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func (k *mqlK8s) GetClusterroles() ([]interface{}, error) {
	return k8sResourceToMql(k.MotorRuntime, "clusterroles", func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (interface{}, error) {
		ts := obj.GetCreationTimestamp()

		manifest, err := core.JsonToDict(resource)
		if err != nil {
			return nil, err
		}

		clusterRole, ok := resource.(*rbacauthorizationv1.ClusterRole)
		if !ok {
			return nil, errors.New("not a k8s clusterrole")
		}

		rules, err := core.JsonToDictSlice(clusterRole.Rules)
		if err != nil {
			return nil, err
		}

		aggregationRule, err := core.JsonToDict(clusterRole.AggregationRule)
		if err != nil {
			return nil, err
		}

		r, err := k.MotorRuntime.CreateResource("k8s.rbac.clusterrole",
			"id", objIdFromK8sObj(obj, objT),
			"uid", string(obj.GetUID()),
			"resourceVersion", obj.GetResourceVersion(),
			"name", obj.GetName(),
			"kind", objT.GetKind(),
			"created", &ts.Time,
			"manifest", manifest,
			"rules", rules,
			"aggregationRule", aggregationRule,
		)
		if err != nil {
			return nil, err
		}
		r.MqlResource().Cache.Store("_resource", &resources.CacheEntry{Data: resource})
		return r, nil
	})
}

func (k *mqlK8sRbacClusterrole) id() (string, error) {
	return k.Id()
}

func (p *mqlK8sRbacClusterrole) init(args *resources.Args) (*resources.Args, K8sRbacClusterrole, error) {
	return initResource[K8sRbacClusterrole](args, p.MotorRuntime, func(k K8s) ([]interface{}, error) { return k.Clusterroles() })
}

func (k *mqlK8sRbacClusterrole) GetAnnotations() (interface{}, error) {
	return k8sAnnotations(k.MqlResource())
}

func (k *mqlK8sRbacClusterrole) GetLabels() (interface{}, error) {
	return k8sLabels(k.MqlResource())
}
