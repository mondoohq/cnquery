// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"sync"

	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/utils/multierr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type mqlK8sNodeInternal struct {
	lock sync.Mutex
	obj  *corev1.Node
}

func initK8sNode(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	// we only look up the node, if we have been supplied by its name and nothing else
	raw, ok := args["name"]
	if !ok || len(args) != 1 {
		return args, nil, nil
	}
	name := raw.Value.(string)

	k8sRaw, err := CreateResource(runtime, "k8s", nil)
	if err != nil {
		return nil, nil, multierr.Wrap(err, "cannot get list of nodes")
	}
	k8s := k8sRaw.(*mqlK8s)

	// Only list nodes if the cache is empty
	if k8s.nodesByName == nil || len(k8s.nodesByName) == 0 {
		list := k8s.GetNodes()
		if list.Error != nil {
			return nil, nil, list.Error
		}
	}

	x, found := k8s.nodesByName[name]
	if !found {
		return nil, nil, errors.New("cannot find node " + name)
	}

	return nil, x, nil
}

func (k *mqlK8s) nodes() ([]interface{}, error) {
	k.mqlK8sInternal.nodesByName = make(map[string]*mqlK8sNode)
	return k8sResourceToMql(k.MqlRuntime, gvkString(corev1.SchemeGroupVersion.WithKind("nodes")), func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (interface{}, error) {
		ts := obj.GetCreationTimestamp()

		n, ok := obj.(*corev1.Node)
		if !ok {
			return nil, errors.New("not a k8s node")
		}

		nodeInfo, err := convert.JsonToDict(n.Status.NodeInfo)
		if err != nil {
			return nil, err
		}

		r, err := CreateResource(k.MqlRuntime, "k8s.node", map[string]*llx.RawData{
			"id":              llx.StringData(objIdFromK8sObj(obj, objT)),
			"uid":             llx.StringData(string(obj.GetUID())),
			"resourceVersion": llx.StringData(obj.GetResourceVersion()),
			"name":            llx.StringData(obj.GetName()),
			"kind":            llx.StringData(objT.GetKind()),
			"created":         llx.TimeData(ts.Time),
			"nodeInfo":        llx.DictData(nodeInfo),
			"kubeletPort":     llx.IntData(n.Status.DaemonEndpoints.KubeletEndpoint.Port),
		})
		if err != nil {
			return nil, err
		}

		r.(*mqlK8sNode).obj = n
		k.mqlK8sInternal.nodesByName[obj.GetName()] = r.(*mqlK8sNode)

		return r, nil
	})
}

func (k *mqlK8sNode) id() (string, error) {
	return k.Id.Data, nil
}

func (k *mqlK8sNode) annotations() (map[string]interface{}, error) {
	return convert.MapToInterfaceMap(k.obj.GetAnnotations()), nil
}

func (k *mqlK8sNode) labels() (map[string]interface{}, error) {
	return convert.MapToInterfaceMap(k.obj.GetLabels()), nil
}
