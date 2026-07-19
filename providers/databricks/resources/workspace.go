// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"strconv"

	databricks "github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/settings"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers/databricks/connection"
	"go.mondoo.com/mql/v13/types"
)

func (r *mqlDatabricks) ipAccessLists() ([]any, error) {
	ws, err := workspaceClient(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	lists, err := ws.IpAccessLists.ListAll(context.Background())
	if err != nil {
		return nil, err
	}

	out := []any{}
	for i := range lists {
		l := lists[i]
		res, err := CreateResource(r.MqlRuntime, "databricks.ipAccessList", map[string]*llx.RawData{
			"__id":         llx.StringData("databricks.ipAccessList/" + l.ListId),
			"id":           llx.StringData(l.ListId),
			"label":        llx.StringData(l.Label),
			"listType":     llx.StringData(string(l.ListType)),
			"ipAddresses":  llx.ArrayData(strSlice(l.IpAddresses), types.String),
			"addressCount": llx.IntData(l.AddressCount),
			"enabled":      llx.BoolData(l.Enabled),
		})
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func (r *mqlDatabricks) tokens() ([]any, error) {
	ws, err := workspaceClient(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	tokens, err := ws.TokenManagement.ListAll(context.Background(), settings.ListTokenManagementRequest{})
	if err != nil {
		return nil, err
	}

	out := []any{}
	for i := range tokens {
		t := tokens[i]
		res, err := CreateResource(r.MqlRuntime, "databricks.token", map[string]*llx.RawData{
			"__id":              llx.StringData("databricks.token/" + t.TokenId),
			"id":                llx.StringData(t.TokenId),
			"comment":           llx.StringData(t.Comment),
			"ownerId":           llx.IntData(t.OwnerId),
			"createdByUsername": llx.StringData(t.CreatedByUsername),
			"creationTime":      llx.TimeDataPtr(epochMsTime(t.CreationTime)),
			"expiryTime":        llx.TimeDataPtr(epochMsTime(t.ExpiryTime)),
		})
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func (r *mqlDatabricks) secretScopes() ([]any, error) {
	ws, err := workspaceClient(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	scopes, err := ws.Secrets.ListScopesAll(context.Background())
	if err != nil {
		return nil, err
	}

	out := []any{}
	for i := range scopes {
		s := scopes[i]
		res, err := CreateResource(r.MqlRuntime, "databricks.secretScope", map[string]*llx.RawData{
			"__id":        llx.StringData("databricks.secretScope/" + s.Name),
			"name":        llx.StringData(s.Name),
			"backendType": llx.StringData(string(s.BackendType)),
		})
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func (r *mqlDatabricksSecretScope) acls() (map[string]any, error) {
	ws, err := workspaceClient(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	resp, err := ws.Secrets.ListAclsByScope(context.Background(), r.Name.Data)
	if err != nil {
		return nil, err
	}

	acls := map[string]any{}
	for i := range resp.Items {
		acls[resp.Items[i].Principal] = string(resp.Items[i].Permission)
	}
	return acls, nil
}

func (r *mqlDatabricks) workspaceSettings() (*mqlDatabricksWorkspaceConf, error) {
	ws, err := workspaceClient(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	conn := r.MqlRuntime.Connection.(*connection.DatabricksConnection)
	ctx := context.Background()

	id := conn.WorkspaceID()
	if id == "" {
		id = conn.Host()
	}

	res, err := CreateResource(r.MqlRuntime, "databricks.workspaceConf", map[string]*llx.RawData{
		"__id":                                             llx.StringData("databricks.workspaceConf/" + id),
		"tokensEnabled":                                    llx.BoolDataPtr(confBool(ctx, ws, "enableTokens")),
		"maxTokenLifetimeDays":                             llx.IntDataPtr(confInt(ctx, ws, "maxTokenLifetimeDays")),
		"ipAccessListsEnabled":                             llx.BoolDataPtr(confBool(ctx, ws, "enableIpAccessLists")),
		"deprecatedGlobalInitScriptsEnabled":               llx.BoolDataPtr(confBool(ctx, ws, "enableDeprecatedGlobalInitScripts")),
		"deprecatedClusterNamedInitScriptsEnabled":         llx.BoolDataPtr(confBool(ctx, ws, "enableDeprecatedClusterNamedInitScripts")),
		"storeInteractiveNotebookResultsInCustomerAccount": llx.BoolDataPtr(confBool(ctx, ws, "storeInteractiveNotebookResultsInCustomerAccount")),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlDatabricksWorkspaceConf), nil
}

// confBool reads a single workspace conf key and interprets it as a boolean,
// returning nil when the key is unset or cannot be read. Each key is fetched
// individually because WorkspaceConf.GetStatus does not accept a
// comma-separated key list (a joined string is treated as one unknown key and
// returns nothing).
func confBool(ctx context.Context, ws *databricks.WorkspaceClient, key string) *bool {
	v, ok := confValue(ctx, ws, key)
	if !ok {
		return nil
	}
	b := v == "true" || v == "1"
	return &b
}

// confInt reads a single workspace conf key and interprets it as an integer,
// returning nil when the key is unset or cannot be parsed.
func confInt(ctx context.Context, ws *databricks.WorkspaceClient, key string) *int64 {
	v, ok := confValue(ctx, ws, key)
	if !ok {
		return nil
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return nil
	}
	return &n
}

func confValue(ctx context.Context, ws *databricks.WorkspaceClient, key string) (string, bool) {
	resp, err := ws.WorkspaceConf.GetStatus(ctx, settings.GetStatusRequest{Keys: key})
	if err != nil || resp == nil {
		return "", false
	}
	v, ok := (*resp)[key]
	return v, ok
}
