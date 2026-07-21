// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"strconv"
	"strings"

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

	// The workspace-conf GET endpoint accepts a comma-separated key list and
	// returns every requested key in one response, so all of these settings are
	// read with a single call rather than one request per key.
	conf := confStatus(ctx, ws,
		"enableTokens",
		"maxTokenLifetimeDays",
		"enableIpAccessLists",
		"enableDeprecatedGlobalInitScripts",
		"enableDeprecatedClusterNamedInitScripts",
		"storeInteractiveNotebookResultsInCustomerAccount",
	)

	res, err := CreateResource(r.MqlRuntime, "databricks.workspaceConf", map[string]*llx.RawData{
		"__id":                                             llx.StringData("databricks.workspaceConf/" + id),
		"tokensEnabled":                                    llx.BoolDataPtr(confBoolFrom(conf, "enableTokens")),
		"maxTokenLifetimeDays":                             llx.IntDataPtr(confIntFrom(conf, "maxTokenLifetimeDays")),
		"ipAccessListsEnabled":                             llx.BoolDataPtr(confBoolFrom(conf, "enableIpAccessLists")),
		"deprecatedGlobalInitScriptsEnabled":               llx.BoolDataPtr(confBoolFrom(conf, "enableDeprecatedGlobalInitScripts")),
		"deprecatedClusterNamedInitScriptsEnabled":         llx.BoolDataPtr(confBoolFrom(conf, "enableDeprecatedClusterNamedInitScripts")),
		"storeInteractiveNotebookResultsInCustomerAccount": llx.BoolDataPtr(confBoolFrom(conf, "storeInteractiveNotebookResultsInCustomerAccount")),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlDatabricksWorkspaceConf), nil
}

// confStatus reads the given workspace conf keys in a single request. The
// workspace-conf GET endpoint accepts a comma-separated key list and returns
// each requested key that is set, so one call covers every key. An empty map is
// returned when the settings cannot be read (for example without workspace admin
// rights), which leaves each derived field null.
func confStatus(ctx context.Context, ws *databricks.WorkspaceClient, keys ...string) map[string]string {
	resp, err := ws.WorkspaceConf.GetStatus(ctx, settings.GetStatusRequest{Keys: strings.Join(keys, ",")})
	if err != nil || resp == nil {
		return map[string]string{}
	}
	return *resp
}

// confBoolFrom interprets a workspace conf value as a boolean, returning nil
// when the key is absent from the fetched status.
func confBoolFrom(conf map[string]string, key string) *bool {
	v, ok := conf[key]
	if !ok {
		return nil
	}
	b := v == "true" || v == "1"
	return &b
}

// confIntFrom interprets a workspace conf value as an integer, returning nil
// when the key is absent or cannot be parsed.
func confIntFrom(conf map[string]string, key string) *int64 {
	v, ok := conf[key]
	if !ok {
		return nil
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return nil
	}
	return &n
}
