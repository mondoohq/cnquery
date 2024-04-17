// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"sync"

	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/types"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type mqlK8sRbacClusterroleInternal struct {
	lock sync.Mutex
	obj  *rbacv1.ClusterRole
}

func (k *mqlK8s) clusterroles() ([]interface{}, error) {
	return k8sResourceToMql(k.MqlRuntime, "clusterroles", func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (interface{}, error) {
		ts := obj.GetCreationTimestamp()
		clusterRole, ok := resource.(*rbacv1.ClusterRole)
		if !ok {
			return nil, errors.New("not a k8s clusterrole")
		}

		rules, err := convert.JsonToDictSlice(clusterRole.Rules)
		if err != nil {
			return nil, err
		}

		aggregationRule, err := convert.JsonToDict(clusterRole.AggregationRule)
		if err != nil {
			return nil, err
		}

		r, err := CreateResource(k.MqlRuntime, "k8s.rbac.clusterrole", map[string]*llx.RawData{
			"id":              llx.StringData(objIdFromK8sObj(obj, objT)),
			"uid":             llx.StringData(string(obj.GetUID())),
			"resourceVersion": llx.StringData(obj.GetResourceVersion()),
			"name":            llx.StringData(obj.GetName()),
			"kind":            llx.StringData(objT.GetKind()),
			"created":         llx.TimeData(ts.Time),
			"rules":           llx.ArrayData(rules, types.Dict),
			"aggregationRule": llx.DictData(aggregationRule),
		})
		if err != nil {
			return nil, err
		}
		r.(*mqlK8sRbacClusterrole).obj = clusterRole
		return r, nil
	})
}

func (k *mqlK8sRbacClusterrole) manifest() (map[string]interface{}, error) {
	manifest, err := convert.JsonToDict(k.obj)
	if err != nil {
		return nil, err
	}
	return manifest, nil
}

func (k *mqlK8sRbacClusterrole) id() (string, error) {
	return k.Id.Data, nil
}

func initK8sRbacClusterrole(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return initResource[*mqlK8sRbacClusterrole](runtime, args, func(k *mqlK8s) *plugin.TValue[[]interface{}] { return k.GetClusterroles() })
}

func (k *mqlK8sRbacClusterrole) annotations() (map[string]interface{}, error) {
	return convert.MapToInterfaceMap(k.obj.GetAnnotations()), nil
}

func (k *mqlK8sRbacClusterrole) labels() (map[string]interface{}, error) {
	return convert.MapToInterfaceMap(k.obj.GetAnnotations()), nil
}
