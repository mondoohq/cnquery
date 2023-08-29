package resources

import (
	"errors"
	"sync"

	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers-sdk/v1/util/convert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
)

type mqlK8sServiceaccountInternal struct {
	lock sync.Mutex
	obj  *corev1.ServiceAccount
}

func (k *mqlK8s) serviceaccounts() ([]interface{}, error) {
	return k8sResourceToMql(k.MqlRuntime, "serviceaccounts", func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (interface{}, error) {
		ts := obj.GetCreationTimestamp()

		manifest, err := convert.JsonToDict(resource)
		if err != nil {
			return nil, err
		}

		serviceAccount, ok := resource.(*corev1.ServiceAccount)
		if !ok {
			return nil, errors.New("not a k8s serviceaccount")
		}

		secrets, err := convert.JsonToDictSlice(serviceAccount.Secrets)
		if err != nil {
			return nil, err
		}

		imagePullSecrets, err := convert.JsonToDictSlice(serviceAccount.ImagePullSecrets)
		if err != nil {
			return nil, err
		}

		// Implement k8s default of auto-mounting:
		// https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/#use-the-default-service-account-to-access-the-api-server
		// As discussed here, this behavior will not change for core/v1:
		// https://github.com/kubernetes/kubernetes/issues/57601
		if serviceAccount.AutomountServiceAccountToken == nil && objT.GetAPIVersion() == "v1" {
			serviceAccount.AutomountServiceAccountToken = pointer.Bool(true)
		}

		r, err := CreateResource(k.MqlRuntime, "k8s.serviceaccount", map[string]*llx.RawData{
			"id":                           llx.StringData(objIdFromK8sObj(obj, objT)),
			"uid":                          llx.StringData(string(obj.GetUID())),
			"resourceVersion":              llx.StringData(obj.GetResourceVersion()),
			"name":                         llx.StringData(obj.GetName()),
			"namespace":                    llx.StringData(obj.GetNamespace()),
			"kind":                         llx.StringData(objT.GetKind()),
			"created":                      llx.TimeData(ts.Time),
			"manifest":                     llx.DictData(manifest),
			"secrets":                      llx.DictData(secrets),
			"imagePullSecrets":             llx.DictData(imagePullSecrets),
			"automountServiceAccountToken": llx.BoolData(*serviceAccount.AutomountServiceAccountToken),
		})
		if err != nil {
			return nil, err
		}
		r.(*mqlK8sServiceaccount).obj = serviceAccount
		return r, nil
	})
}

func (k *mqlK8sServiceaccount) id() (string, error) {
	return k.Id.Data, nil
}

// func (p *mqlK8sServiceaccount) init(args *resources.Args) (*resources.Args, K8sServiceaccount, error) {
// 	return initNamespacedResource[K8sServiceaccount](args, p.MotorRuntime, func(k K8s) ([]interface{}, error) { return k.Serviceaccounts() })
// }

func (k *mqlK8sServiceaccount) annotations() (map[string]interface{}, error) {
	return convert.MapToInterfaceMap(k.obj.GetAnnotations()), nil
}

func (k *mqlK8sServiceaccount) labels() (map[string]interface{}, error) {
	return convert.MapToInterfaceMap(k.obj.GetLabels()), nil
}
