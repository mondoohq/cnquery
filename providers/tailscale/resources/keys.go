// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"time"

	"go.mondoo.com/mql/llx"
	"go.mondoo.com/mql/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/providers/tailscale/connection"
	"go.mondoo.com/mql/types"
	tsclient "tailscale.com/client/tailscale/v2"
)

func (r *mqlTailscaleAuthKey) id() (string, error) {
	return "tailscale/authKey/" + r.Id.Data, nil
}

// timeIsSet reports whether a Tailscale timestamp represents a real value.
// Tailscale encodes "unset" timestamps as the Go zero time (0001-01-01), which
// the resource carries as a nil or zero-valued *time.Time. A genuine Unix epoch
// 0 is a real instant in Go terms and is therefore reported as set.
func timeIsSet(t *time.Time) bool {
	return t != nil && !t.IsZero()
}

// hasExpiration reports whether the key has an expiration set. A key with no
// expiration carries the zero time in `expires`.
func (r *mqlTailscaleAuthKey) hasExpiration() (bool, error) {
	if r.Expires.Error != nil {
		return false, r.Expires.Error
	}
	return timeIsSet(r.Expires.Data), nil
}

// isRevoked reports whether the key has been revoked. A key that has not been
// revoked carries the zero time in `revoked`.
func (r *mqlTailscaleAuthKey) isRevoked() (bool, error) {
	if r.Revoked.Error != nil {
		return false, r.Revoked.Error
	}
	return timeIsSet(r.Revoked.Data), nil
}

func initTailscaleAuthKey(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	id, err := requiredStringArg(args, "id")
	if err != nil {
		return nil, nil, err
	}

	conn := runtime.Connection.(*connection.TailscaleConnection)
	key, err := conn.Client().Keys().Get(context.Background(), id)
	if err != nil {
		return nil, nil, err
	}

	resource, err := createTailscaleAuthKeyResource(runtime, key)
	if err != nil {
		return nil, nil, err
	}

	return args, resource, nil
}

func createTailscaleAuthKeyResource(runtime *plugin.Runtime, key *tsclient.Key) (plugin.Resource, error) {
	if key == nil {
		return nil, errors.New("tailscale.authKey: nil key returned by API")
	}
	caps := key.Capabilities.Devices.Create
	tags := make([]any, 0, len(caps.Tags))
	for _, t := range caps.Tags {
		tags = append(tags, t)
	}
	scopes := make([]any, 0, len(key.Scopes))
	for _, s := range key.Scopes {
		scopes = append(scopes, s)
	}
	claimRules := make(map[string]any, len(key.CustomClaimRules))
	for k, v := range key.CustomClaimRules {
		claimRules[k] = v
	}

	// An OAuth client or federated identity carries its ACL tags on the key
	// itself, while a pre-auth key carries them under the device-create
	// capability. Only one of the two is ever populated.
	if len(key.Tags) > 0 {
		tags = tags[:0]
		for _, t := range key.Tags {
			tags = append(tags, t)
		}
	}

	return CreateResource(runtime, "tailscale.authKey", map[string]*llx.RawData{
		"id":               llx.StringData(key.ID),
		"keyType":          llx.StringData(key.KeyType),
		"description":      llx.StringData(key.Description),
		"userId":           llx.StringData(key.UserID),
		"scopes":           llx.ArrayData(scopes, types.String),
		"audience":         llx.StringData(key.Audience),
		"issuer":           llx.StringData(key.Issuer),
		"subject":          llx.StringData(key.Subject),
		"customClaimRules": llx.MapData(claimRules, types.String),
		"created":          llx.TimeData(key.Created),
		"updated":          llx.TimeData(key.Updated),
		"expires":          llx.TimeData(key.Expires),
		"revoked":          llx.TimeData(key.Revoked),
		"invalid":          llx.BoolData(key.Invalid),
		"reusable":         llx.BoolData(caps.Reusable),
		"ephemeral":        llx.BoolData(caps.Ephemeral),
		"preauthorized":    llx.BoolData(caps.Preauthorized),
		"tags":             llx.ArrayData(tags, types.String),
	})
}

// authKeys lists every auth key (pre-auth key) in the tailnet. The List API
// returns IDs only, so each key's metadata is fetched individually via Get.
func (t *mqlTailscale) authKeys() ([]any, error) {
	conn := t.MqlRuntime.Connection.(*connection.TailscaleConnection)
	ctx := context.Background()

	keys, err := conn.Client().Keys().List(ctx, true)
	if err != nil {
		return nil, err
	}

	resources := []any{}
	for _, k := range keys {
		full, err := conn.Client().Keys().Get(ctx, k.ID)
		if err != nil {
			return nil, err
		}
		resource, err := createTailscaleAuthKeyResource(t.MqlRuntime, full)
		if err != nil {
			return nil, err
		}
		resources = append(resources, resource)
	}
	return resources, nil
}
