// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"sync"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type mqlK8sAdmissionMutatingwebhookconfigurationInternal struct {
	lock sync.Mutex
	obj  *admissionregistrationv1.MutatingWebhookConfiguration
}

func (k *mqlK8s) mutatingWebhookConfigurations() ([]any, error) {
	return k8sResourceToMql(k.MqlRuntime, gvkString(admissionregistrationv1.SchemeGroupVersion.WithKind("MutatingWebhookConfiguration")), func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (any, error) {
		ts := obj.GetCreationTimestamp()

		mwc, ok := resource.(*admissionregistrationv1.MutatingWebhookConfiguration)
		if !ok {
			return nil, errors.New("not a k8s mutatingwebhookconfiguration")
		}

		r, err := CreateResource(k.MqlRuntime, "k8s.admission.mutatingwebhookconfiguration", map[string]*llx.RawData{
			"id":              llx.StringData(objIdFromK8sObj(obj, objT)),
			"uid":             llx.StringData(string(obj.GetUID())),
			"resourceVersion": llx.StringData(obj.GetResourceVersion()),
			"name":            llx.StringData(obj.GetName()),
			"kind":            llx.StringData(objT.GetKind()),
			"created":         llx.TimeData(ts.Time),
		})
		if err != nil {
			return nil, err
		}
		r.(*mqlK8sAdmissionMutatingwebhookconfiguration).obj = mwc
		return r, nil
	})
}

func (k *mqlK8sAdmissionMutatingwebhookconfiguration) id() (string, error) {
	return k.Id.Data, nil
}

func (k *mqlK8sAdmissionMutatingwebhookconfiguration) manifest() (map[string]any, error) {
	return convert.JsonToDict(k.obj)
}

func (k *mqlK8sAdmissionMutatingwebhookconfiguration) annotations() (map[string]any, error) {
	return convert.MapToInterfaceMap(k.obj.GetAnnotations()), nil
}

func (k *mqlK8sAdmissionMutatingwebhookconfiguration) labels() (map[string]any, error) {
	return convert.MapToInterfaceMap(k.obj.GetLabels()), nil
}

func (k *mqlK8sAdmissionMutatingwebhookconfiguration) webhooks() ([]any, error) {
	return convert.JsonToDictSlice(k.obj.Webhooks)
}

func initK8sAdmissionMutatingwebhookconfiguration(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return initResource[*mqlK8sAdmissionMutatingwebhookconfiguration](runtime, args, func(k *mqlK8s) *plugin.TValue[[]any] {
		return k.GetMutatingWebhookConfigurations()
	})
}

func (k *mqlK8sAdmissionMutatingwebhookconfiguration) ownerReferences() ([]any, error) {
	return k8sOwnerReferences(k.MqlRuntime, k.obj)
}

func (k *mqlK8sAdmissionMutatingwebhookconfiguration) managedFields() ([]any, error) {
	return k8sManagedFields(k.MqlRuntime, k.obj)
}
