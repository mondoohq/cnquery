// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"sync"

	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type mqlK8sPodsecuritypolicyInternal struct {
	lock sync.Mutex
	obj  *policyv1beta1.PodSecurityPolicy
}

func (k *mqlK8s) podSecurityPolicies() ([]interface{}, error) {
	return k8sResourceToMql(k.MqlRuntime, policyv1beta1.Resource("podsecuritypolicies").String(), func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (interface{}, error) {
		ts := obj.GetCreationTimestamp()

		psp, ok := resource.(*policyv1beta1.PodSecurityPolicy)
		if !ok {
			return nil, errors.New("not a k8s podsecuritypolicy")
		}

		r, err := CreateResource(k.MqlRuntime, "k8s.podsecuritypolicy", map[string]*llx.RawData{
			"id":              llx.StringData(objIdFromK8sObj(obj, objT)),
			"uid":             llx.StringData(string(obj.GetUID())),
			"resourceVersion": llx.StringData(obj.GetResourceVersion()),
			"name":            llx.StringData(obj.GetName()),
			"kind":            llx.StringData(objT.GetKind()),
			"created":         llx.TimeData(ts.Time),
		})
		if err != nil {
			return nil, err
		}
		r.(*mqlK8sPodsecuritypolicy).obj = psp
		return r, nil
	})
}

func (k *mqlK8sPodsecuritypolicy) manifest() (map[string]interface{}, error) {
	manifest, err := convert.JsonToDict(k.obj)
	if err != nil {
		return nil, err
	}
	return manifest, nil
}

func (k *mqlK8sPodsecuritypolicy) spec() (map[string]interface{}, error) {
	dict, err := convert.JsonToDict(k.obj.Spec)
	if err != nil {
		return nil, err
	}
	return dict, nil
}

func (k *mqlK8sPodsecuritypolicy) id() (string, error) {
	return k.Id.Data, nil
}

func (k *mqlK8sPodsecuritypolicy) annotations() (map[string]interface{}, error) {
	return convert.MapToInterfaceMap(k.obj.GetAnnotations()), nil
}

func (k *mqlK8sPodsecuritypolicy) labels() (map[string]interface{}, error) {
	return convert.MapToInterfaceMap(k.obj.GetLabels()), nil
}
