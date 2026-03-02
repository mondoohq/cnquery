// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/utils/multierr"
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

func (k *mqlK8s) nodes() ([]any, error) {
	k.mqlK8sInternal.nodesByName = make(map[string]*mqlK8sNode)
	return k8sResourceToMql(k.MqlRuntime, gvkString(corev1.SchemeGroupVersion.WithKind("nodes")), func(kind string, resource runtime.Object, obj metav1.Object, objT metav1.Type) (any, error) {
		ts := obj.GetCreationTimestamp()

		n, ok := obj.(*corev1.Node)
		if !ok {
			return nil, errors.New("not a k8s node")
		}

		nodeInfo, err := convert.JsonToDict(n.Status.NodeInfo)
		if err != nil {
			return nil, err
		}

		capacity, err := convert.JsonToDict(n.Status.Capacity)
		if err != nil {
			return nil, err
		}

		allocatable, err := convert.JsonToDict(n.Status.Allocatable)
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
			"capacity":        llx.DictData(capacity),
			"allocatable":     llx.DictData(allocatable),
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

func (k *mqlK8sNode) annotations() (map[string]any, error) {
	return convert.MapToInterfaceMap(k.obj.GetAnnotations()), nil
}

func (k *mqlK8sNode) labels() (map[string]any, error) {
	return convert.MapToInterfaceMap(k.obj.GetLabels()), nil
}

func (k *mqlK8sNode) taints() ([]any, error) {
	uid := string(k.obj.GetUID())
	res := make([]any, 0, len(k.obj.Spec.Taints))
	for _, t := range k.obj.Spec.Taints {
		var timeAdded *time.Time
		if t.TimeAdded != nil {
			ta := t.TimeAdded.Time
			timeAdded = &ta
		}
		r, err := CreateResource(k.MqlRuntime, "k8s.nodeTaint", map[string]*llx.RawData{
			"__id":      llx.StringData(fmt.Sprintf("%s/taint/%s/%s", uid, t.Key, t.Effect)),
			"key":       llx.StringData(t.Key),
			"value":     llx.StringData(t.Value),
			"effect":    llx.StringData(string(t.Effect)),
			"timeAdded": llx.TimeDataPtr(timeAdded),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}
	return res, nil
}

func (k *mqlK8sNode) conditions() ([]any, error) {
	uid := string(k.obj.GetUID())
	res := make([]any, 0, len(k.obj.Status.Conditions))
	for _, c := range k.obj.Status.Conditions {
		r, err := CreateResource(k.MqlRuntime, "k8s.nodeCondition", map[string]*llx.RawData{
			"__id":               llx.StringData(fmt.Sprintf("%s/condition/%s", uid, c.Type)),
			"type":               llx.StringData(string(c.Type)),
			"status":             llx.StringData(string(c.Status)),
			"lastHeartbeatTime":  llx.TimeData(c.LastHeartbeatTime.Time),
			"lastTransitionTime": llx.TimeData(c.LastTransitionTime.Time),
			"reason":             llx.StringData(c.Reason),
			"message":            llx.StringData(c.Message),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}
	return res, nil
}

func (k *mqlK8sNode) addresses() ([]any, error) {
	uid := string(k.obj.GetUID())
	res := make([]any, 0, len(k.obj.Status.Addresses))
	for _, a := range k.obj.Status.Addresses {
		r, err := CreateResource(k.MqlRuntime, "k8s.nodeAddress", map[string]*llx.RawData{
			"__id":    llx.StringData(fmt.Sprintf("%s/address/%s", uid, a.Type)),
			"type":    llx.StringData(string(a.Type)),
			"address": llx.StringData(a.Address),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}
	return res, nil
}
