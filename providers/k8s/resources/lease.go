// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"sync"
	"time"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	coordinationv1 "k8s.io/api/coordination/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type mqlK8sLeaseInternal struct {
	lock sync.Mutex
	obj  *coordinationv1.Lease
}

func (k *mqlK8s) leases() ([]any, error) {
	return k8sResourceToMql(k.MqlRuntime, gvkString(coordinationv1.SchemeGroupVersion.WithKind("Lease")), func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (any, error) {
		ts := obj.GetCreationTimestamp()

		l, ok := resource.(*coordinationv1.Lease)
		if !ok {
			return nil, errors.New("not a k8s lease")
		}

		var holderIdentity, strategy, preferredHolder string
		if l.Spec.HolderIdentity != nil {
			holderIdentity = *l.Spec.HolderIdentity
		}
		if l.Spec.Strategy != nil {
			strategy = string(*l.Spec.Strategy)
		}
		if l.Spec.PreferredHolder != nil {
			preferredHolder = *l.Spec.PreferredHolder
		}

		var leaseDurationSeconds, leaseTransitions int64
		if l.Spec.LeaseDurationSeconds != nil {
			leaseDurationSeconds = int64(*l.Spec.LeaseDurationSeconds)
		}
		if l.Spec.LeaseTransitions != nil {
			leaseTransitions = int64(*l.Spec.LeaseTransitions)
		}

		var acquireTime, renewTime *time.Time
		if l.Spec.AcquireTime != nil {
			t := l.Spec.AcquireTime.Time
			acquireTime = &t
		}
		if l.Spec.RenewTime != nil {
			t := l.Spec.RenewTime.Time
			renewTime = &t
		}

		r, err := CreateResource(k.MqlRuntime, "k8s.lease", map[string]*llx.RawData{
			"id":                   llx.StringData(objIdFromK8sObj(obj, objT)),
			"uid":                  llx.StringData(string(obj.GetUID())),
			"resourceVersion":      llx.StringData(obj.GetResourceVersion()),
			"name":                 llx.StringData(obj.GetName()),
			"namespace":            llx.StringData(obj.GetNamespace()),
			"kind":                 llx.StringData(objT.GetKind()),
			"created":              llx.TimeData(ts.Time),
			"holderIdentity":       llx.StringData(holderIdentity),
			"leaseDurationSeconds": llx.IntData(leaseDurationSeconds),
			"acquireTime":          llx.TimeDataPtr(acquireTime),
			"renewTime":            llx.TimeDataPtr(renewTime),
			"leaseTransitions":     llx.IntData(leaseTransitions),
			"strategy":             llx.StringData(strategy),
			"preferredHolder":      llx.StringData(preferredHolder),
		})
		if err != nil {
			return nil, err
		}
		r.(*mqlK8sLease).obj = l
		return r, nil
	})
}

func (k *mqlK8sLease) id() (string, error) {
	return k.Id.Data, nil
}

func (k *mqlK8sLease) manifest() (map[string]any, error) {
	return convert.JsonToDict(k.obj)
}

func (k *mqlK8sLease) annotations() (map[string]any, error) {
	return convert.MapToInterfaceMap(k.obj.GetAnnotations()), nil
}

func (k *mqlK8sLease) labels() (map[string]any, error) {
	return convert.MapToInterfaceMap(k.obj.GetLabels()), nil
}

func initK8sLease(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return initNamespacedResource[*mqlK8sLease](runtime, args, func(k *mqlK8s) *plugin.TValue[[]any] { return k.GetLeases() })
}
