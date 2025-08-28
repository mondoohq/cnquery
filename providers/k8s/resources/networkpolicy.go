// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"sync"

	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/util/convert"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type mqlK8sNetworkpolicyInternal struct {
	lock sync.Mutex
	obj  *networkingv1.NetworkPolicy
}

func (k *mqlK8s) networkPolicies() ([]any, error) {
	return k8sResourceToMql(k.MqlRuntime, gvkString(networkingv1.SchemeGroupVersion.WithKind("networkpolicies")), func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (any, error) {
		ts := obj.GetCreationTimestamp()

		networkPolicy, ok := resource.(*networkingv1.NetworkPolicy)
		if !ok {
			return nil, errors.New("not a k8s networkpolicy")
		}

		spec, err := convert.JsonToDict(networkPolicy.Spec)
		if err != nil {
			return nil, err
		}

		r, err := CreateResource(k.MqlRuntime, "k8s.networkpolicy", map[string]*llx.RawData{
			"id":              llx.StringData(objIdFromK8sObj(obj, objT)),
			"uid":             llx.StringData(string(obj.GetUID())),
			"resourceVersion": llx.StringData(obj.GetResourceVersion()),
			"name":            llx.StringData(obj.GetName()),
			"namespace":       llx.StringData(obj.GetNamespace()),
			"kind":            llx.StringData(objT.GetKind()),
			"created":         llx.TimeData(ts.Time),
			"spec":            llx.DictData(spec),
		})
		if err != nil {
			return nil, err
		}
		r.(*mqlK8sNetworkpolicy).obj = networkPolicy
		return r, nil
	})
}

func (k *mqlK8sNetworkpolicy) manifest() (map[string]any, error) {
	manifest, err := convert.JsonToDict(k.obj)
	if err != nil {
		return nil, err
	}
	return manifest, nil
}

func (k *mqlK8sNetworkpolicy) spec() (map[string]any, error) {
	dict, err := convert.JsonToDict(k.obj.Spec)
	if err != nil {
		return nil, err
	}
	return dict, nil
}

func (k *mqlK8sNetworkpolicy) id() (string, error) {
	return k.Id.Data, nil
}

func initK8sNetworkpolicy(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return initNamespacedResource[*mqlK8sNetworkpolicy](runtime, args, func(k *mqlK8s) *plugin.TValue[[]any] { return k.GetNetworkPolicies() })
}

func (k *mqlK8sNetworkpolicy) annotations() (map[string]any, error) {
	return convert.MapToInterfaceMap(k.obj.GetAnnotations()), nil
}

func (k *mqlK8sNetworkpolicy) labels() (map[string]any, error) {
	return convert.MapToInterfaceMap(k.obj.GetLabels()), nil
}
