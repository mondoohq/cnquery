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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type mqlK8sPodInternal struct {
	lock sync.Mutex
	obj  runtime.Object
}

func (k *mqlK8sPod) getPod() (*corev1.Pod, error) {
	p, ok := k.obj.(*corev1.Pod)
	if ok {
		return p, nil
	}
	return nil, errors.New("invalid k8s pod")
}

func (k *mqlK8s) pods() ([]any, error) {
	return k8sResourceToMql(k.MqlRuntime, gvkString(corev1.SchemeGroupVersion.WithKind("pods")), func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (any, error) {
		ts := obj.GetCreationTimestamp()

		r, err := CreateResource(k.MqlRuntime, "k8s.pod", map[string]*llx.RawData{
			"id":              llx.StringData(objIdFromK8sObj(obj, objT)),
			"uid":             llx.StringData(string(obj.GetUID())),
			"resourceVersion": llx.StringData(obj.GetResourceVersion()),
			"name":            llx.StringData(obj.GetName()),
			"namespace":       llx.StringData(obj.GetNamespace()),
			"apiVersion":      llx.StringData(objT.GetAPIVersion()),
			"kind":            llx.StringData(objT.GetKind()),
			"created":         llx.TimeData(ts.Time),
		})
		if err != nil {
			return nil, err
		}

		r.(*mqlK8sPod).obj = resource
		return r, nil
	})
}

func (k *mqlK8sPod) manifest() (map[string]any, error) {
	manifest, err := convert.JsonToDict(k.obj)
	if err != nil {
		return nil, err
	}
	return manifest, nil
}

func (k *mqlK8sPod) podSpec() (map[string]any, error) {
	pod, err := k.getPod()
	if err != nil {
		return nil, err
	}
	podSpec, err := resources.GetPodSpec(pod)
	if err != nil {
		return nil, err
	}
	dict, err := convert.JsonToDict(podSpec)
	if err != nil {
		return nil, err
	}
	return dict, nil
}

func (k *mqlK8sPod) id() (string, error) {
	return k.Id.Data, nil
}

func initK8sPod(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return initNamespacedResource[*mqlK8sPod](runtime, args, func(k *mqlK8s) *plugin.TValue[[]any] { return k.GetPods() })
}

func (k *mqlK8sPod) initContainers() ([]any, error) {
	pod, err := k.getPod()
	if err != nil {
		return nil, err
	}
	return getContainers(pod, &pod.ObjectMeta, k.MqlRuntime, InitContainerType)
}

func (k *mqlK8sPod) ephemeralContainers() ([]any, error) {
	pod, err := k.getPod()
	if err != nil {
		return nil, err
	}
	return getContainers(pod, &pod.ObjectMeta, k.MqlRuntime, EphemeralContainerType)
}

func (k *mqlK8sPod) containers() ([]any, error) {
	pod, err := k.getPod()
	if err != nil {
		return nil, err
	}
	return getContainers(pod, &pod.ObjectMeta, k.MqlRuntime, ContainerContainerType)
}

func (k *mqlK8sPod) annotations() (map[string]any, error) {
	pod, err := k.getPod()
	if err != nil {
		return nil, err
	}
	return convert.MapToInterfaceMap(pod.GetAnnotations()), nil
}

func (k *mqlK8sPod) labels() (map[string]any, error) {
	pod, err := k.getPod()
	if err != nil {
		return nil, err
	}
	return convert.MapToInterfaceMap(pod.GetLabels()), nil
}

func (k *mqlK8sPod) node() (*mqlK8sNode, error) {
	pod, err := k.getPod()
	if err != nil {
		return nil, err
	}
	podSpec, err := resources.GetPodSpec(pod)
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
