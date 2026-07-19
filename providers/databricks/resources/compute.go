// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/sql"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/types"
)

type mqlDatabricksClusterInternal struct {
	cachePolicyId string
}

func (r *mqlDatabricks) clusterPolicies() ([]any, error) {
	ws, err := workspaceClient(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	policies, err := ws.ClusterPolicies.ListAll(context.Background(), compute.ListClusterPoliciesRequest{})
	if err != nil {
		return nil, err
	}

	out := []any{}
	for i := range policies {
		mqlPolicy, err := newMqlDatabricksClusterPolicy(r.MqlRuntime, policies[i])
		if err != nil {
			return nil, err
		}
		out = append(out, mqlPolicy)
	}
	return out, nil
}

func newMqlDatabricksClusterPolicy(runtime *plugin.Runtime, p compute.Policy) (*mqlDatabricksClusterPolicy, error) {
	res, err := CreateResource(runtime, "databricks.clusterPolicy", map[string]*llx.RawData{
		"__id":               llx.StringData("databricks.clusterPolicy/" + p.PolicyId),
		"id":                 llx.StringData(p.PolicyId),
		"name":               llx.StringData(p.Name),
		"description":        llx.StringData(p.Description),
		"definition":         llx.StringData(p.Definition),
		"maxClustersPerUser": llx.IntData(p.MaxClustersPerUser),
		"isDefault":          llx.BoolData(p.IsDefault),
		"creatorUserName":    llx.StringData(p.CreatorUserName),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlDatabricksClusterPolicy), nil
}

func (r *mqlDatabricks) clusters() ([]any, error) {
	ws, err := workspaceClient(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	clusters, err := ws.Clusters.ListAll(context.Background(), compute.ListClustersRequest{})
	if err != nil {
		return nil, err
	}

	out := []any{}
	for i := range clusters {
		c := clusters[i]
		res, err := CreateResource(r.MqlRuntime, "databricks.cluster", map[string]*llx.RawData{
			"__id":                       llx.StringData("databricks.cluster/" + c.ClusterId),
			"id":                         llx.StringData(c.ClusterId),
			"clusterName":                llx.StringData(c.ClusterName),
			"state":                      llx.StringData(string(c.State)),
			"dataSecurityMode":           llx.StringData(string(c.DataSecurityMode)),
			"singleUserName":             llx.StringData(c.SingleUserName),
			"sparkVersion":               llx.StringData(c.SparkVersion),
			"runtimeEngine":              llx.StringData(string(c.RuntimeEngine)),
			"sparkConf":                  llx.MapData(strMap(c.SparkConf), types.String),
			"sparkEnvVars":               llx.MapData(strMap(c.SparkEnvVars), types.String),
			"localDiskEncryptionEnabled": llx.BoolData(c.EnableLocalDiskEncryption),
			"autoterminationMinutes":     llx.IntData(c.AutoterminationMinutes),
			"creatorUserName":            llx.StringData(c.CreatorUserName),
			"customTags":                 llx.MapData(strMap(c.CustomTags), types.String),
		})
		if err != nil {
			return nil, err
		}
		mqlCluster := res.(*mqlDatabricksCluster)
		mqlCluster.cachePolicyId = c.PolicyId
		out = append(out, mqlCluster)
	}
	return out, nil
}

func (r *mqlDatabricksCluster) policy() (*mqlDatabricksClusterPolicy, error) {
	if r.cachePolicyId == "" {
		r.Policy.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	ws, err := workspaceClient(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	p, err := ws.ClusterPolicies.Get(context.Background(), compute.GetClusterPolicyRequest{PolicyId: r.cachePolicyId})
	if err != nil {
		return nil, err
	}
	if p == nil {
		r.Policy.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return newMqlDatabricksClusterPolicy(r.MqlRuntime, *p)
}

func (r *mqlDatabricks) warehouses() ([]any, error) {
	ws, err := workspaceClient(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	warehouses, err := ws.Warehouses.ListAll(context.Background(), sql.ListWarehousesRequest{})
	if err != nil {
		return nil, err
	}

	out := []any{}
	for i := range warehouses {
		w := warehouses[i]
		channel := ""
		if w.Channel != nil {
			channel = string(w.Channel.Name)
		}
		res, err := CreateResource(r.MqlRuntime, "databricks.warehouse", map[string]*llx.RawData{
			"__id":              llx.StringData("databricks.warehouse/" + w.Id),
			"id":                llx.StringData(w.Id),
			"name":              llx.StringData(w.Name),
			"state":             llx.StringData(string(w.State)),
			"warehouseType":     llx.StringData(string(w.WarehouseType)),
			"photonEnabled":     llx.BoolData(w.EnablePhoton),
			"serverlessEnabled": llx.BoolData(w.EnableServerlessCompute),
			"channel":           llx.StringData(channel),
			"clusterSize":       llx.StringData(w.ClusterSize),
			"autoStopMinutes":   llx.IntData(w.AutoStopMins),
			"creatorName":       llx.StringData(w.CreatorName),
		})
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}
