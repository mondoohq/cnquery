// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"time"

	databricks "github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/databricks/connection"
)

func (r *mqlDatabricks) id() (string, error) {
	conn := r.MqlRuntime.Connection.(*connection.DatabricksConnection)
	if conn.Plane() == connection.PlaneWorkspace {
		id := conn.WorkspaceID()
		if id == "" {
			id = conn.Host()
		}
		return "databricks/workspace/" + id, nil
	}
	return "databricks/account/" + conn.AccountID(), nil
}

// accountClient returns the account console client, or an error when the asset
// is connected to a single workspace rather than the account.
func accountClient(runtime *plugin.Runtime) (*databricks.AccountClient, error) {
	conn := runtime.Connection.(*connection.DatabricksConnection)
	acc := conn.Account()
	if acc == nil {
		return nil, errors.New("this resource requires connecting to the Databricks account console (use --account-id)")
	}
	return acc, nil
}

// workspaceClient returns the workspace client, or an error when the asset is
// connected to the account console rather than a workspace.
func workspaceClient(runtime *plugin.Runtime) (*databricks.WorkspaceClient, error) {
	conn := runtime.Connection.(*connection.DatabricksConnection)
	ws := conn.Workspace()
	if ws == nil {
		return nil, errors.New("this resource requires connecting to a Databricks workspace")
	}
	return ws, nil
}

// complexValues extracts the meaningful Value of each SCIM ComplexValue,
// dropping empties. Used for entitlements, roles, and group membership.
func complexValues(vals []iam.ComplexValue) []any {
	out := []any{}
	for i := range vals {
		if vals[i].Value != "" {
			out = append(out, vals[i].Value)
		}
	}
	return out
}

// strSlice converts a string slice to the []any form llx.ArrayData expects.
func strSlice(vals []string) []any {
	out := make([]any, 0, len(vals))
	for _, v := range vals {
		out = append(out, v)
	}
	return out
}

// strMap converts a string map to the map[string]any form llx.MapData expects.
func strMap(m map[string]string) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// epochMsTime converts a Databricks epoch-millisecond timestamp to a time,
// returning nil for the zero/negative sentinels the API uses for "unset".
func epochMsTime(ms int64) *time.Time {
	if ms <= 0 {
		return nil
	}
	t := time.UnixMilli(ms)
	return &t
}
