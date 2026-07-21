// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/types"
)

func bgctx() context.Context { return context.Background() }

// timeOrNil turns a (time, ok) result from the SDK's GetXxxOk methods into the
// *time.Time form that llx.TimeDataPtr wants. Zero time is treated as unset.
func timeOrNil(t time.Time, ok bool) *time.Time {
	if !ok || t.IsZero() {
		return nil
	}
	return &t
}

// parseRFC3339 turns an RFC3339 timestamp string into the *time.Time form
// llx.TimeDataPtr wants, or nil if the string is empty or malformed. Several
// STACKIT services return timestamps as strings rather than time.Time.
func parseRFC3339(s string) *time.Time {
	if s == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil
	}
	return &t
}

// parseKeyBitSize extracts the numeric key length from STACKIT's human-readable
// key-strength string (for example "RSA 2048" -> 2048). It returns nil for
// elliptic-curve keys, whose strength is expressed as a curve name rather than
// a bit count (for example "ECDSA P-256" or "Ed25519"); only a whitespace-
// delimited all-digit token counts, so the "256" in "P-256" is not mistaken
// for a bit size.
func parseKeyBitSize(s string) *int64 {
	for _, tok := range strings.Fields(s) {
		if n, err := strconv.ParseInt(tok, 10, 64); err == nil {
			return &n
		}
	}
	return nil
}

// strSlice converts a []string into the any-typed slice MQL expects.
func strSlice(in []string) []any {
	out := make([]any, len(in))
	for i, s := range in {
		out[i] = s
	}
	return out
}

// stringMap converts a map[string]string into the any-valued form MQL expects
// for `map[string]string` fields.
func stringMap(in map[string]string) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// strSliceData wraps a []string for assignment to a `[]string` field.
func strSliceData(in []string) *llx.RawData {
	return llx.ArrayData(strSlice(in), types.String)
}

// stringMapData wraps a map[string]string for a `map[string]string` field.
func stringMapData(in map[string]string) *llx.RawData {
	return llx.MapData(stringMap(in), types.String)
}

// labelData wraps a STACKIT label map (string→string OR string→interface{},
// depending on which SDK module emitted it) for a `map[string]string` field.
// STACKIT only stores string values in labels even when the type is broader.
func labelData(in any) *llx.RawData {
	out := map[string]any{}
	switch m := in.(type) {
	case map[string]string:
		for k, v := range m {
			out[k] = v
		}
	case map[string]interface{}:
		for k, v := range m {
			if s, ok := v.(string); ok {
				out[k] = s
			}
		}
	}
	return llx.MapData(out, types.String)
}

// metadataData is the same as labelData; kept distinct so callers read clearly.
func metadataData(in any) *llx.RawData { return labelData(in) }

// ptrStr derefs a nullable string returned by the SDK's getter methods.
func ptrStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// toDict marshals any SDK struct (or other value) into the JSON-equivalent
// map/slice form that MQL `dict` fields accept. The cheap way to avoid
// hand-rolling getter conversions for every nested SDK object — at the cost
// of one allocation per call. Returns nil for nil inputs or marshal errors.
func toDict(v any) any {
	if v == nil {
		return nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	var out any
	if err := json.Unmarshal(b, &out); err != nil {
		return nil
	}
	return out
}

// dictAny renders any value into MQL-compatible dict form. Maps/slices are
// recursed; nil → nil; scalars pass through.
func dictAny(v any) any {
	switch t := v.(type) {
	case nil:
		return nil
	case map[string]interface{}:
		out := make(map[string]any, len(t))
		for k, val := range t {
			out[k] = dictAny(val)
		}
		return out
	case []interface{}:
		out := make([]any, len(t))
		for i, val := range t {
			out[i] = dictAny(val)
		}
		return out
	default:
		return v
	}
}

// isAccessDenied returns true for 401/403 — the standard "permission
// missing for this scope" fallback so a single denied call doesn't fail the
// whole query. 404 is intentionally excluded: a "not found" can mean a wrong
// project/instance ID and should surface as an error rather than be silently
// swallowed as an empty result.
func isAccessDenied(err error) bool {
	if err == nil {
		return false
	}
	var oerr *oapierror.GenericOpenAPIError
	if errors.As(err, &oerr) {
		switch oerr.StatusCode {
		case http.StatusUnauthorized, http.StatusForbidden:
			return true
		}
	}
	msg := err.Error()
	return strings.Contains(msg, "status 401") ||
		strings.Contains(msg, "status 403")
}

// isNotFound returns true for HTTP 404. Use this for optional sub-resources
// where "not configured" is a legitimate state distinct from access-denied.
func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	var oerr *oapierror.GenericOpenAPIError
	if errors.As(err, &oerr) {
		return oerr.StatusCode == http.StatusNotFound
	}
	return strings.Contains(err.Error(), "status 404")
}

// markNull marks a typed-resource pointer field as set+null and returns nil.
// Used by typed-ref methods when the source identifier is empty.
func markNull[T any](field *plugin.TValue[*T]) (*T, error) {
	field.State = plugin.StateIsSet | plugin.StateIsNull
	return nil, nil
}

// idArg pulls a single string arg out of an init args map.
func idArg(args map[string]*llx.RawData, key string) (string, bool) {
	v, ok := args[key]
	if !ok || v == nil {
		return "", false
	}
	s, ok := v.Value.(string)
	return s, ok && s != ""
}

// makeNamespace creates the bare namespace resource (no fields, just child
// collection getters).
func makeNamespace(runtime *plugin.Runtime, name string) (plugin.Resource, error) {
	return CreateResource(runtime, name, map[string]*llx.RawData{})
}

// serverRef resolves a stackit.server by its UUID, marking the given field
// null when the ID is empty. Shared by the server-scoped sub-resources
// (backups, schedules, updates) that carry a back-reference to their server.
func serverRef(runtime *plugin.Runtime, id string, field *plugin.TValue[*mqlStackitServer]) (*mqlStackitServer, error) {
	if id == "" {
		return markNull[mqlStackitServer](field)
	}
	res, err := NewResource(runtime, "stackit.server", map[string]*llx.RawData{
		"id": llx.StringData(id),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlStackitServer), nil
}

// volumeRef resolves a single stackit.volume by its UUID, marking the given
// field null when the ID is empty.
func volumeRef(runtime *plugin.Runtime, id string, field *plugin.TValue[*mqlStackitVolume]) (*mqlStackitVolume, error) {
	if id == "" {
		return markNull[mqlStackitVolume](field)
	}
	res, err := NewResource(runtime, "stackit.volume", map[string]*llx.RawData{
		"id": llx.StringData(id),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlStackitVolume), nil
}

// volumeRefs resolves a list of stackit.volume resources from their UUIDs,
// skipping empty IDs.
func volumeRefs(runtime *plugin.Runtime, ids []string) ([]any, error) {
	out := make([]any, 0, len(ids))
	for _, id := range ids {
		if id == "" {
			continue
		}
		v, err := NewResource(runtime, "stackit.volume", map[string]*llx.RawData{
			"id": llx.StringData(id),
		})
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}
