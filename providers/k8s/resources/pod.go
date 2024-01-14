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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type mqlK8sPodInternal struct {
	lock sync.Mutex
	obj  *corev1.Pod
}

func (k *mqlK8s) pods() ([]interface{}, error) {
	return k8sResourceToMql(k.MqlRuntime, "pods.v1.", func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (interface{}, error) {
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

		r, err := CreateResource(k.MqlRuntime, "k8s.pod", map[string]*llx.RawData{
			"id":              llx.StringData(objIdFromK8sObj(obj, objT)),
			"uid":             llx.StringData(string(obj.GetUID())),
			"resourceVersion": llx.StringData(obj.GetResourceVersion()),
			"name":            llx.StringData(obj.GetName()),
			"namespace":       llx.StringData(obj.GetNamespace()),
			"apiVersion":      llx.StringData(objT.GetAPIVersion()),
			"kind":            llx.StringData(objT.GetKind()),
			"created":         llx.TimeData(ts.Time),
			"podSpec":         llx.DictData(podSpecDict),
			"manifest":        llx.DictData(manifest),
		})
		if err != nil {
			return nil, err
		}

		p, ok := resource.(*corev1.Pod)
		if !ok {
			return nil, errors.New("not a k8s pod")
		}
		r.(*mqlK8sPod).obj = p
		return r, nil
	})
}

func (k *mqlK8sPod) id() (string, error) {
	return k.Id.Data, nil
}

func initK8sPod(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return initNamespacedResource[*mqlK8sPod](runtime, args, func(k *mqlK8s) *plugin.TValue[[]interface{}] { return k.GetPods() })
}

func (k *mqlK8sPod) initContainers() ([]interface{}, error) {
	return getContainers(k.obj, &k.obj.ObjectMeta, k.MqlRuntime, InitContainerType)
}

func (k *mqlK8sPod) ephemeralContainers() ([]interface{}, error) {
	return getContainers(k.obj, &k.obj.ObjectMeta, k.MqlRuntime, EphemeralContainerType)
}

func (k *mqlK8sPod) containers() ([]interface{}, error) {
	return getContainers(k.obj, &k.obj.ObjectMeta, k.MqlRuntime, ContainerContainerType)
}

func (k *mqlK8sPod) annotations() (map[string]interface{}, error) {
	return convert.MapToInterfaceMap(k.obj.GetAnnotations()), nil
}

func (k *mqlK8sPod) labels() (map[string]interface{}, error) {
	return convert.MapToInterfaceMap(k.obj.GetLabels()), nil
}

func (k *mqlK8sPod) node() (*mqlK8sNode, error) {
	podSpec, err := resources.GetPodSpec(k.obj)
	if err != nil {
		return nil, err
	}

	node, err := NewResource(k.MqlRuntime, "k8s.node", map[string]*llx.RawData{
		"name": llx.StringData(podSpec.NodeName),
	})
	if err != nil {
		return nil, err
	}

	return node.(*mqlK8sNode), nil
}
