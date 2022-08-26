package k8s

import (
	"go.mondoo.com/cnquery/resources"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func (k *mqlK8s) GetNodes() ([]interface{}, error) {
	return k8sResourceToMql(k.MotorRuntime, "nodes.v1.", func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (interface{}, error) {
		r, err := k.MotorRuntime.CreateResource("k8s.node",
			"id", objIdFromK8sObj(obj, objT),
			"uid", string(obj.GetUID()),
			"resourceVersion", obj.GetResourceVersion(),
			"name", obj.GetName(),
			"kind", objT.GetKind(),
		)
		if err != nil {
			return nil, err
		}
		r.MqlResource().Cache.Store("_resource", &resources.CacheEntry{Data: resource})
		return r, nil
	})
}

func (k *mqlK8sNode) id() (string, error) {
	return k.Id()
}

func (k *mqlK8sNode) GetAnnotations() (interface{}, error) {
	return k8sAnnotations(k.MqlResource())
}

func (k *mqlK8sNode) GetLabels() (interface{}, error) {
	return k8sLabels(k.MqlResource())
}
