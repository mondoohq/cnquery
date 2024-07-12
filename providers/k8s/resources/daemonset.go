// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"sync"

	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers/k8s/connection/shared/resources"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type mqlK8sDaemonsetInternal struct {
	lock sync.Mutex
	obj  *appsv1.DaemonSet
}

func (k *mqlK8s) daemonsets() ([]interface{}, error) {
	return k8sResourceToMql(k.MqlRuntime, appsv1.Resource("daemonsets").String(), func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (interface{}, error) {
		ts := obj.GetCreationTimestamp()

		r, err := CreateResource(k.MqlRuntime, "k8s.daemonset", map[string]*llx.RawData{
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

		d, ok := resource.(*appsv1.DaemonSet)
		if !ok {
			return nil, errors.New("not a k8s daemonset")
		}
		r.(*mqlK8sDaemonset).obj = d
		return r, nil
	})
}

func (k *mqlK8sDaemonset) manifest() (map[string]interface{}, error) {
	manifest, err := convert.JsonToDict(k.obj)
	if err != nil {
		return nil, err
	}
	return manifest, nil
}

func (k *mqlK8sDaemonset) podSpec() (map[string]interface{}, error) {
	podSpec, err := resources.GetPodSpec(k.obj)
	if err != nil {
		return nil, err
	}
	dict, err := convert.JsonToDict(podSpec)
	if err != nil {
		return nil, err
	}
	return dict, nil
}

func (k *mqlK8sDaemonset) id() (string, error) {
	return k.Id.Data, nil
}

func initK8sDaemonset(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return initNamespacedResource[*mqlK8sDaemonset](runtime, args, func(k *mqlK8s) *plugin.TValue[[]interface{}] { return k.GetDaemonsets() })
}

func (k *mqlK8sDaemonset) annotations() (map[string]interface{}, error) {
	return convert.MapToInterfaceMap(k.obj.GetAnnotations()), nil
}

func (k *mqlK8sDaemonset) labels() (map[string]interface{}, error) {
	return convert.MapToInterfaceMap(k.obj.GetLabels()), nil
}

func (k *mqlK8sDaemonset) initContainers() ([]interface{}, error) {
	return getContainers(k.obj, &k.obj.ObjectMeta, k.MqlRuntime, InitContainerType)
}

func (k *mqlK8sDaemonset) containers() ([]interface{}, error) {
	return getContainers(k.obj, &k.obj.ObjectMeta, k.MqlRuntime, ContainerContainerType)
}
