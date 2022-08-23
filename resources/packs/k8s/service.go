package k8s

import (
	"errors"

	"go.mondoo.io/mondoo/resources"
	"go.mondoo.io/mondoo/resources/packs/core"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func (k *mqlK8s) GetServices() ([]interface{}, error) {
	return k8sResourceToMql(k.MotorRuntime, "services", func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (interface{}, error) {
		ts := obj.GetCreationTimestamp()

		manifest, err := core.JsonToDict(resource)
		if err != nil {
			return nil, err
		}

		srv, ok := resource.(*corev1.Service)
		if !ok {
			return nil, errors.New("not a k8s service")
		}

		spec, err := core.JsonToDict(srv.Spec)
		if err != nil {
			return nil, err
		}

		r, err := k.MotorRuntime.CreateResource("k8s.service",
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

func (k *mqlK8sService) id() (string, error) {
	return k.Id()
}

func (p *mqlK8sService) init(args *resources.Args) (*resources.Args, K8sService, error) {
	return initNamespacedResource[K8sService](args, p.MotorRuntime, func(k K8s) ([]interface{}, error) { return k.Services() })
}

func (k *mqlK8sService) GetAnnotations() (interface{}, error) {
	return k8sAnnotations(k.MqlResource())
}

func (k *mqlK8sService) GetLabels() (interface{}, error) {
	return k8sLabels(k.MqlResource())
}
