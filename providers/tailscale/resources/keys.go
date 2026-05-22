// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"

	tsclient "github.com/tailscale/tailscale-client-go/v2"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/tailscale/connection"
	"go.mondoo.com/mql/v13/types"
)

func (r *mqlTailscaleAuthKey) id() (string, error) {
	return "tailscale/authKey/" + r.Id.Data, nil
}

func initTailscaleAuthKey(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	id, ok := args["id"]
	if !ok {
		return nil, nil, errors.New("missing required argument 'id'")
	}

	conn := runtime.Connection.(*connection.TailscaleConnection)
	key, err := conn.Client().Keys().Get(context.Background(), id.Value.(string))
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
	return CreateResource(runtime, "tailscale.authKey", map[string]*llx.RawData{
		"id":            llx.StringData(key.ID),
		"description":   llx.StringData(key.Description),
		"userId":        llx.StringData(key.UserID),
		"created":       llx.TimeData(key.Created),
		"expires":       llx.TimeData(key.Expires),
		"revoked":       llx.TimeData(key.Revoked),
		"invalid":       llx.BoolData(key.Invalid),
		"reusable":      llx.BoolData(caps.Reusable),
		"ephemeral":     llx.BoolData(caps.Ephemeral),
		"preauthorized": llx.BoolData(caps.Preauthorized),
		"tags":          llx.ArrayData(tags, types.String),
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
