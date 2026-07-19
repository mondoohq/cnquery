// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	databricks "github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/types"
)

func (r *mqlDatabricks) catalogs() ([]any, error) {
	ws, err := workspaceClient(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	catalogs, err := ws.Catalogs.ListAll(context.Background(), catalog.ListCatalogsRequest{})
	if err != nil {
		return nil, err
	}

	out := []any{}
	for i := range catalogs {
		c := catalogs[i]
		res, err := CreateResource(r.MqlRuntime, "databricks.catalog", map[string]*llx.RawData{
			"__id":          llx.StringData("databricks.catalog/" + c.FullName),
			"name":          llx.StringData(c.Name),
			"fullName":      llx.StringData(c.FullName),
			"owner":         llx.StringData(c.Owner),
			"metastoreId":   llx.StringData(c.MetastoreId),
			"isolationMode": llx.StringData(string(c.IsolationMode)),
			"comment":       llx.StringData(c.Comment),
			"catalogType":   llx.StringData(string(c.CatalogType)),
		})
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func (r *mqlDatabricksCatalog) schemas() ([]any, error) {
	ws, err := workspaceClient(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	schemas, err := ws.Schemas.ListAll(context.Background(), catalog.ListSchemasRequest{CatalogName: r.Name.Data})
	if err != nil {
		return nil, err
	}

	out := []any{}
	for i := range schemas {
		s := schemas[i]
		res, err := CreateResource(r.MqlRuntime, "databricks.schema", map[string]*llx.RawData{
			"__id":        llx.StringData("databricks.schema/" + s.FullName),
			"id":          llx.StringData(s.SchemaId),
			"name":        llx.StringData(s.Name),
			"fullName":    llx.StringData(s.FullName),
			"catalogName": llx.StringData(s.CatalogName),
			"owner":       llx.StringData(s.Owner),
			"comment":     llx.StringData(s.Comment),
		})
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func (r *mqlDatabricksCatalog) grants() ([]any, error) {
	ws, err := workspaceClient(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	return mqlDatabricksGrants(r.MqlRuntime, ws, catalog.SecurableTypeCatalog, r.FullName.Data)
}

func (r *mqlDatabricksSchema) grants() ([]any, error) {
	ws, err := workspaceClient(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	return mqlDatabricksGrants(r.MqlRuntime, ws, catalog.SecurableTypeSchema, r.FullName.Data)
}

// mqlDatabricksGrants fetches the direct privilege assignments on a Unity
// Catalog securable and maps each principal's grant to a databricks.grant.
func mqlDatabricksGrants(runtime *plugin.Runtime, ws *databricks.WorkspaceClient, securableType catalog.SecurableType, fullName string) ([]any, error) {
	// ListAll follows the response page tokens; GetBySecurableTypeAndFullName
	// would return only the first page.
	assignments, err := ws.Grants.ListAll(context.Background(), catalog.ListPrivilegeAssignmentsRequest{
		SecurableType: string(securableType),
		FullName:      fullName,
	})
	if err != nil {
		return nil, err
	}

	out := []any{}
	for i := range assignments {
		pa := assignments[i]
		privileges := make([]any, 0, len(pa.Privileges))
		for _, p := range pa.Privileges {
			privileges = append(privileges, string(p))
		}
		res, err := CreateResource(runtime, "databricks.grant", map[string]*llx.RawData{
			"__id":          llx.StringData(string(securableType) + "/" + fullName + "/" + pa.Principal),
			"principal":     llx.StringData(pa.Principal),
			"securableType": llx.StringData(string(securableType)),
			"securableName": llx.StringData(fullName),
			"privileges":    llx.ArrayData(privileges, types.String),
		})
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}
