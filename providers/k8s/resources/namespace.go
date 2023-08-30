// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"sync"

	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers-sdk/v1/util/convert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type mqlK8sNamespaceInternal struct {
	lock sync.Mutex
	obj  *corev1.Namespace
}

func (k *mqlK8s) namespaces() ([]interface{}, error) {
	return k8sResourceToMql(k.MqlRuntime, "namespaces", func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (interface{}, error) {
		ts := obj.GetCreationTimestamp()

		manifest, err := convert.JsonToDict(resource)
		if err != nil {
			return nil, err
		}

		r, err := CreateResource(k.MqlRuntime, "k8s.namespace", map[string]*llx.RawData{
			"id":       llx.StringData(objIdFromK8sObj(obj, objT)),
			"uid":      llx.StringData(string(obj.GetUID())),
			"name":     llx.StringData(obj.GetName()),
			"created":  llx.TimeData(ts.Time),
			"manifest": llx.DictData(manifest),
			"kind":     llx.StringData(objT.GetKind()),
		})
		if err != nil {
			return nil, err
		}

		ns, ok := resource.(*corev1.Namespace)
		if !ok {
			return nil, errors.New("not a k8s namespace")
		}
		r.(*mqlK8sNamespace).obj = ns
		return r, nil
	})
}

func (k *mqlK8sNamespace) id() (string, error) {
	return k.Id.Data, nil
}

func (k *mqlK8sNamespace) annotations() (map[string]interface{}, error) {
	return convert.MapToInterfaceMap(k.obj.GetAnnotations()), nil
}

func (k *mqlK8sNamespace) labels() (map[string]interface{}, error) {
	return convert.MapToInterfaceMap(k.obj.GetLabels()), nil
}
