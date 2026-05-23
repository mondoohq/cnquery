// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"sync"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

type mqlK8sGatewayclassInternal struct {
	lock sync.Mutex
	obj  *gatewayv1.GatewayClass
}

func (k *mqlK8s) gatewayClasses() ([]any, error) {
	return k8sResourceToMql(k.MqlRuntime, gvkString(gatewayv1.SchemeGroupVersion.WithKind("gatewayclasses")), func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (any, error) {
		ts := obj.GetCreationTimestamp()

		gc, ok := resource.(*gatewayv1.GatewayClass)
		if !ok {
			return nil, errors.New("not a k8s gatewayclass")
		}

		description := ""
		if gc.Spec.Description != nil {
			description = *gc.Spec.Description
		}

		parametersRef, err := convert.JsonToDict(gc.Spec.ParametersRef)
		if err != nil {
			return nil, err
		}

		conditions, err := convert.JsonToDictSlice(gc.Status.Conditions)
		if err != nil {
			return nil, err
		}

		r, err := CreateResource(k.MqlRuntime, "k8s.gatewayclass", map[string]*llx.RawData{
			"id":              llx.StringData(objIdFromK8sObj(obj, objT)),
			"uid":             llx.StringData(string(obj.GetUID())),
			"resourceVersion": llx.StringData(obj.GetResourceVersion()),
			"name":            llx.StringData(obj.GetName()),
			"kind":            llx.StringData(objT.GetKind()),
			"created":         llx.TimeData(ts.Time),
			"controllerName":  llx.StringData(string(gc.Spec.ControllerName)),
			"description":     llx.StringData(description),
			"parametersRef":   llx.DictData(parametersRef),
			"conditions":      llx.ArrayData(conditions, types.Dict),
		})
		if err != nil {
			return nil, err
		}
		r.(*mqlK8sGatewayclass).obj = gc
		return r, nil
	})
}

func (k *mqlK8sGatewayclass) id() (string, error) {
	return k.Id.Data, nil
}

func (k *mqlK8sGatewayclass) manifest() (map[string]any, error) {
	return convert.JsonToDict(k.obj)
}

func (k *mqlK8sGatewayclass) annotations() (map[string]any, error) {
	return convert.MapToInterfaceMap(k.obj.GetAnnotations()), nil
}

func (k *mqlK8sGatewayclass) labels() (map[string]any, error) {
	return convert.MapToInterfaceMap(k.obj.GetLabels()), nil
}

func initK8sGatewayclass(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return initResource[*mqlK8sGatewayclass](runtime, args, func(k *mqlK8s) *plugin.TValue[[]any] { return k.GetGatewayClasses() })
}
