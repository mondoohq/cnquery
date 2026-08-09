// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/types"
)

type accountRecord struct {
	ID               string            `json:"id"`
	Type             string            `json:"type"`
	BusinessType     string            `json:"business_type"`
	Country          string            `json:"country"`
	DefaultCurrency  string            `json:"default_currency"`
	Email            string            `json:"email"`
	ChargesEnabled   bool              `json:"charges_enabled"`
	PayoutsEnabled   bool              `json:"payouts_enabled"`
	DetailsSubmitted bool              `json:"details_submitted"`
	Created          int64             `json:"created"`
	Capabilities     map[string]string `json:"capabilities"`
	BusinessProfile  *struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"business_profile"`
}

func (c *mqlStripeAccount) id() (string, error) {
	return c.Id.Data, c.Id.Error
}

// fetchAccount retrieves the account associated with the connection's key.
func fetchAccount(runtime *plugin.Runtime) (*mqlStripeAccount, error) {
	c := stripeConn(runtime)

	var rec accountRecord
	if err := c.Get(context.Background(), "/v1/account", nil, &rec); err != nil {
		return nil, err
	}

	businessName := ""
	businessURL := ""
	if rec.BusinessProfile != nil {
		businessName = rec.BusinessProfile.Name
		businessURL = rec.BusinessProfile.URL
	}

	res, err := CreateResource(runtime, "stripe.account", map[string]*llx.RawData{
		"id":               llx.StringData(rec.ID),
		"type":             llx.StringData(rec.Type),
		"businessType":     llx.StringData(rec.BusinessType),
		"country":          llx.StringData(rec.Country),
		"defaultCurrency":  llx.StringData(rec.DefaultCurrency),
		"email":            llx.StringData(rec.Email),
		"chargesEnabled":   llx.BoolData(rec.ChargesEnabled),
		"payoutsEnabled":   llx.BoolData(rec.PayoutsEnabled),
		"detailsSubmitted": llx.BoolData(rec.DetailsSubmitted),
		"businessName":     llx.StringData(businessName),
		"businessUrl":      llx.StringData(businessURL),
		"capabilities":     llx.MapData(mapStrToAny(rec.Capabilities), types.String),
		"created":          llx.TimeDataPtr(unixPtr(rec.Created)),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlStripeAccount), nil
}
