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
)

type mqlK8sServiceInternal struct {
	lock sync.Mutex
	obj  *corev1.Service
}

func (k *mqlK8s) services() ([]interface{}, error) {
	return k8sResourceToMql(k.MqlRuntime, gvkString(corev1.SchemeGroupVersion.WithKind("services")), getNamespaceScope(k.MqlRuntime), func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (interface{}, error) {
		ts := obj.GetCreationTimestamp()
		srv, ok := resource.(*corev1.Service)
		if !ok {
			return nil, errors.New("not a k8s service")
		}

		r, err := CreateResource(k.MqlRuntime, "k8s.service", map[string]*llx.RawData{
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
		r.(*mqlK8sService).obj = srv
		return r, nil
	})
}

func (k *mqlK8sService) manifest() (map[string]interface{}, error) {
	manifest, err := convert.JsonToDict(k.obj)
	if err != nil {
		return nil, err
	}
	return manifest, nil
}

func (k *mqlK8sService) spec() (map[string]interface{}, error) {
	dict, err := convert.JsonToDict(k.obj.Spec)
	if err != nil {
		return nil, err
	}
	return dict, nil
}

func (k *mqlK8sService) id() (string, error) {
	return k.Id.Data, nil
}

func initK8sService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return initNamespacedResource[*mqlK8sService](runtime, args, func(k *mqlK8s) *plugin.TValue[[]interface{}] { return k.GetServices() })
}

func (k *mqlK8sService) annotations() (map[string]interface{}, error) {
	return convert.MapToInterfaceMap(k.obj.GetAnnotations()), nil
}

func (k *mqlK8sService) labels() (map[string]interface{}, error) {
	return convert.MapToInterfaceMap(k.obj.GetLabels()), nil
}
