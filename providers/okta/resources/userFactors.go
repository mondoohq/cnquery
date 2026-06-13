// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/okta/connection"
)

// mqlOktaUserFactorInternal caches the owning user's id so the typed user()
// accessor can resolve without exposing a deprecated public field.
type mqlOktaUserFactorInternal struct {
	cacheUserId string
}

// userFactorRaw decodes the user factors endpoint directly so we can capture
// the type-specific Profile object that the SDK's UserFactor struct discards.
type userFactorRaw struct {
	Id          string         `json:"id,omitempty"`
	FactorType  string         `json:"factorType,omitempty"`
	Provider    string         `json:"provider,omitempty"`
	Status      string         `json:"status,omitempty"`
	Created     *time.Time     `json:"created,omitempty"`
	LastUpdated *time.Time     `json:"lastUpdated,omitempty"`
	Profile     map[string]any `json:"profile,omitempty"`
}

// factors returns the MFA factors enrolled by this user.
func (o *mqlOktaUser) factors() ([]any, error) {
	if o.Id.Error != nil {
		return nil, o.Id.Error
	}
	if o.Id.Data == "" {
		return []any{}, nil
	}

	conn := o.MqlRuntime.Connection.(*connection.OktaConnection)

	ctx := context.Background()

	// The factors endpoint returns all enrolled factors in a single response;
	// we issue the request directly (rather than client.UserFactorAPI.ListFactors)
	// to capture the per-factorType `profile` object that the SDK's typed
	// UserFactor union discards.
	endpoint := fmt.Sprintf("https://%s/api/v1/users/%s/factors", conn.OrganizationID(), url.PathEscape(o.Id.Data))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "SSWS "+conn.Token())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch user factors: %s: %s", resp.Status, string(body))
	}

	var page []userFactorRaw
	if err := json.Unmarshal(body, &page); err != nil {
		return nil, err
	}

	list := []any{}
	for i := range page {
		r, err := newMqlOktaUserFactor(o.MqlRuntime, o.Id.Data, &page[i])
		if err != nil {
			return nil, err
		}
		list = append(list, r)
	}

	return list, nil
}

func newMqlOktaUserFactor(runtime *plugin.Runtime, userId string, factor *userFactorRaw) (*mqlOktaUserFactor, error) {
	args := map[string]*llx.RawData{
		"id":          llx.StringData(factor.Id),
		"factorType":  llx.StringData(factor.FactorType),
		"provider":    llx.StringData(factor.Provider),
		"status":      llx.StringData(factor.Status),
		"created":     llx.TimeDataPtr(factor.Created),
		"lastUpdated": llx.TimeDataPtr(factor.LastUpdated),
	}
	if factor.Profile != nil {
		args["profile"] = llx.DictData(factor.Profile)
	} else {
		args["profile"] = llx.NilData
	}

	r, err := CreateResource(runtime, "okta.userFactor", args)
	if err != nil {
		return nil, err
	}
	mqlFactor := r.(*mqlOktaUserFactor)
	mqlFactor.cacheUserId = userId
	return mqlFactor, nil
}

func (o *mqlOktaUserFactor) id() (string, error) {
	return "okta.userFactor/" + o.cacheUserId + "/" + o.Id.Data, o.Id.Error
}

// user resolves the typed user this factor belongs to. The runtime caches
// okta.user instances keyed by id, so repeated lookups across factors reuse a
// single GetUser call.
func (o *mqlOktaUserFactor) user() (*mqlOktaUser, error) {
	if o.cacheUserId == "" {
		o.User.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}

	r, err := NewResource(o.MqlRuntime, "okta.user", map[string]*llx.RawData{
		"id": llx.StringData(o.cacheUserId),
	})
	if err != nil {
		return nil, err
	}
	return r.(*mqlOktaUser), nil
}
