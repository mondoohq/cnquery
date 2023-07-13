package k8s

import (
	"errors"

	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func (k *mqlK8s) GetConfigmaps() ([]interface{}, error) {
	return k8sResourceToMql(k.MotorRuntime, "configmaps", func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (interface{}, error) {
		ts := obj.GetCreationTimestamp()

		manifest, err := core.JsonToDict(resource)
		if err != nil {
			return nil, err
		}

		cm, ok := resource.(*corev1.ConfigMap)
		if !ok {
			return nil, errors.New("not a k8s configmap")
		}

		r, err := k.MotorRuntime.CreateResource("k8s.configmap",
			"id", objIdFromK8sObj(obj, objT),
			"uid", string(obj.GetUID()),
			"resourceVersion", obj.GetResourceVersion(),
			"name", obj.GetName(),
			"namespace", obj.GetNamespace(),
			"kind", objT.GetKind(),
			"created", &ts.Time,
			"manifest", manifest,
			"data", core.StrMapToInterface(cm.Data),
		)
		if err != nil {
			return nil, err
		}
		r.MqlResource().Cache.Store("_resource", &resources.CacheEntry{Data: resource})
		return r, nil
	})
}

func (k *mqlK8sConfigmap) id() (string, error) {
	return k.Id()
}

func (p *mqlK8sConfigmap) init(args *resources.Args) (*resources.Args, K8sConfigmap, error) {
	return initNamespacedResource[K8sConfigmap](args, p.MotorRuntime, func(k K8s) ([]interface{}, error) { return k.Configmaps() })
}

func (k *mqlK8sConfigmap) GetAnnotations() (interface{}, error) {
	return k8sAnnotations(k.MqlResource())
}

func (k *mqlK8sConfigmap) GetLabels() (interface{}, error) {
	return k8sLabels(k.MqlResource())
}
