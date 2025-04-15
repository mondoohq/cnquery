// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"sync"

	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v12/providers/k8s/connection/shared/resources"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type mqlK8sCronjobInternal struct {
	lock sync.Mutex
	obj  runtime.Object
}

func (k *mqlK8sCronjob) getCronJob() (*batchv1.CronJob, error) {
	cj, ok := k.obj.(*batchv1.CronJob)
	if ok {
		return cj, nil
	}
	return nil, errors.New("invalid k8s cronjob")
}

func (k *mqlK8s) cronjobs() ([]any, error) {
	return k8sResourceToMql(k.MqlRuntime, gvkString(batchv1.SchemeGroupVersion.WithKind("cronjobs")), func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (any, error) {
		ts := obj.GetCreationTimestamp()

		r, err := CreateResource(k.MqlRuntime, "k8s.cronjob", map[string]*llx.RawData{
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

		r.(*mqlK8sCronjob).obj = resource
		return r, nil
	})
}

func (k *mqlK8sCronjob) manifest() (map[string]any, error) {
	manifest, err := convert.JsonToDict(k.obj)
	if err != nil {
		return nil, err
	}
	return manifest, nil
}

func (k *mqlK8sCronjob) podSpec() (map[string]any, error) {
	podSpec, err := resources.GetPodSpec(k.obj)
	if err != nil {
		return nil, err
	}
	dict, err := convert.JsonToDict(podSpec)
	if err != nil {
		return nil, err
	}
	return dict, nil
}

func (k *mqlK8sCronjob) id() (string, error) {
	return k.Id.Data, nil
}

func initK8sCronjob(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return initNamespacedResource[*mqlK8sCronjob](runtime, args, func(k *mqlK8s) *plugin.TValue[[]any] { return k.GetCronjobs() })
}

func (k *mqlK8sCronjob) annotations() (map[string]any, error) {
	// Get the CronJob object
	cj, err := k.getCronJob()
	if err != nil {
		return nil, err
	}
	return convert.MapToInterfaceMap(cj.GetAnnotations()), nil
}

func (k *mqlK8sCronjob) labels() (map[string]any, error) {
	cj, err := k.getCronJob()
	if err != nil {
		return nil, err
	}
	return convert.MapToInterfaceMap(cj.GetLabels()), nil
}

func (k *mqlK8sCronjob) initContainers() ([]any, error) {
	// Get the CronJob object
	cj, err := k.getCronJob()
	if err != nil {
		return nil, err
	}
	return getContainers(cj, &cj.ObjectMeta, k.MqlRuntime, InitContainerType)
}

func (k *mqlK8sCronjob) containers() ([]any, error) {
	cj, err := k.getCronJob()
	if err != nil {
		return nil, err
	}
	return getContainers(cj, &cj.ObjectMeta, k.MqlRuntime, ContainerContainerType)
}
