// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"sync"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/k8s/connection/shared/resources"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type mqlK8sStatefulsetInternal struct {
	lock sync.Mutex
	obj  runtime.Object
}

func (k *mqlK8sStatefulset) getStatefulSet() (*appsv1.StatefulSet, error) {
	s, ok := k.obj.(*appsv1.StatefulSet)
	if ok {
		return s, nil
	}
	return nil, errors.New("invalid k8s statefulset")
}

func (k *mqlK8s) statefulsets() ([]any, error) {
	return k8sResourceToMql(k.MqlRuntime, gvkString(appsv1.SchemeGroupVersion.WithKind("statefulsets")), func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (any, error) {
		ts := obj.GetCreationTimestamp()

		r, err := CreateResource(k.MqlRuntime, "k8s.statefulset", map[string]*llx.RawData{
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

		r.(*mqlK8sStatefulset).obj = resource
		return r, nil
	})
}

func (k *mqlK8sStatefulset) manifest() (map[string]any, error) {
	manifest, err := convert.JsonToDict(k.obj)
	if err != nil {
		return nil, err
	}
	return manifest, nil
}

func (k *mqlK8sStatefulset) podSpec() (map[string]any, error) {
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

func (k *mqlK8sStatefulset) id() (string, error) {
	return k.Id.Data, nil
}

func initK8sStatefulset(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return initNamespacedResource[*mqlK8sStatefulset](runtime, args, func(k *mqlK8s) *plugin.TValue[[]any] { return k.GetStatefulsets() })
}

func (k *mqlK8sStatefulset) annotations() (map[string]any, error) {
	s, err := k.getStatefulSet()
	if err != nil {
		return nil, err
	}
	return convert.MapToInterfaceMap(s.GetAnnotations()), nil
}

func (k *mqlK8sStatefulset) labels() (map[string]any, error) {
	s, err := k.getStatefulSet()
	if err != nil {
		return nil, err
	}
	return convert.MapToInterfaceMap(s.GetLabels()), nil
}

func (k *mqlK8sStatefulset) initContainers() ([]any, error) {
	s, err := k.getStatefulSet()
	if err != nil {
		return nil, err
	}
	return getContainers(s, &s.ObjectMeta, k.MqlRuntime, InitContainerType)
}

func (k *mqlK8sStatefulset) containers() ([]any, error) {
	s, err := k.getStatefulSet()
	if err != nil {
		return nil, err
	}
	return getContainers(s, &s.ObjectMeta, k.MqlRuntime, ContainerContainerType)
}
