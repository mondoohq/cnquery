// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"time"

	"github.com/stackitcloud/stackit-sdk-go/services/ske"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/types"
)

func (r *mqlStackitSke) clusters() ([]any, error) {
	c := conn(r.MqlRuntime)
	client, err := c.SKE()
	if err != nil {
		return nil, err
	}
	resp, err := client.ListClustersExecute(bgctx(), c.ProjectID(), c.Region())
	if err != nil {
		if isAccessDenied(err) {
			return []any{}, nil
		}
		return nil, err
	}
	items, _ := resp.GetItemsOk()
	out := make([]any, 0, len(items))
	for i := range items {
		res, err := buildSkeCluster(r.MqlRuntime, &items[i])
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

// mqlStackitSkeClusterInternal caches the raw SKE Nodepool slice so the
// computed nodePools() field can build typed sub-resources on access
// without a second API round-trip.
type mqlStackitSkeClusterInternal struct {
	rawNodepools []ske.Nodepool
}

func buildSkeCluster(runtime *plugin.Runtime, cluster *ske.Cluster) (plugin.Resource, error) {
	var aggregated string
	var creationTime *time.Time
	statusDict := toDict(cluster.GetStatus())
	if status, ok := cluster.GetStatusOk(); ok {
		aggregated = string(status.GetAggregated())
		if ct, ok := status.GetCreationTimeOk(); ok && !ct.IsZero() {
			creationTime = &ct
		}
	}

	var kVersion string
	if k, ok := cluster.GetKubernetesOk(); ok {
		kVersion = k.GetVersion()
	}

	hibernations := []any{}
	if h, ok := cluster.GetHibernationOk(); ok {
		hibernations = anySliceToDict(h.GetSchedules())
	}

	args := map[string]*llx.RawData{
		"name":              llx.StringData(cluster.GetName()),
		"status":            llx.StringData(aggregated),
		"statusDetails":     llx.DictData(statusDict),
		"kubernetesVersion": llx.StringData(kVersion),
		"hibernations":      llx.ArrayData(hibernations, types.Dict),
		"maintenance":       llx.DictData(toDict(cluster.GetMaintenance())),
		"extensions":        llx.DictData(toDict(cluster.GetExtensions())),
		"network":           llx.DictData(toDict(cluster.GetNetwork())),
		"creationTime":      llx.TimeDataPtr(creationTime),
	}
	res, err := CreateResource(runtime, "stackit.ske.cluster", args)
	if err != nil {
		return nil, err
	}
	if mc, ok := res.(*mqlStackitSkeCluster); ok {
		mc.rawNodepools = cluster.GetNodepools()
	}
	return res, nil
}

func (r *mqlStackitSkeCluster) nodePools() ([]any, error) {
	out := make([]any, 0, len(r.rawNodepools))
	for i := range r.rawNodepools {
		np := &r.rawNodepools[i]

		var imageName, imageVersion string
		var machineType string
		if m, ok := np.GetMachineOk(); ok {
			machineType = m.GetType()
			if img, ok := m.GetImageOk(); ok {
				imageName = img.GetName()
				imageVersion = img.GetVersion()
			}
		}

		var volSize int64
		var volType string
		if v, ok := np.GetVolumeOk(); ok {
			volSize = v.GetSize()
			volType = v.GetType()
		}

		var cri string
		if c, ok := np.GetCriOk(); ok {
			cri = string(c.GetName())
		}

		taints := []any{}
		if t, ok := np.GetTaintsOk(); ok {
			taints = anySliceToDict(t)
		}

		args := map[string]*llx.RawData{
			"name":                  llx.StringData(np.GetName()),
			"clusterName":           llx.StringData(r.Name.Data),
			"machineType":           llx.StringData(machineType),
			"machineImage":          llx.StringData(imageName),
			"machineImageVersion":   llx.StringData(imageVersion),
			"volumeSize":            llx.IntData(volSize),
			"volumeType":            llx.StringData(volType),
			"minimum":               llx.IntData(np.GetMinimum()),
			"maximum":               llx.IntData(np.GetMaximum()),
			"maxSurge":              llx.IntData(np.GetMaxSurge()),
			"maxUnavailable":        llx.IntData(np.GetMaxUnavailable()),
			"availabilityZones":     strSliceData(np.GetAvailabilityZones()),
			"cri":                   llx.StringData(cri),
			"taints":                llx.ArrayData(taints, types.Dict),
			"labels":                stringMapData(np.GetLabels()),
			"allowSystemComponents": llx.BoolData(np.GetAllowSystemComponents()),
		}
		res, err := CreateResource(r.MqlRuntime, "stackit.ske.cluster.nodePool", args)
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func (r *mqlStackitSkeClusterNodePool) id() (string, error) {
	return "stackit.ske.cluster.nodePool/" + r.ClusterName.Data + "/" + r.Name.Data, nil
}

// anySliceToDict marshals a slice into []any of dict form.
func anySliceToDict[T any](in []T) []any {
	out := make([]any, len(in))
	for i := range in {
		out[i] = toDict(in[i])
	}
	return out
}

func (r *mqlStackitSkeCluster) id() (string, error) {
	return "stackit.ske.cluster/" + conn(r.MqlRuntime).ProjectID() + "/" + r.Name.Data, nil
}

func initStackitSkeCluster(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	v, ok := args["name"]
	if !ok || v == nil {
		return args, nil, nil
	}
	name, ok := v.Value.(string)
	if !ok || name == "" {
		return args, nil, nil
	}
	c := conn(runtime)
	client, err := c.SKE()
	if err != nil {
		return nil, nil, err
	}
	cluster, err := client.GetClusterExecute(bgctx(), c.ProjectID(), c.Region(), name)
	if err != nil {
		return nil, nil, err
	}
	res, err := buildSkeCluster(runtime, cluster)
	if err != nil {
		return nil, nil, err
	}
	return nil, res, nil
}
