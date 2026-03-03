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

type mqlK8sPersistentvolumeInternal struct {
	lock sync.Mutex
	obj  *corev1.PersistentVolume
}

func (k *mqlK8s) persistentVolumes() ([]any, error) {
	return k8sResourceToMql(k.MqlRuntime, gvkString(corev1.SchemeGroupVersion.WithKind("persistentvolumes")), func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (any, error) {
		ts := obj.GetCreationTimestamp()

		pv, ok := resource.(*corev1.PersistentVolume)
		if !ok {
			return nil, errors.New("not a k8s persistentvolume")
		}

		r, err := CreateResource(k.MqlRuntime, "k8s.persistentvolume", map[string]*llx.RawData{
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
		r.(*mqlK8sPersistentvolume).obj = pv
		return r, nil
	})
}

func (k *mqlK8sPersistentvolume) manifest() (map[string]any, error) {
	return convert.JsonToDict(k.obj)
}

func (k *mqlK8sPersistentvolume) spec() (map[string]any, error) {
	return convert.JsonToDict(k.obj.Spec)
}

func (k *mqlK8sPersistentvolume) status() (map[string]any, error) {
	return convert.JsonToDict(k.obj.Status)
}

func (k *mqlK8sPersistentvolume) id() (string, error) {
	return k.Id.Data, nil
}

func initK8sPersistentvolume(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return initResource[*mqlK8sPersistentvolume](runtime, args, func(k *mqlK8s) *plugin.TValue[[]any] { return k.GetPersistentVolumes() })
}

func (k *mqlK8sPersistentvolume) annotations() (map[string]any, error) {
	return convert.MapToInterfaceMap(k.obj.GetAnnotations()), nil
}

func (k *mqlK8sPersistentvolume) labels() (map[string]any, error) {
	return convert.MapToInterfaceMap(k.obj.GetLabels()), nil
}
