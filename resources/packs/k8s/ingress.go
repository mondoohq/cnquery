package k8s

import (
	"errors"

	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func (k *mqlK8s) GetIngresses() ([]interface{}, error) {
	return k8sResourceToMql(k.MotorRuntime, "ingresses", func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (interface{}, error) {
		ts := obj.GetCreationTimestamp()

		manifest, err := core.JsonToDict(resource)
		if err != nil {
			return nil, err
		}

		ingress, ok := resource.(*networkingv1.Ingress)
		if !ok {
			return nil, errors.New("not a k8s ingress")
		}

		spec, err := core.JsonToDict(ingress.Spec)
		if err != nil {
			return nil, err
		}

		r, err := k.MotorRuntime.CreateResource("k8s.ingress",
			"id", objIdFromK8sObj(obj, objT),
			"uid", string(obj.GetUID()),
			"resourceVersion", obj.GetResourceVersion(),
			"name", obj.GetName(),
			"namespace", obj.GetNamespace(),
			"kind", objT.GetKind(),
			"created", &ts.Time,
			"manifest", manifest,
			"spec", spec,
		)
		if err != nil {
			return nil, err
		}
		r.MqlResource().Cache.Store("_resource", &resources.CacheEntry{Data: resource})
		return r, nil
	})
}

func (k *mqlK8sIngress) id() (string, error) {
	return k.Id()
}

func (p *mqlK8sIngress) init(args *resources.Args) (*resources.Args, K8sIngress, error) {
	return initNamespacedResource[K8sIngress](args, p.MotorRuntime, func(k K8s) ([]interface{}, error) { return k.Ingresses() })
}

func (k *mqlK8sIngress) GetAnnotations() (interface{}, error) {
	return k8sAnnotations(k.MqlResource())
}

func (k *mqlK8sIngress) GetLabels() (interface{}, error) {
	return k8sLabels(k.MqlResource())
}
