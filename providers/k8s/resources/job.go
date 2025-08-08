// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"sync"

	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers/k8s/connection/shared/resources"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type mqlK8sJobInternal struct {
	lock sync.Mutex
	obj  runtime.Object
}

func (k *mqlK8sJob) getJob() (*batchv1.Job, error) {
	j, ok := k.obj.(*batchv1.Job)
	if ok {
		return j, nil
	}
	return nil, errors.New("invalid k8s job")
}

func (k *mqlK8s) jobs() ([]interface{}, error) {
	return k8sResourceToMql(k.MqlRuntime, gvkString(batchv1.SchemeGroupVersion.WithKind("jobs")), func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (interface{}, error) {
		ts := obj.GetCreationTimestamp()

		r, err := CreateResource(k.MqlRuntime, "k8s.job", map[string]*llx.RawData{
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

		r.(*mqlK8sJob).obj = resource
		return r, nil
	})
}

func (k *mqlK8sJob) manifest() (map[string]interface{}, error) {
	manifest, err := convert.JsonToDict(k.obj)
	if err != nil {
		return nil, err
	}
	return manifest, nil
}

func (k *mqlK8sJob) podSpec() (map[string]interface{}, error) {
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

func (k *mqlK8sJob) id() (string, error) {
	return k.Id.Data, nil
}

func initK8sJob(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return initNamespacedResource[*mqlK8sJob](runtime, args, func(k *mqlK8s) *plugin.TValue[[]interface{}] { return k.GetJobs() })
}

func (k *mqlK8sJob) annotations() (map[string]interface{}, error) {
	j, err := k.getJob()
	if err != nil {
		return nil, err
	}
	return convert.MapToInterfaceMap(j.GetAnnotations()), nil
}

func (k *mqlK8sJob) labels() (map[string]interface{}, error) {
	j, err := k.getJob()
	if err != nil {
		return nil, err
	}
	return convert.MapToInterfaceMap(j.GetLabels()), nil
}

func (k *mqlK8sJob) initContainers() ([]interface{}, error) {
	j, err := k.getJob()
	if err != nil {
		return nil, err
	}
	return getContainers(j, &j.ObjectMeta, k.MqlRuntime, InitContainerType)
}

func (k *mqlK8sJob) containers() ([]interface{}, error) {
	j, err := k.getJob()
	if err != nil {
		return nil, err
	}
	return getContainers(j, &j.ObjectMeta, k.MqlRuntime, ContainerContainerType)
}
