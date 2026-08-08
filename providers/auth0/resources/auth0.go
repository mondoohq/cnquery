// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/auth0/connection"
	"go.mondoo.com/mql/v13/types"
)

func (r *mqlAuth0) id() (string, error) {
	return "auth0", nil
}

// tenant reads the tenant-wide settings singleton. Its cache key is the tenant
// domain, since there is exactly one tenant per connection.
func (r *mqlAuth0) tenant() (*mqlAuth0Tenant, error) {
	conn := r.conn()
	t, err := conn.Client().Tenant.Read(context.Background())
	if err != nil {
		return nil, err
	}

	flags, err := convert.JsonToDict(t.Flags)
	if err != nil {
		return nil, err
	}

	res, err := CreateResource(r.MqlRuntime, "auth0.tenant", map[string]*llx.RawData{
		"__id":                llx.StringData(conn.Domain()),
		"friendlyName":        llx.StringDataPtr(t.FriendlyName),
		"supportEmail":        llx.StringDataPtr(t.SupportEmail),
		"supportUrl":          llx.StringDataPtr(t.SupportURL),
		"pictureUrl":          llx.StringDataPtr(t.PictureURL),
		"allowedLogoutUrls":   llx.ArrayData(strList(t.AllowedLogoutURLs), types.String),
		"sessionLifetime":     floatPtr(t.SessionLifetime),
		"idleSessionLifetime": floatPtr(t.IdleSessionLifetime),
		"defaultAudience":     llx.StringDataPtr(t.DefaultAudience),
		"defaultDirectory":    llx.StringDataPtr(t.DefaultDirectory),
		"enabledLocales":      llx.ArrayData(strList(t.EnabledLocales), types.String),
		"flags":               llx.DictData(flags),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAuth0Tenant), nil
}

// auth0conn returns the Auth0 connection backing this runtime.
func (r *mqlAuth0) conn() *connection.Auth0Connection {
	return r.MqlRuntime.Connection.(*connection.Auth0Connection)
}

// strList converts a *[]string (as returned by the SDK) into an MQL string
// array value, treating a nil pointer as an empty list.
func strList(s *[]string) []any {
	if s == nil {
		return []any{}
	}
	res := make([]any, 0, len(*s))
	for _, v := range *s {
		res = append(res, v)
	}
	return res
}

// floatPtr converts a *float64 into an MQL float value, preserving nil as null.
func floatPtr(v *float64) *llx.RawData {
	if v == nil {
		return llx.NilData
	}
	return llx.FloatData(*v)
}

// mapBool safely extracts a boolean value from a loosely typed JSON map
// (map[string]interface{}), returning nil when the key is absent or not a bool.
func mapBool(m map[string]interface{}, key string) *bool {
	if m == nil {
		return nil
	}
	if v, ok := m[key].(bool); ok {
		return &v
	}
	return nil
}

// mapInt safely extracts an integer value from a loosely typed JSON map,
// tolerating the float64 that encoding/json produces for numbers.
func mapInt(m map[string]interface{}, key string) *int64 {
	if m == nil {
		return nil
	}
	switch v := m[key].(type) {
	case float64:
		i := int64(v)
		return &i
	case int:
		i := int64(v)
		return &i
	case int64:
		return &v
	}
	return nil
}
