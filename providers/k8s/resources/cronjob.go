// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"sync"

	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v10/providers/k8s/connection/shared/resources"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type mqlK8sCronjobInternal struct {
	lock sync.Mutex
	obj  *batchv1.CronJob
}

func (k *mqlK8s) cronjobs() ([]interface{}, error) {
	return k8sResourceToMql(k.MqlRuntime, "cronjobs", func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (interface{}, error) {
		ts := obj.GetCreationTimestamp()

		manifest, err := convert.JsonToDict(resource)
		if err != nil {
			return nil, err
		}

		podSpec, err := resources.GetPodSpec(resource)
		if err != nil {
			return nil, err
		}

		podSpecDict, err := convert.JsonToDict(podSpec)
		if err != nil {
			return nil, err
		}

		r, err := CreateResource(k.MqlRuntime, "k8s.cronjob", map[string]*llx.RawData{
			"id":              llx.StringData(objIdFromK8sObj(obj, objT)),
			"uid":             llx.StringData(string(obj.GetUID())),
			"resourceVersion": llx.StringData(obj.GetResourceVersion()),
			"name":            llx.StringData(obj.GetName()),
			"namespace":       llx.StringData(obj.GetNamespace()),
			"kind":            llx.StringData(objT.GetKind()),
			"created":         llx.TimeData(ts.Time),
			"manifest":        llx.DictData(manifest),
			"podSpec":         llx.DictData(podSpecDict),
		})
		if err != nil {
			return nil, err
		}

		cj, ok := resource.(*batchv1.CronJob)
		if !ok {
			return nil, errors.New("not a k8s cronjob")
		}
		r.(*mqlK8sCronjob).obj = cj
		return r, nil
	})
}

func (k *mqlK8sCronjob) id() (string, error) {
	return k.Id.Data, nil
}

func initK8sCronjob(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return initNamespacedResource[*mqlK8sCronjob](runtime, args, func(k *mqlK8s) *plugin.TValue[[]interface{}] { return k.GetCronjobs() })
}

func (k *mqlK8sCronjob) annotations() (map[string]interface{}, error) {
	return convert.MapToInterfaceMap(k.obj.GetAnnotations()), nil
}

func (k *mqlK8sCronjob) labels() (map[string]interface{}, error) {
	return convert.MapToInterfaceMap(k.obj.GetLabels()), nil
}

func (k *mqlK8sCronjob) initContainers() ([]interface{}, error) {
	return getContainers(k.obj, &k.obj.ObjectMeta, k.MqlRuntime, InitContainerType)
}

func (k *mqlK8sCronjob) containers() ([]interface{}, error) {
	return getContainers(k.obj, &k.obj.ObjectMeta, k.MqlRuntime, ContainerContainerType)
}
