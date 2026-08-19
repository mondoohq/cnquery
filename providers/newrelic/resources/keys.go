// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strconv"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

// mqlNewrelicApiKeyInternal keeps the identifiers a key's references resolve
// against. Neither is the keystring: the provider never asks New Relic for it,
// so there is nothing here to leak.
type mqlNewrelicApiKeyInternal struct {
	cachedAccountID int
	cachedUserID    int
}

// apiKeyID builds the cache key of an API key. Ingest keys and user keys are
// numbered in separate spaces and can collide, so the type is part of the key.
func apiKeyID(key apiKey) string {
	return "apiKey/" + key.Type + "/" + key.ID
}

func newAPIKeyResource(runtime *plugin.Runtime, key apiKey) (*mqlNewrelicApiKey, error) {
	res, err := CreateResource(runtime, "newrelic.apiKey", map[string]*llx.RawData{
		"__id":       llx.StringData(apiKeyID(key)),
		"id":         llx.StringData(key.ID),
		"name":       llx.StringData(key.Name),
		"notes":      llx.StringData(key.Notes),
		"type":       llx.StringData(key.Type),
		"ingestType": llx.StringData(key.IngestType),
		"createdAt":  llx.TimeDataPtr(key.CreatedAt.Time()),
	})
	if err != nil {
		return nil, err
	}

	mqlKey := res.(*mqlNewrelicApiKey)
	mqlKey.cachedAccountID = key.AccountID
	mqlKey.cachedUserID = key.UserID
	return mqlKey, nil
}

func (r *mqlNewrelicApiKey) account() (*mqlNewrelicAccount, error) {
	account, found, err := resolveAccount(r.MqlRuntime, r.cachedAccountID)
	if err != nil {
		return nil, err
	}
	if !found {
		r.Account.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return account, nil
}

func (r *mqlNewrelicApiKey) userRef() (*mqlNewrelicUser, error) {
	if r.cachedUserID <= 0 {
		// An ingest key acts as no person, so there is no owner to resolve.
		r.UserRef.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}

	user, found, err := resolveUser(r.MqlRuntime, strconv.Itoa(r.cachedUserID))
	if err != nil {
		return nil, err
	}
	if !found {
		r.UserRef.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return user, nil
}
