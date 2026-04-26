// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"github.com/okta/okta-sdk-golang/v2/okta"
	"github.com/okta/okta-sdk-golang/v2/okta/query"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/okta/connection"
	"go.mondoo.com/mql/v13/types"
)

func (o *mqlOkta) applications() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OktaConnection)
	client := conn.Client()

	ctx := context.Background()
	appSetSlice, resp, err := client.Application.ListApplications(
		ctx,
		query.NewQueryParams(
			query.WithLimit(queryLimit),
		),
	)
	if err != nil {
		return nil, err
	}

	if len(appSetSlice) == 0 {
		return nil, nil
	}

	list := []any{}
	appendEntry := func(datalist []okta.App) error {
		for i := range datalist {
			entry := datalist[i]
			if entry.IsApplicationInstance() {
				app := entry.(*okta.Application)
				r, err := newMqlOktaApplication(o.MqlRuntime, app)
				if err != nil {
					return err
				}
				list = append(list, r)
			}
		}
		return nil
	}

	err = appendEntry(appSetSlice)
	if err != nil {
		return nil, err
	}

	for resp != nil && resp.HasNextPage() {
		var userSetSlice []okta.App
		resp, err = resp.Next(ctx, &userSetSlice)
		if err != nil {
			return nil, err
		}
		err = appendEntry(userSetSlice)
		if err != nil {
			return nil, err
		}
	}
	return list, nil
}

func newMqlOktaApplication(runtime *plugin.Runtime, entry *okta.Application) (any, error) {
	credentials, err := convert.JsonToDict(entry.Credentials)
	if err != nil {
		return nil, err
	}

	licensing, err := convert.JsonToDict(entry.Licensing)
	if err != nil {
		return nil, err
	}

	profile, err := convert.JsonToDict(entry.Profile)
	if err != nil {
		return nil, err
	}

	settings, err := convert.JsonToDict(entry.Settings)
	if err != nil {
		return nil, err
	}

	visibility, err := convert.JsonToDict(entry.Visibility)
	if err != nil {
		return nil, err
	}

	return CreateResource(runtime, "okta.application", map[string]*llx.RawData{
		"id":          llx.StringData(entry.Id),
		"name":        llx.StringData(entry.Name),
		"label":       llx.StringData(entry.Label),
		"created":     llx.TimeDataPtr(entry.Created),
		"lastUpdated": llx.TimeDataPtr(entry.LastUpdated),
		"credentials": llx.DictData(credentials),
		"features":    llx.ArrayData(convert.SliceAnyToInterface(entry.Features), types.String),
		"licensing":   llx.DictData(licensing),
		"profile":     llx.DictData(profile),
		"settings":    llx.DictData(settings),
		"signOnMode":  llx.StringData(entry.SignOnMode),
		"status":      llx.StringData(entry.Status),
		"visibility":  llx.DictData(visibility),
	})
}

func (o *mqlOktaApplication) id() (string, error) {
	return "okta.application/" + o.Id.Data, o.Id.Error
}

// signingKeys returns the X.509 signing certificates / JWKs published for this app.
// These are used to sign SAML assertions and OIDC tokens, so checking expiresAt and
// status is essential for catching expired/about-to-expire signing certs.
func (o *mqlOktaApplication) signingKeys() ([]any, error) {
	if o.Id.Error != nil {
		return nil, o.Id.Error
	}
	if o.Id.Data == "" {
		return []any{}, nil
	}

	conn := o.MqlRuntime.Connection.(*connection.OktaConnection)
	client := conn.Client()

	ctx := context.Background()
	keys, resp, err := client.Application.ListApplicationKeys(ctx, o.Id.Data)
	if err != nil {
		return nil, err
	}

	list := []any{}
	appendKeys := func(entries []*okta.JsonWebKey) error {
		for i := range entries {
			k := entries[i]
			if k == nil {
				continue
			}
			r, err := CreateResource(o.MqlRuntime, "okta.application.key", map[string]*llx.RawData{
				"applicationId": llx.StringData(o.Id.Data),
				"kid":           llx.StringData(k.Kid),
				"status":        llx.StringData(k.Status),
				"alg":           llx.StringData(k.Alg),
				"kty":           llx.StringData(k.Kty),
				"use":           llx.StringData(k.Use),
				"keyOps":        llx.ArrayData(convert.SliceAnyToInterface(k.KeyOps), types.String),
				"created":       llx.TimeDataPtr(k.Created),
				"lastUpdated":   llx.TimeDataPtr(k.LastUpdated),
				"expiresAt":     llx.TimeDataPtr(k.ExpiresAt),
				"x5c":           llx.ArrayData(convert.SliceAnyToInterface(k.X5c), types.String),
				"x5t":           llx.StringData(k.X5t),
				"x5tS256":       llx.StringData(k.X5tS256),
				"n":             llx.StringData(k.N),
				"e":             llx.StringData(k.E),
			})
			if err != nil {
				return err
			}
			list = append(list, r)
		}
		return nil
	}

	if err := appendKeys(keys); err != nil {
		return nil, err
	}

	for resp != nil && resp.HasNextPage() {
		var page []*okta.JsonWebKey
		resp, err = resp.Next(ctx, &page)
		if err != nil {
			return nil, err
		}
		if err := appendKeys(page); err != nil {
			return nil, err
		}
	}
	return list, nil
}

func (o *mqlOktaApplicationKey) id() (string, error) {
	return "okta.application.key/" + o.ApplicationId.Data + "/" + o.Kid.Data, o.Kid.Error
}
