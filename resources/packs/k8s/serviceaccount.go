package k8s

import (
	"errors"

	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func (k *mqlK8s) GetServiceaccounts() ([]interface{}, error) {
	return k8sResourceToMql(k.MotorRuntime, "serviceaccounts", func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (interface{}, error) {
		ts := obj.GetCreationTimestamp()

		manifest, err := core.JsonToDict(resource)
		if err != nil {
			return nil, err
		}

		serviceAccount, ok := resource.(*corev1.ServiceAccount)
		if !ok {
			return nil, errors.New("not a k8s serviceaccount")
		}

		secrets, err := core.JsonToDictSlice(serviceAccount.Secrets)
		if err != nil {
			return nil, err
		}

		imagePullSecrets, err := core.JsonToDictSlice(serviceAccount.ImagePullSecrets)
		if err != nil {
			return nil, err
		}

		r, err := k.MotorRuntime.CreateResource("k8s.serviceaccount",
			"id", objIdFromK8sObj(obj, objT),
			"uid", string(obj.GetUID()),
			"resourceVersion", obj.GetResourceVersion(),
			"name", obj.GetName(),
			"namespace", obj.GetNamespace(),
			"kind", objT.GetKind(),
			"created", &ts.Time,
			"manifest", manifest,
			"secrets", secrets,
			"imagePullSecrets", imagePullSecrets,
			"automountServiceAccountToken", core.ToBool(serviceAccount.AutomountServiceAccountToken),
		)
		if err != nil {
			return nil, err
		}
		r.MqlResource().Cache.Store("_resource", &resources.CacheEntry{Data: resource})
		return r, nil
	})
}

func (k *mqlK8sServiceaccount) id() (string, error) {
	return k.Id()
}

func (p *mqlK8sServiceaccount) init(args *resources.Args) (*resources.Args, K8sServiceaccount, error) {
	return initNamespacedResource[K8sServiceaccount](args, p.MotorRuntime, func(k K8s) ([]interface{}, error) { return k.Serviceaccounts() })
}

func (k *mqlK8sServiceaccount) GetAnnotations() (interface{}, error) {
	return k8sAnnotations(k.MqlResource())
}

func (k *mqlK8sServiceaccount) GetLabels() (interface{}, error) {
	return k8sLabels(k.MqlResource())
}
