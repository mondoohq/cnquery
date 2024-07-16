// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"sync"

	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
)

type mqlK8sServiceaccountInternal struct {
	lock sync.Mutex
	obj  *corev1.ServiceAccount
}

func (k *mqlK8s) serviceaccounts() ([]interface{}, error) {
	return k8sResourceToMql(k.MqlRuntime, gvkString(corev1.SchemeGroupVersion.WithKind("serviceaccounts")), func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (interface{}, error) {
		ts := obj.GetCreationTimestamp()

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
			serviceAccount.AutomountServiceAccountToken = ptr.To(true)
		}

		r, err := CreateResource(k.MqlRuntime, "k8s.serviceaccount", map[string]*llx.RawData{
			"id":                           llx.StringData(objIdFromK8sObj(obj, objT)),
			"uid":                          llx.StringData(string(obj.GetUID())),
			"resourceVersion":              llx.StringData(obj.GetResourceVersion()),
			"name":                         llx.StringData(obj.GetName()),
			"namespace":                    llx.StringData(obj.GetNamespace()),
			"kind":                         llx.StringData(objT.GetKind()),
			"created":                      llx.TimeData(ts.Time),
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

func (k *mqlK8sServiceaccount) manifest() (map[string]interface{}, error) {
	manifest, err := convert.JsonToDict(k.obj)
	if err != nil {
		return nil, err
	}
	return manifest, nil
}

func (k *mqlK8sServiceaccount) id() (string, error) {
	return k.Id.Data, nil
}

func initK8sServiceaccount(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return initNamespacedResource[*mqlK8sServiceaccount](runtime, args, func(k *mqlK8s) *plugin.TValue[[]interface{}] { return k.GetServiceaccounts() })
}

func (k *mqlK8sServiceaccount) annotations() (map[string]interface{}, error) {
	return convert.MapToInterfaceMap(k.obj.GetAnnotations()), nil
}

func (k *mqlK8sServiceaccount) labels() (map[string]interface{}, error) {
	return convert.MapToInterfaceMap(k.obj.GetLabels()), nil
}
