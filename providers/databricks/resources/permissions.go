// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"strconv"

	"github.com/databricks/databricks-sdk-go/service/iam"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/types"
)

// Access control object types accepted by the workspace permissions API. Each
// value is the path segment the API expects, so "sql/warehouses" carries its
// slash into the request path by design.
const (
	permissionObjectCluster         = "clusters"
	permissionObjectClusterPolicy   = "cluster-policies"
	permissionObjectWarehouse       = "sql/warehouses"
	permissionObjectJob             = "jobs"
	permissionObjectPipeline        = "pipelines"
	permissionObjectServingEndpoint = "serving-endpoints"
)

// principalKinds classifies an access control entry. Exactly one of the three
// name fields is set per entry, which is what distinguishes a user from a group
// from a service principal.
const (
	principalKindUser             = "USER"
	principalKindGroup            = "GROUP"
	principalKindServicePrincipal = "SERVICE_PRINCIPAL"
)

// principalOf resolves the name and kind of the principal an access control
// entry belongs to. An entry that names no principal is reported as empty so
// the caller can drop it rather than emit an unattributable permission.
func principalOf(ac iam.AccessControlResponse) (name string, kind string) {
	switch {
	case ac.UserName != "":
		return ac.UserName, principalKindUser
	case ac.GroupName != "":
		return ac.GroupName, principalKindGroup
	case ac.ServicePrincipalName != "":
		return ac.ServicePrincipalName, principalKindServicePrincipal
	}
	return "", ""
}

// mqlDatabricksPermissions fetches the workspace access control list of a single
// object and maps it to one databricks.permission per principal and permission
// level. The API returns the levels of a principal grouped under that principal,
// but each level carries its own inheritance, so keeping them grouped would lose
// which level is direct and which comes from a parent.
func mqlDatabricksPermissions(runtime *plugin.Runtime, objectType string, objectId string) ([]any, error) {
	ws, err := workspaceClient(runtime)
	if err != nil {
		return nil, err
	}

	perms, err := ws.Permissions.GetByRequestObjectTypeAndRequestObjectId(context.Background(), objectType, objectId)
	if err != nil {
		return nil, err
	}

	out := []any{}
	if perms == nil {
		return out, nil
	}

	for i := range perms.AccessControlList {
		ac := perms.AccessControlList[i]
		principal, kind := principalOf(ac)
		if principal == "" {
			continue
		}

		for j := range ac.AllPermissions {
			p := ac.AllPermissions[j]
			level := string(p.PermissionLevel)
			res, err := CreateResource(runtime, "databricks.permission", map[string]*llx.RawData{
				"__id":                llx.StringData("databricks.permission/" + objectType + "/" + objectId + "/" + principal + "/" + level),
				"principal":           llx.StringData(principal),
				"principalType":       llx.StringData(kind),
				"displayName":         llx.StringData(ac.DisplayName),
				"permissionLevel":     llx.StringData(level),
				"inherited":           llx.BoolData(p.Inherited),
				"inheritedFromObject": llx.ArrayData(strSlice(p.InheritedFromObject), types.String),
				"objectType":          llx.StringData(objectType),
				"objectId":            llx.StringData(objectId),
			})
			if err != nil {
				return nil, err
			}
			out = append(out, res)
		}
	}
	return out, nil
}

func (r *mqlDatabricksCluster) permissions() ([]any, error) {
	return mqlDatabricksPermissions(r.MqlRuntime, permissionObjectCluster, r.Id.Data)
}

func (r *mqlDatabricksClusterPolicy) permissions() ([]any, error) {
	return mqlDatabricksPermissions(r.MqlRuntime, permissionObjectClusterPolicy, r.Id.Data)
}

func (r *mqlDatabricksWarehouse) permissions() ([]any, error) {
	return mqlDatabricksPermissions(r.MqlRuntime, permissionObjectWarehouse, r.Id.Data)
}

func (r *mqlDatabricksJob) permissions() ([]any, error) {
	return mqlDatabricksPermissions(r.MqlRuntime, permissionObjectJob, strconv.FormatInt(r.Id.Data, 10))
}

func (r *mqlDatabricksPipeline) permissions() ([]any, error) {
	return mqlDatabricksPermissions(r.MqlRuntime, permissionObjectPipeline, r.Id.Data)
}

// permissions reads the endpoint access control list. The permissions API keys
// serving endpoints on the endpoint id rather than its name, unlike the serving
// endpoints API itself.
func (r *mqlDatabricksServingEndpoint) permissions() ([]any, error) {
	return mqlDatabricksPermissions(r.MqlRuntime, permissionObjectServingEndpoint, r.Id.Data)
}
