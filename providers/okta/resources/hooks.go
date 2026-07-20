// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"encoding/json"

	"github.com/okta/okta-sdk-golang/v5/okta"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/okta/connection"
	"go.mondoo.com/mql/v13/types"
)

// --- event hooks ---

func (o *mqlOkta) eventHooks() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OktaConnection)
	client := conn.Client()

	ctx := context.Background()
	slice, resp, err := client.EventHookAPI.ListEventHooks(ctx).Execute()
	if err != nil {
		return nil, err
	}

	list := []any{}
	appendEntry := func(datalist []okta.EventHook) error {
		for i := range datalist {
			r, err := newMqlOktaEventHook(o.MqlRuntime, &datalist[i])
			if err != nil {
				return err
			}
			list = append(list, r)
		}
		return nil
	}

	if err := appendEntry(slice); err != nil {
		return nil, err
	}
	for resp != nil && resp.HasNextPage() {
		var page []okta.EventHook
		resp, err = resp.Next(&page)
		if err != nil {
			return nil, err
		}
		if err := appendEntry(page); err != nil {
			return nil, err
		}
	}
	return list, nil
}

func newMqlOktaEventHook(runtime *plugin.Runtime, entry *okta.EventHook) (any, error) {
	authScheme, err := convert.JsonToDict(entry.Channel.Config.AuthScheme)
	if err != nil {
		return nil, err
	}
	headers, err := convert.JsonToDictSlice(entry.Channel.Config.Headers)
	if err != nil {
		return nil, err
	}

	return CreateResource(runtime, "okta.eventHook", map[string]*llx.RawData{
		"id":                 llx.StringData(oktaStr(entry.Id)),
		"name":               llx.StringData(entry.Name),
		"description":        llx.StringData(oktaStr(entry.Description.Get())),
		"status":             llx.StringData(oktaStr(entry.Status)),
		"verificationStatus": llx.StringData(oktaStr(entry.VerificationStatus)),
		"events":             llx.ArrayData(convert.SliceAnyToInterface(entry.Events.Items), types.String),
		"channelType":        llx.StringData(entry.Channel.Type),
		"channelUri":         llx.StringData(entry.Channel.Config.Uri),
		"channelAuthScheme":  llx.DictData(authScheme),
		"headers":            llx.ArrayData(headers, types.Dict),
		"created":            llx.TimeDataPtr(entry.Created),
		"lastUpdated":        llx.TimeDataPtr(entry.LastUpdated),
	})
}

func (o *mqlOktaEventHook) id() (string, error) {
	return "okta.eventHook/" + o.Id.Data, o.Id.Error
}

// --- inline hooks ---

// oktaInlineHookRaw flattens the inline hook's channel, whose config (uri,
// authScheme, headers) the v5 SDK carries in the generic channel's untyped
// AdditionalProperties rather than a typed field. Re-marshaling to JSON gives
// one stable path to the fields regardless of the concrete channel variant.
type oktaInlineHookRaw struct {
	Channel struct {
		Type   string `json:"type"`
		Config struct {
			Uri        string         `json:"uri"`
			AuthScheme map[string]any `json:"authScheme"`
		} `json:"config"`
	} `json:"channel"`
}

func (o *mqlOkta) inlineHooks() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OktaConnection)
	client := conn.Client()

	ctx := context.Background()
	slice, resp, err := client.InlineHookAPI.ListInlineHooks(ctx).Execute()
	if err != nil {
		return nil, err
	}

	list := []any{}
	appendEntry := func(datalist []okta.InlineHook) error {
		for i := range datalist {
			r, err := newMqlOktaInlineHook(o.MqlRuntime, &datalist[i])
			if err != nil {
				return err
			}
			list = append(list, r)
		}
		return nil
	}

	if err := appendEntry(slice); err != nil {
		return nil, err
	}
	for resp != nil && resp.HasNextPage() {
		var page []okta.InlineHook
		resp, err = resp.Next(&page)
		if err != nil {
			return nil, err
		}
		if err := appendEntry(page); err != nil {
			return nil, err
		}
	}
	return list, nil
}

func newMqlOktaInlineHook(runtime *plugin.Runtime, entry *okta.InlineHook) (any, error) {
	var channelType, channelUri string
	var authScheme any
	if entry.Channel != nil {
		raw, err := json.Marshal(entry.Channel)
		if err != nil {
			return nil, err
		}
		var parsed oktaInlineHookRaw
		if err := json.Unmarshal(raw, &parsed); err != nil {
			return nil, err
		}
		channelType = parsed.Channel.Type
		channelUri = parsed.Channel.Config.Uri
		authScheme = parsed.Channel.Config.AuthScheme
	}
	authSchemeDict, err := convert.JsonToDict(authScheme)
	if err != nil {
		return nil, err
	}

	metadata := map[string]any{}
	if entry.Metadata != nil {
		for k, v := range *entry.Metadata {
			metadata[k] = v
		}
	}

	return CreateResource(runtime, "okta.inlineHook", map[string]*llx.RawData{
		"id":                llx.StringData(oktaStr(entry.Id)),
		"name":              llx.StringData(oktaStr(entry.Name)),
		"type":              llx.StringData(oktaStr(entry.Type)),
		"status":            llx.StringData(oktaStr(entry.Status)),
		"version":           llx.StringData(oktaStr(entry.Version)),
		"channelType":       llx.StringData(channelType),
		"channelUri":        llx.StringData(channelUri),
		"channelAuthScheme": llx.DictData(authSchemeDict),
		"metadata":          llx.MapData(metadata, types.String),
		"created":           llx.TimeDataPtr(entry.Created),
		"lastUpdated":       llx.TimeDataPtr(entry.LastUpdated),
	})
}

func (o *mqlOktaInlineHook) id() (string, error) {
	return "okta.inlineHook/" + o.Id.Data, o.Id.Error
}

// --- hook keys ---

func (o *mqlOkta) hookKeys() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OktaConnection)
	client := conn.Client()

	ctx := context.Background()
	slice, resp, err := client.HookKeyAPI.ListHookKeys(ctx).Execute()
	if err != nil {
		return nil, err
	}

	list := []any{}
	appendEntry := func(datalist []okta.HookKey) error {
		for i := range datalist {
			r, err := newMqlOktaHookKey(o.MqlRuntime, &datalist[i])
			if err != nil {
				return err
			}
			list = append(list, r)
		}
		return nil
	}

	if err := appendEntry(slice); err != nil {
		return nil, err
	}
	for resp != nil && resp.HasNextPage() {
		var page []okta.HookKey
		resp, err = resp.Next(&page)
		if err != nil {
			return nil, err
		}
		if err := appendEntry(page); err != nil {
			return nil, err
		}
	}
	return list, nil
}

func newMqlOktaHookKey(runtime *plugin.Runtime, entry *okta.HookKey) (any, error) {
	publicKey, err := convert.JsonToDict(entry.Embedded)
	if err != nil {
		return nil, err
	}

	var isUsed bool
	if entry.IsUsed != nil {
		isUsed = *entry.IsUsed
	}

	return CreateResource(runtime, "okta.hookKey", map[string]*llx.RawData{
		"id":          llx.StringData(oktaStr(entry.Id)),
		"name":        llx.StringData(oktaStr(entry.Name)),
		"keyId":       llx.StringData(oktaStr(entry.KeyId)),
		"isUsed":      llx.BoolData(isUsed),
		"publicKey":   llx.DictData(publicKey),
		"created":     llx.TimeDataPtr(entry.Created),
		"lastUpdated": llx.TimeDataPtr(entry.LastUpdated),
	})
}

func (o *mqlOktaHookKey) id() (string, error) {
	return "okta.hookKey/" + o.Id.Data, o.Id.Error
}
