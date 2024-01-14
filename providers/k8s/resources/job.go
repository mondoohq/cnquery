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

type mqlK8sJobInternal struct {
	lock sync.Mutex
	obj  *batchv1.Job
}

func (k *mqlK8s) jobs() ([]interface{}, error) {
	return k8sResourceToMql(k.MqlRuntime, "jobs", func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (interface{}, error) {
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

		r, err := CreateResource(k.MqlRuntime, "k8s.job", map[string]*llx.RawData{
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

		j, ok := resource.(*batchv1.Job)
		if !ok {
			return nil, errors.New("not a k8s job")
		}
		r.(*mqlK8sJob).obj = j
		return r, nil
	})
}

func (k *mqlK8sJob) id() (string, error) {
	return k.Id.Data, nil
}

func initK8sJob(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return initNamespacedResource[*mqlK8sJob](runtime, args, func(k *mqlK8s) *plugin.TValue[[]interface{}] { return k.GetJobs() })
}

func (k *mqlK8sJob) annotations() (map[string]interface{}, error) {
	return convert.MapToInterfaceMap(k.obj.GetAnnotations()), nil
}

func (k *mqlK8sJob) labels() (map[string]interface{}, error) {
	return convert.MapToInterfaceMap(k.obj.GetLabels()), nil
}

func (k *mqlK8sJob) initContainers() ([]interface{}, error) {
	return getContainers(k.obj, &k.obj.ObjectMeta, k.MqlRuntime, InitContainerType)
}

func (k *mqlK8sJob) containers() ([]interface{}, error) {
	return getContainers(k.obj, &k.obj.ObjectMeta, k.MqlRuntime, ContainerContainerType)
}
