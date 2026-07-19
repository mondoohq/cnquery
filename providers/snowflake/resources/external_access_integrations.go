// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/snowflake/connection"
	"go.mondoo.com/mql/v13/types"
)

// External access integrations have no typed SDK object, so they are read with
// raw SHOW / DESC statements through the driver.
func (r *mqlSnowflakeAccount) externalAccessIntegrations() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.SnowflakeConnection)
	client := conn.Client()
	ctx := context.Background()

	rows, err := client.QueryUnsafe(ctx, "SHOW EXTERNAL ACCESS INTEGRATIONS")
	if err != nil {
		return nil, err
	}

	list := []any{}
	for _, row := range rows {
		name := unsafeString(row["name"])
		if name == "" {
			continue
		}
		res, err := newMqlSnowflakeExternalAccessIntegration(r.MqlRuntime, name, unsafeBool(row["enabled"]), unsafeString(row["comment"]))
		if err != nil {
			return nil, err
		}
		list = append(list, res)
	}
	return list, nil
}

// initSnowflakeExternalAccessIntegration resolves a single external access
// integration by name so typed references (such as
// snowflake.function.externalAccessIntegrations) can hydrate a full integration
// from just its name.
func initSnowflakeExternalAccessIntegration(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 {
		return args, nil, nil
	}
	nameRaw, ok := args["name"]
	if !ok {
		return args, nil, nil
	}
	name, _ := nameRaw.Value.(string)
	if name == "" {
		return args, nil, nil
	}

	conn := runtime.Connection.(*connection.SnowflakeConnection)
	client := conn.Client()
	ctx := context.Background()

	rows, err := client.QueryUnsafe(ctx, fmt.Sprintf("SHOW EXTERNAL ACCESS INTEGRATIONS LIKE '%s'", strings.ReplaceAll(name, "'", "''")))
	if err != nil {
		return nil, nil, err
	}
	for _, row := range rows {
		if unsafeString(row["name"]) != name {
			continue
		}
		res, err := newMqlSnowflakeExternalAccessIntegration(runtime, name, unsafeBool(row["enabled"]), unsafeString(row["comment"]))
		if err != nil {
			return nil, nil, err
		}
		return nil, res, nil
	}
	return nil, nil, fmt.Errorf("snowflake.externalAccessIntegration %q not found", name)
}

// newMqlSnowflakeExternalAccessIntegration builds the resource, running DESC to
// populate the egress allowlists that SHOW omits.
func newMqlSnowflakeExternalAccessIntegration(runtime *plugin.Runtime, name string, enabled bool, comment string) (*mqlSnowflakeExternalAccessIntegration, error) {
	conn := runtime.Connection.(*connection.SnowflakeConnection)
	client := conn.Client()
	ctx := context.Background()

	networkRules := []any{}
	secrets := []any{}
	apiAuth := []any{}
	descRows, derr := client.QueryUnsafe(ctx, fmt.Sprintf(`DESC EXTERNAL ACCESS INTEGRATION "%s"`, name))
	if derr != nil {
		// Distinguish a DESC failure (permissions or transient) from a
		// genuinely empty integration: the allowlists stay empty either way, so
		// surface the error rather than swallowing it silently.
		log.Warn().Err(derr).Str("integration", name).Msg("snowflake: DESC EXTERNAL ACCESS INTEGRATION failed, allowlists will be empty")
	} else {
		for _, drow := range descRows {
			value := unsafeString(drow["property_value"])
			switch strings.ToUpper(unsafeString(drow["property"])) {
			case "ALLOWED_NETWORK_RULES":
				networkRules = parseExternalAccessList(value)
			case "ALLOWED_AUTHENTICATION_SECRETS":
				secrets = parseExternalAccessList(value)
			case "ALLOWED_API_AUTHENTICATION_INTEGRATIONS":
				apiAuth = parseExternalAccessList(value)
			}
		}
	}

	res, err := CreateResource(runtime, "snowflake.externalAccessIntegration", map[string]*llx.RawData{
		"__id":                                 llx.StringData("snowflake.externalAccessIntegration/" + name),
		"name":                                 llx.StringData(name),
		"enabled":                              llx.BoolData(enabled),
		"comment":                              llx.StringData(comment),
		"allowedNetworkRules":                  llx.ArrayData(networkRules, types.String),
		"allowedAuthenticationSecrets":         llx.ArrayData(secrets, types.String),
		"allowedApiAuthenticationIntegrations": llx.ArrayData(apiAuth, types.String),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlSnowflakeExternalAccessIntegration), nil
}

// unsafeString coerces a QueryUnsafe cell (which the driver returns as *any) to
// a string.
func unsafeString(v *any) string {
	if v == nil || *v == nil {
		return ""
	}
	switch x := (*v).(type) {
	case string:
		return x
	case bool:
		if x {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprintf("%v", x)
	}
}

// unsafeBool coerces a QueryUnsafe cell to a bool, accepting the driver's bool
// or the string form SHOW returns.
func unsafeBool(v *any) bool {
	if v == nil || *v == nil {
		return false
	}
	switch x := (*v).(type) {
	case bool:
		return x
	case string:
		return strings.EqualFold(strings.TrimSpace(x), "true")
	default:
		return false
	}
}

// parseExternalAccessList splits a DESC property value like "[DB.SCH.RULE1,
// DB.SCH.RULE2]" into the individual fully qualified names, dropping brackets
// and blank entries.
func parseExternalAccessList(raw string) []any {
	out := []any{}
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "[")
	raw = strings.TrimSuffix(raw, "]")
	for _, part := range strings.Split(raw, ",") {
		p := strings.TrimSpace(part)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
