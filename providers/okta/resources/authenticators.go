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

type mqlOktaAuthenticatorInternal struct {
	provider *okta.AuthenticatorProvider
	settings *okta.AuthenticatorSettings
}

func (o *mqlOkta) authenticators() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OktaConnection)
	client := conn.Client()

	ctx := context.Background()
	authenticators, resp, err := client.Authenticator.ListAuthenticators(ctx)
	if err != nil {
		return nil, err
	}

	list := []any{}
	appendEntries := func(entries []*okta.Authenticator) error {
		for i := range entries {
			r, err := newMqlOktaAuthenticator(o.MqlRuntime, entries[i])
			if err != nil {
				return err
			}
			list = append(list, r)
		}
		return nil
	}

	if err := appendEntries(authenticators); err != nil {
		return nil, err
	}

	for resp != nil && resp.HasNextPage() {
		var page []*okta.Authenticator
		resp, err = resp.Next(ctx, &page)
		if err != nil {
			return nil, err
		}
		if err := appendEntries(page); err != nil {
			return nil, err
		}
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

	if entry.Settings != nil {
		settings, err := convert.JsonToDict(entry.Settings)
		if err != nil {
			return nil, err
		}
		args["settings"] = llx.DictData(settings)
	} else {
		args["settings"] = llx.DictData(map[string]any{})
	}

	r, err := CreateResource(runtime, "okta.authenticator", args)
	if err != nil {
		return nil, err
	}
	mqlAuth := r.(*mqlOktaAuthenticator)
	mqlAuth.provider = entry.Provider
	mqlAuth.settings = entry.Settings
	return mqlAuth, nil
}

func (o *mqlOktaAuthenticator) id() (string, error) {
	return "okta.authenticator/" + o.Id.Data, o.Id.Error
}

func (o *mqlOktaAuthenticator) providerType() (string, error) {
	if o.provider == nil || o.provider.Type == "" {
		o.ProviderType.State = plugin.StateIsSet | plugin.StateIsNull
		return "", nil
	}
	return o.provider.Type, nil
}

func (o *mqlOktaAuthenticator) providerConfiguration() (any, error) {
	if o.provider == nil || o.provider.Configuration == nil {
		o.ProviderConfiguration.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return convert.JsonToDict(o.provider.Configuration)
}

func (o *mqlOktaAuthenticator) allowedFor() (string, error) {
	if o.settings == nil || o.settings.AllowedFor == "" {
		o.AllowedFor.State = plugin.StateIsSet | plugin.StateIsNull
		return "", nil
	}
	return o.settings.AllowedFor, nil
}

func (o *mqlOktaAuthenticator) tokenLifetimeInMinutes() (int64, error) {
	if o.settings == nil || o.settings.TokenLifetimeInMinutesPtr == nil {
		o.TokenLifetimeInMinutes.State = plugin.StateIsSet | plugin.StateIsNull
		return 0, nil
	}
	return *o.settings.TokenLifetimeInMinutesPtr, nil
}

func (o *mqlOktaAuthenticator) userVerification() (string, error) {
	if o.settings == nil || o.settings.UserVerification == "" {
		o.UserVerification.State = plugin.StateIsSet | plugin.StateIsNull
		return "", nil
	}
	return o.settings.UserVerification, nil
}
