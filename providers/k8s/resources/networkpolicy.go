package resources

import (
	"errors"
	"sync"

	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers-sdk/v1/util/convert"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type mqlK8sNetworkpolicyInternal struct {
	lock sync.Mutex
	obj  *networkingv1.NetworkPolicy
}

func (k *mqlK8s) networkPolicies() ([]interface{}, error) {
	return k8sResourceToMql(k.MqlRuntime, "networkpolicies", func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (interface{}, error) {
		ts := obj.GetCreationTimestamp()

		manifest, err := convert.JsonToDict(resource)
		if err != nil {
			return nil, err
		}

		networkPolicy, ok := resource.(*networkingv1.NetworkPolicy)
		if !ok {
			return nil, errors.New("not a k8s networkpolicy")
		}

		spec, err := convert.JsonToDict(networkPolicy.Spec)
		if err != nil {
			return nil, err
		}

		r, err := CreateResource(k.MqlRuntime, "k8s.networkpolicy", map[string]*llx.RawData{
			"id":              llx.StringData(objIdFromK8sObj(obj, objT)),
			"uid":             llx.StringData(string(obj.GetUID())),
			"resourceVersion": llx.StringData(obj.GetResourceVersion()),
			"name":            llx.StringData(obj.GetName()),
			"namespace":       llx.StringData(obj.GetNamespace()),
			"kind":            llx.StringData(objT.GetKind()),
			"created":         llx.TimeData(ts.Time),
			"manifest":        llx.DictData(manifest),
			"spec":            llx.DictData(spec),
		})
		if err != nil {
			return nil, err
		}
		r.(*mqlK8sNetworkpolicy).obj = networkPolicy
		return r, nil
	})
}

func (k *mqlK8sNetworkpolicy) id() (string, error) {
	return k.Id.Data, nil
}

// func (p *mqlK8sNetworkpolicy) init(args *resources.Args) (*resources.Args, K8sNetworkpolicy, error) {
// 	return initNamespacedResource[K8sNetworkpolicy](args, p.MotorRuntime, func(k K8s) ([]interface{}, error) { return k.NetworkPolicies() })
// }

func (k *mqlK8sNetworkpolicy) annotations() (map[string]interface{}, error) {
	return convert.MapToInterfaceMap(k.obj.GetAnnotations()), nil
}

func (k *mqlK8sNetworkpolicy) labels() (map[string]interface{}, error) {
	return convert.MapToInterfaceMap(k.obj.GetLabels()), nil
}
