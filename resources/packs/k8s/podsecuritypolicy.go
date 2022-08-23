package k8s

import (
	"errors"

	"go.mondoo.io/mondoo/resources"
	"go.mondoo.io/mondoo/resources/packs/core"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func (k *mqlK8s) GetPodSecurityPolicies() ([]interface{}, error) {
	return k8sResourceToMql(k.MotorRuntime, "podsecuritypolicies", func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (interface{}, error) {
		ts := obj.GetCreationTimestamp()

		manifest, err := core.JsonToDict(resource)
		if err != nil {
			return nil, err
		}

		psp, ok := resource.(*policyv1beta1.PodSecurityPolicy)
		if !ok {
			return nil, errors.New("not a k8s podsecuritypolicy")
		}

		spec, err := core.JsonToDict(psp.Spec)
		if err != nil {
			return nil, err
		}

		r, err := k.MotorRuntime.CreateResource("k8s.podsecuritypolicy",
			"id", objIdFromK8sObj(obj, objT),
			"uid", string(obj.GetUID()),
			"resourceVersion", obj.GetResourceVersion(),
			"name", obj.GetName(),
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

func (k *mqlK8sPodsecuritypolicy) id() (string, error) {
	return k.Id()
}

func (k *mqlK8sPodsecuritypolicy) GetAnnotations() (interface{}, error) {
	return k8sAnnotations(k.MqlResource())
}

func (k *mqlK8sPodsecuritypolicy) GetLabels() (interface{}, error) {
	return k8sLabels(k.MqlResource())
}
