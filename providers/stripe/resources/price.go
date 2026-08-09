// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/stripe/connection"
)

// mqlStripePriceInternal caches the raw product id so the typed product()
// accessor can resolve it lazily.
type mqlStripePriceInternal struct {
	cacheProductID string
}

type priceRecord struct {
	ID         string `json:"id"`
	Active     bool   `json:"active"`
	Currency   string `json:"currency"`
	UnitAmount int64  `json:"unit_amount"`
	Type       string `json:"type"`
	Nickname   string `json:"nickname"`
	Livemode   bool   `json:"livemode"`
	Created    int64  `json:"created"`
	Product    string `json:"product"`
	Recurring  *struct {
		Interval      string `json:"interval"`
		IntervalCount int64  `json:"interval_count"`
	} `json:"recurring"`
}

func (r priceRecord) GetID() string { return r.ID }

func (c *mqlStripePrice) id() (string, error) {
	return c.Id.Data, c.Id.Error
}

func listPrices(runtime *plugin.Runtime) ([]any, error) {
	c := stripeConn(runtime)
	records, err := connection.List[priceRecord](context.Background(), c, "/v1/prices", nil)
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(records))
	for i := range records {
		rec := records[i]

		interval := ""
		var intervalCount int64
		if rec.Recurring != nil {
			interval = rec.Recurring.Interval
			intervalCount = rec.Recurring.IntervalCount
		}

		mqlPrice, err := CreateResource(runtime, "stripe.price", map[string]*llx.RawData{
			"id":                     llx.StringData(rec.ID),
			"active":                 llx.BoolData(rec.Active),
			"currency":               llx.StringData(rec.Currency),
			"unitAmount":             llx.IntData(rec.UnitAmount),
			"type":                   llx.StringData(rec.Type),
			"recurringInterval":      llx.StringData(interval),
			"recurringIntervalCount": llx.IntData(intervalCount),
			"nickname":               llx.StringData(rec.Nickname),
			"livemode":               llx.BoolData(rec.Livemode),
			"created":                llx.TimeDataPtr(unixPtr(rec.Created)),
		})
		if err != nil {
			return nil, err
		}
		mqlPrice.(*mqlStripePrice).cacheProductID = rec.Product
		res = append(res, mqlPrice)
	}
	return res, nil
}

// product resolves the product this price is attached to.
func (c *mqlStripePrice) product() (*mqlStripeProduct, error) {
	if c.cacheProductID == "" {
		c.Product.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(c.MqlRuntime, "stripe.product", map[string]*llx.RawData{
		"id": llx.StringData(c.cacheProductID),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlStripeProduct), nil
}
