// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"sync"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type mqlK8sHorizontalpodautoscalerInternal struct {
	lock sync.Mutex
	obj  *autoscalingv2.HorizontalPodAutoscaler
}

func (k *mqlK8s) horizontalPodAutoscalers() ([]any, error) {
	return k8sResourceToMql(k.MqlRuntime, gvkString(autoscalingv2.SchemeGroupVersion.WithKind("horizontalpodautoscalers")), func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (any, error) {
		ts := obj.GetCreationTimestamp()

		hpa, ok := resource.(*autoscalingv2.HorizontalPodAutoscaler)
		if !ok {
			return nil, errors.New("not a k8s horizontalpodautoscaler")
		}

		r, err := CreateResource(k.MqlRuntime, "k8s.horizontalpodautoscaler", map[string]*llx.RawData{
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
		r.(*mqlK8sHorizontalpodautoscaler).obj = hpa
		return r, nil
	})
}

func (k *mqlK8sHorizontalpodautoscaler) manifest() (map[string]any, error) {
	return convert.JsonToDict(k.obj)
}

func (k *mqlK8sHorizontalpodautoscaler) spec() (map[string]any, error) {
	return convert.JsonToDict(k.obj.Spec)
}

func (k *mqlK8sHorizontalpodautoscaler) status() (map[string]any, error) {
	return convert.JsonToDict(k.obj.Status)
}

func (k *mqlK8sHorizontalpodautoscaler) id() (string, error) {
	return k.Id.Data, nil
}

func initK8sHorizontalpodautoscaler(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return initNamespacedResource[*mqlK8sHorizontalpodautoscaler](runtime, args, func(k *mqlK8s) *plugin.TValue[[]any] { return k.GetHorizontalPodAutoscalers() })
}

func (k *mqlK8sHorizontalpodautoscaler) annotations() (map[string]any, error) {
	return convert.MapToInterfaceMap(k.obj.GetAnnotations()), nil
}

func (k *mqlK8sHorizontalpodautoscaler) labels() (map[string]any, error) {
	return convert.MapToInterfaceMap(k.obj.GetLabels()), nil
}
