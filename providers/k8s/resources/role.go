// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"sync"

	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v10/types"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type mqlK8sRbacRoleInternal struct {
	lock sync.Mutex
	obj  *rbacv1.Role
}

func (k *mqlK8s) roles() ([]interface{}, error) {
	return k8sResourceToMql(k.MqlRuntime, "roles", func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (interface{}, error) {
		ts := obj.GetCreationTimestamp()

		manifest, err := convert.JsonToDict(resource)
		if err != nil {
			return nil, err
		}

		role, ok := resource.(*rbacv1.Role)
		if !ok {
			return nil, errors.New("not a k8s role")
		}

		rules, err := convert.JsonToDictSlice(role.Rules)
		if err != nil {
			return nil, err
		}

		r, err := CreateResource(k.MqlRuntime, "k8s.rbac.role", map[string]*llx.RawData{
			"id":              llx.StringData(objIdFromK8sObj(obj, objT)),
			"uid":             llx.StringData(string(obj.GetUID())),
			"resourceVersion": llx.StringData(obj.GetResourceVersion()),
			"name":            llx.StringData(obj.GetName()),
			"namespace":       llx.StringData(obj.GetNamespace()),
			"kind":            llx.StringData(objT.GetKind()),
			"created":         llx.TimeData(ts.Time),
			"manifest":        llx.DictData(manifest),
			"rules":           llx.ArrayData(rules, types.Dict),
		})
		if err != nil {
			return nil, err
		}
		r.(*mqlK8sRbacRole).obj = role
		return r, nil
	})
}

func (k *mqlK8sRbacRole) id() (string, error) {
	return k.Id.Data, nil
}

func initK8sRbacRole(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return initNamespacedResource[*mqlK8sRbacRole](runtime, args, func(k *mqlK8s) *plugin.TValue[[]interface{}] { return k.GetRoles() })
}

func (k *mqlK8sRbacRole) annotations() (map[string]interface{}, error) {
	return convert.MapToInterfaceMap(k.obj.GetAnnotations()), nil
}

func (k *mqlK8sRbacRole) labels() (map[string]interface{}, error) {
	return convert.MapToInterfaceMap(k.obj.GetLabels()), nil
}
