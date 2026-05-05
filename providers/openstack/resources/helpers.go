// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/openstack/connection"
	"go.mondoo.com/mql/v13/types"
)

func conn(runtime *plugin.Runtime) *connection.OpenstackConnection {
	return runtime.Connection.(*connection.OpenstackConnection)
}

// userOptionBool reads a Keystone user option key as a bool. Absent or
// non-bool values resolve to false, matching Keystone's default behaviour
// when an option is unset.
func userOptionBool(options map[string]any, key string) bool {
	if options == nil {
		return false
	}
	v, ok := options[key].(bool)
	return ok && v
}

func ctx() context.Context {
	return context.Background()
}

// translateOpenstackError maps 401/403/404 to "no data" so a missing capability
// (e.g. non-admin token calling Keystone list APIs) doesn't fail the whole
// query. Other errors propagate. Mirrors the AWS Is400AccessDenied pattern.
func translateOpenstackError(err error) error {
	if err == nil {
		return nil
	}
	var resp gophercloud.ErrUnexpectedResponseCode
	if errors.As(err, &resp) {
		switch resp.Actual {
		case 401, 403, 404:
			log.Warn().Err(err).Int("status", resp.Actual).Msg("openstack> permission denied or not found; returning empty result")
			return nil
		}
	}
	return err
}

func stringSlice(in []string) []any {
	out := make([]any, len(in))
	for i, v := range in {
		out[i] = v
	}
	return out
}

func stringSliceData(in []string) *llx.RawData {
	return llx.ArrayData(stringSlice(in), types.String)
}

func stringMap(in map[string]string) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func stringMapData(in map[string]string) *llx.RawData {
	return llx.MapData(stringMap(in), types.String)
}

// dictSliceData wraps a slice of map[string]any (or nil) as a []dict field.
func dictSliceData(in []any) *llx.RawData {
	if in == nil {
		in = []any{}
	}
	return llx.ArrayData(in, types.Dict)
}

// timePtr returns a pointer to t, or nil if t is the zero value. OpenStack
// returns the zero time for unset timestamps (e.g. terminated_at while running).
func timePtr(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

// stringArg returns a string argument from an init args map. Returns
// (zero-value, false) when the arg is missing or the wrong type.
func stringArg(args map[string]*llx.RawData, key string) (string, bool) {
	v, ok := args[key]
	if !ok || v == nil {
		return "", false
	}
	s, ok := v.Value.(string)
	if !ok {
		return "", false
	}
	return s, true
}
