// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"sync"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type mqlK8sPersistentvolumeclaimInternal struct {
	lock sync.Mutex
	obj  *corev1.PersistentVolumeClaim
}

func (k *mqlK8s) persistentVolumeClaims() ([]any, error) {
	return k8sResourceToMql(k.MqlRuntime, gvkString(corev1.SchemeGroupVersion.WithKind("persistentvolumeclaims")), func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (any, error) {
		ts := obj.GetCreationTimestamp()

		pvc, ok := resource.(*corev1.PersistentVolumeClaim)
		if !ok {
			return nil, errors.New("not a k8s persistentvolumeclaim")
		}

		r, err := CreateResource(k.MqlRuntime, "k8s.persistentvolumeclaim", map[string]*llx.RawData{
			"id":              llx.StringData(objIdFromK8sObj(obj, objT)),
			"uid":             llx.StringData(string(obj.GetUID())),
			"resourceVersion": llx.StringData(obj.GetResourceVersion()),
			"name":            llx.StringData(obj.GetName()),
			"namespace":       llx.StringData(obj.GetNamespace()),
			"kind":            llx.StringData(objT.GetKind()),
			"created":         llx.TimeData(ts.Time),
		})
		if err != nil {
			return nil, err
		}
		r.(*mqlK8sPersistentvolumeclaim).obj = pvc
		return r, nil
	})
}

func (k *mqlK8sPersistentvolumeclaim) manifest() (map[string]any, error) {
	return convert.JsonToDict(k.obj)
}

func (k *mqlK8sPersistentvolumeclaim) spec() (map[string]any, error) {
	return convert.JsonToDict(k.obj.Spec)
}

func (k *mqlK8sPersistentvolumeclaim) status() (map[string]any, error) {
	return convert.JsonToDict(k.obj.Status)
}

func (k *mqlK8sPersistentvolumeclaim) id() (string, error) {
	return k.Id.Data, nil
}

func initK8sPersistentvolumeclaim(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return initNamespacedResource[*mqlK8sPersistentvolumeclaim](runtime, args, func(k *mqlK8s) *plugin.TValue[[]any] { return k.GetPersistentVolumeClaims() })
}

func (k *mqlK8sPersistentvolumeclaim) annotations() (map[string]any, error) {
	return convert.MapToInterfaceMap(k.obj.GetAnnotations()), nil
}

func (k *mqlK8sPersistentvolumeclaim) labels() (map[string]any, error) {
	return convert.MapToInterfaceMap(k.obj.GetLabels()), nil
}
