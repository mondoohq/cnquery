// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	tsclient "github.com/tailscale/tailscale-client-go/v2"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/tailscale/connection"
	"go.mondoo.com/mql/v13/types"
)

// mqlTailscaleAclPolicyInternal caches the raw HuJSON body across `raw()`
// reads. The Tailscale API exposes the structured ACL (`PolicyFile().Get()`)
// and the raw HuJSON (`PolicyFile().Raw()`) as separate endpoints — the
// structured response does not include the source HuJSON — so a second call
// is required the first time `raw()` is read.
type mqlTailscaleAclPolicyInternal struct {
	rawLock    sync.Mutex
	rawFetched bool
	rawValue   string
}

func (a *mqlTailscaleAclPolicy) id() (string, error) {
	return "tailscale/tailnet/" + a.Tailnet.Data + "/aclPolicy", nil
}

func createTailscaleAclPolicyResource(runtime *plugin.Runtime, tailnet string, acl *tsclient.ACL) (plugin.Resource, error) {
	autoApproverExitNodes := []any{}
	autoApproverRoutes := map[string]any{}
	if acl.AutoApprovers != nil {
		for _, v := range acl.AutoApprovers.ExitNode {
			autoApproverExitNodes = append(autoApproverExitNodes, v)
		}
		for k, owners := range acl.AutoApprovers.Routes {
			arr := make([]any, 0, len(owners))
			for _, owner := range owners {
				arr = append(arr, owner)
			}
			autoApproverRoutes[k] = arr
		}
	}

	acls, err := structSliceToDictSlice(acl.ACLs)
	if err != nil {
		return nil, err
	}
	ssh, err := structSliceToDictSlice(acl.SSH)
	if err != nil {
		return nil, err
	}
	tests, err := structSliceToDictSlice(acl.Tests)
	if err != nil {
		return nil, err
	}
	nodeAttrs, err := structSliceToDictSlice(acl.NodeAttrs)
	if err != nil {
		return nil, err
	}

	return CreateResource(runtime, "tailscale.aclPolicy", map[string]*llx.RawData{
		"tailnet":               llx.StringData(tailnet),
		"acls":                  llx.ArrayData(acls, types.Dict),
		"groups":                llx.MapData(stringSliceMapToAny(acl.Groups), types.Array(types.String)),
		"hosts":                 llx.MapData(stringMapToAny(acl.Hosts), types.String),
		"tagOwners":             llx.MapData(stringSliceMapToAny(acl.TagOwners), types.Array(types.String)),
		"ssh":                   llx.ArrayData(ssh, types.Dict),
		"tests":                 llx.ArrayData(tests, types.Dict),
		"nodeAttrs":             llx.ArrayData(nodeAttrs, types.Dict),
		"autoApproverExitNodes": llx.ArrayData(autoApproverExitNodes, types.String),
		"autoApproverRoutes":    llx.MapData(autoApproverRoutes, types.Array(types.String)),
		"defaultSourcePosture":  llx.ArrayData(stringSliceToAny(acl.DefaultSourcePosture), types.String),
		"postures":              llx.MapData(stringSliceMapToAny(acl.Postures), types.Array(types.String)),
		"disableIPv4":           llx.BoolData(acl.DisableIPv4),
		"oneCGNATRoute":         llx.StringData(acl.OneCGNATRoute),
		"randomizeClientPort":   llx.BoolData(acl.RandomizeClientPort),
		"etag":                  llx.StringData(acl.ETag),
	})
}

// raw lazily fetches the raw HuJSON representation of the policy.
// Note: the `etag` field reflects the structured policy snapshot returned by
// PolicyFile().Get() at resource creation, not this raw HuJSON fetch — the two
// are independent API calls and may briefly diverge if the policy is edited
// between them.
func (a *mqlTailscaleAclPolicy) raw() (string, error) {
	if a.rawFetched {
		return a.rawValue, nil
	}
	a.rawLock.Lock()
	defer a.rawLock.Unlock()
	if a.rawFetched {
		return a.rawValue, nil
	}
	conn := a.MqlRuntime.Connection.(*connection.TailscaleConnection)
	raw, err := conn.Client().PolicyFile().Raw(context.Background())
	if err != nil {
		return "", err
	}
	a.rawValue = raw.HuJSON
	a.rawFetched = true
	return a.rawValue, nil
}

// structSliceToDictSlice JSON-round-trips a slice of policy structs into a
// slice of generic maps, suitable for use as MQL []dict. Field names match
// the JSON tags on the Tailscale SDK types (e.g. "src", "dst", "ports",
// "action", "proto", "users"). Any conversion error is propagated so
// security-sensitive policy entries are never silently dropped.
func structSliceToDictSlice[T any](in []T) ([]any, error) {
	if len(in) == 0 {
		return []any{}, nil
	}
	out := make([]any, 0, len(in))
	for i := range in {
		b, err := json.Marshal(in[i])
		if err != nil {
			return nil, fmt.Errorf("tailscale: failed to marshal policy entry at index %d: %w", i, err)
		}
		var m map[string]any
		if err := json.Unmarshal(b, &m); err != nil {
			return nil, fmt.Errorf("tailscale: failed to unmarshal policy entry at index %d: %w", i, err)
		}
		out = append(out, m)
	}
	return out, nil
}

func stringSliceMapToAny(in map[string][]string) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		arr := make([]any, 0, len(v))
		for _, s := range v {
			arr = append(arr, s)
		}
		out[k] = arr
	}
	return out
}

func stringMapToAny(in map[string]string) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func stringSliceToAny(in []string) []any {
	out := make([]any, 0, len(in))
	for _, s := range in {
		out = append(out, s)
	}
	return out
}
