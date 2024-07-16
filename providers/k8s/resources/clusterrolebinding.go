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

type mqlK8sRbacClusterrolebindingInternal struct {
	lock sync.Mutex
	obj  *rbacv1.ClusterRoleBinding
}

func (k *mqlK8s) clusterrolebindings() ([]interface{}, error) {
	return k8sResourceToMql(k.MqlRuntime, gvkString(rbacv1.SchemeGroupVersion.WithKind("clusterrolebindings")), func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (interface{}, error) {
		ts := obj.GetCreationTimestamp()

		clusterRoleBinding, ok := resource.(*rbacv1.ClusterRoleBinding)
		if !ok {
			return nil, errors.New("not a k8s clusterrolebinding")
		}

		subjects, err := convert.JsonToDictSlice(clusterRoleBinding.Subjects)
		if err != nil {
			return nil, err
		}

		roleRef, err := convert.JsonToDict(clusterRoleBinding.RoleRef)
		if err != nil {
			return nil, err
		}

		r, err := CreateResource(k.MqlRuntime, "k8s.rbac.clusterrolebinding", map[string]*llx.RawData{
			"id":              llx.StringData(objIdFromK8sObj(obj, objT)),
			"uid":             llx.StringData(string(obj.GetUID())),
			"resourceVersion": llx.StringData(obj.GetResourceVersion()),
			"name":            llx.StringData(obj.GetName()),
			"kind":            llx.StringData(objT.GetKind()),
			"created":         llx.TimeData(ts.Time),
			"subjects":        llx.ArrayData(subjects, types.Dict),
			"roleRef":         llx.DictData(roleRef),
		})
		if err != nil {
			return nil, err
		}
		r.(*mqlK8sRbacClusterrolebinding).obj = clusterRoleBinding
		return r, nil
	})
}

func (k *mqlK8sRbacClusterrolebinding) manifest() (map[string]interface{}, error) {
	manifest, err := convert.JsonToDict(k.obj)
	if err != nil {
		return nil, err
	}
	return manifest, nil
}

func (k *mqlK8sRbacClusterrolebinding) id() (string, error) {
	return k.Id.Data, nil
}

func initK8sRbacClusterrolebinding(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return initResource[*mqlK8sRbacClusterrolebinding](runtime, args, func(k *mqlK8s) *plugin.TValue[[]interface{}] { return k.GetClusterrolebindings() })
}

func (k *mqlK8sRbacClusterrolebinding) annotations() (map[string]interface{}, error) {
	return convert.MapToInterfaceMap(k.obj.GetAnnotations()), nil
}

func (k *mqlK8sRbacClusterrolebinding) labels() (map[string]interface{}, error) {
	return convert.MapToInterfaceMap(k.obj.GetLabels()), nil
}
