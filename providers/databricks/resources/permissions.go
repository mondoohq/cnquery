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
	permissionObjectRepo            = "repos"
	permissionObjectInstancePool    = "instance-pools"
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

// permissionRecord is one principal's hold on one permission level, flattened
// out of the grouped shape the API returns.
type permissionRecord struct {
	id                  string
	principal           string
	principalType       string
	displayName         string
	permissionLevel     string
	inherited           bool
	inheritedFromObject []string
	objectType          string
	objectId            string
}

// flattenObjectPermissions maps an access control list to one record per
// principal and permission level. The API groups a principal's levels under that
// principal, but each level carries its own inheritance, so keeping them grouped
// would lose which level is direct and which comes from a parent. An entry that
// names no principal is dropped rather than reported as held by "".
func flattenObjectPermissions(objectType string, objectId string, perms *iam.ObjectPermissions) []permissionRecord {
	out := []permissionRecord{}
	if perms == nil {
		return out
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
			// An entry with no level grants nothing, and since the level is what
			// makes a record's id unique, two such entries for one principal
			// would collide in the cache and the second would be dropped without
			// a trace. Dropping them here is deliberate and total.
			if level == "" {
				continue
			}
			out = append(out, permissionRecord{
				id:                  "databricks.permission/" + objectType + "/" + objectId + "/" + principal + "/" + level,
				principal:           principal,
				principalType:       kind,
				displayName:         ac.DisplayName,
				permissionLevel:     level,
				inherited:           p.Inherited,
				inheritedFromObject: p.InheritedFromObject,
				objectType:          objectType,
				objectId:            objectId,
			})
		}
	}
	return out
}

// mqlDatabricksPermissions fetches the workspace access control list of a single
// object and maps it to one databricks.permission per principal and permission
// level.
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
	for _, rec := range flattenObjectPermissions(objectType, objectId, perms) {
		res, err := CreateResource(runtime, "databricks.permission", map[string]*llx.RawData{
			"__id":                llx.StringData(rec.id),
			"principal":           llx.StringData(rec.principal),
			"principalType":       llx.StringData(rec.principalType),
			"displayName":         llx.StringData(rec.displayName),
			"permissionLevel":     llx.StringData(rec.permissionLevel),
			"inherited":           llx.BoolData(rec.inherited),
			"inheritedFromObject": llx.ArrayData(strSlice(rec.inheritedFromObject), types.String),
			"objectType":          llx.StringData(rec.objectType),
			"objectId":            llx.StringData(rec.objectId),
		})
		if err != nil {
			return nil, err
		}
		out = append(out, res)
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
