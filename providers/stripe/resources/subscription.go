// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"net/url"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/stripe/connection"
)

// mqlStripeSubscriptionInternal caches the raw customer id so the typed
// customer() accessor can resolve it lazily.
type mqlStripeSubscriptionInternal struct {
	cacheCustomerID string
}

type subscriptionRecord struct {
	ID                 string `json:"id"`
	Status             string `json:"status"`
	Currency           string `json:"currency"`
	CancelAtPeriodEnd  bool   `json:"cancel_at_period_end"`
	Livemode           bool   `json:"livemode"`
	Created            int64  `json:"created"`
	CurrentPeriodStart int64  `json:"current_period_start"`
	CurrentPeriodEnd   int64  `json:"current_period_end"`
	CanceledAt         int64  `json:"canceled_at"`
	Customer           string `json:"customer"`
}

func (r subscriptionRecord) GetID() string { return r.ID }

func (c *mqlStripeSubscription) id() (string, error) {
	return c.Id.Data, c.Id.Error
}

func listSubscriptions(runtime *plugin.Runtime) ([]any, error) {
	c := stripeConn(runtime)
	// status=all includes canceled and incomplete subscriptions, which the
	// default listing omits.
	query := url.Values{}
	query.Set("status", "all")
	records, err := connection.List[subscriptionRecord](context.Background(), c, "/v1/subscriptions", query)
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(records))
	for i := range records {
		rec := records[i]
		mqlSub, err := CreateResource(runtime, "stripe.subscription", map[string]*llx.RawData{
			"id":                 llx.StringData(rec.ID),
			"status":             llx.StringData(rec.Status),
			"currency":           llx.StringData(rec.Currency),
			"cancelAtPeriodEnd":  llx.BoolData(rec.CancelAtPeriodEnd),
			"livemode":           llx.BoolData(rec.Livemode),
			"created":            llx.TimeDataPtr(unixPtr(rec.Created)),
			"currentPeriodStart": llx.TimeDataPtr(unixPtr(rec.CurrentPeriodStart)),
			"currentPeriodEnd":   llx.TimeDataPtr(unixPtr(rec.CurrentPeriodEnd)),
			"canceledAt":         llx.TimeDataPtr(unixPtr(rec.CanceledAt)),
		})
		if err != nil {
			return nil, err
		}
		mqlSub.(*mqlStripeSubscription).cacheCustomerID = rec.Customer
		res = append(res, mqlSub)
	}
	return res, nil
}

// customer resolves the customer billed by this subscription.
func (c *mqlStripeSubscription) customer() (*mqlStripeCustomer, error) {
	if c.cacheCustomerID == "" {
		c.Customer.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(c.MqlRuntime, "stripe.customer", map[string]*llx.RawData{
		"id": llx.StringData(c.cacheCustomerID),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlStripeCustomer), nil
}
