// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"github.com/okta/okta-sdk-golang/v2/okta"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/okta/connection"
)

func (o *mqlOkta) authenticators() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OktaConnection)
	client := conn.Client()

	ctx := context.Background()
	authenticators, _, err := client.Authenticator.ListAuthenticators(ctx)
	if err != nil {
		return nil, err
	}

	if len(authenticators) == 0 {
		return nil, nil
	}

	list := []any{}
	for i := range authenticators {
		r, err := newMqlOktaAuthenticator(o.MqlRuntime, authenticators[i])
		if err != nil {
			return nil, err
		}
		list = append(list, r)
	}
	return list, nil
}

func newMqlOktaAuthenticator(runtime *plugin.Runtime, entry *okta.Authenticator) (*mqlOktaAuthenticator, error) {
	args := map[string]*llx.RawData{
		"id":          llx.StringData(entry.Id),
		"key":         llx.StringData(entry.Key),
		"name":        llx.StringData(entry.Name),
		"type":        llx.StringData(entry.Type),
		"status":      llx.StringData(entry.Status),
		"created":     llx.TimeDataPtr(entry.Created),
		"lastUpdated": llx.TimeDataPtr(entry.LastUpdated),
	}

	if entry.Provider != nil {
		args["providerType"] = llx.StringData(entry.Provider.Type)
		providerCfg, err := convert.JsonToDict(entry.Provider.Configuration)
		if err != nil {
			return nil, err
		}
		args["providerConfiguration"] = llx.DictData(providerCfg)
	} else {
		args["providerType"] = llx.StringData("")
		args["providerConfiguration"] = llx.DictData(map[string]any{})
	}

	if entry.Settings != nil {
		settings, err := convert.JsonToDict(entry.Settings)
		if err != nil {
			return nil, err
		}
		args["settings"] = llx.DictData(settings)
		args["allowedFor"] = llx.StringData(entry.Settings.AllowedFor)
		args["userVerification"] = llx.StringData(entry.Settings.UserVerification)
		if entry.Settings.TokenLifetimeInMinutesPtr != nil {
			args["tokenLifetimeInMinutes"] = llx.IntData(*entry.Settings.TokenLifetimeInMinutesPtr)
		} else {
			args["tokenLifetimeInMinutes"] = llx.IntData(0)
		}
	} else {
		args["settings"] = llx.DictData(map[string]any{})
		args["allowedFor"] = llx.StringData("")
		args["userVerification"] = llx.StringData("")
		args["tokenLifetimeInMinutes"] = llx.IntData(0)
	}

	r, err := CreateResource(runtime, "okta.authenticator", args)
	if err != nil {
		return nil, err
	}
	return r.(*mqlOktaAuthenticator), nil
}

func (o *mqlOktaAuthenticator) id() (string, error) {
	return "okta.authenticator/" + o.Id.Data, o.Id.Error
}
