// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/stripe/connection"
)

type customerRecord struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Currency    string `json:"currency"`
	Balance     int64  `json:"balance"`
	Delinquent  bool   `json:"delinquent"`
	Livemode    bool   `json:"livemode"`
	Created     int64  `json:"created"`
}

func (r customerRecord) GetID() string { return r.ID }

func (c *mqlStripeCustomer) id() (string, error) {
	return c.Id.Data, c.Id.Error
}

func newMqlStripeCustomer(runtime *plugin.Runtime, rec *customerRecord) (*mqlStripeCustomer, error) {
	res, err := CreateResource(runtime, "stripe.customer", map[string]*llx.RawData{
		"id":          llx.StringData(rec.ID),
		"email":       llx.StringData(rec.Email),
		"name":        llx.StringData(rec.Name),
		"description": llx.StringData(rec.Description),
		"currency":    llx.StringData(rec.Currency),
		"balance":     llx.IntData(rec.Balance),
		"delinquent":  llx.BoolData(rec.Delinquent),
		"livemode":    llx.BoolData(rec.Livemode),
		"created":     llx.TimeDataPtr(unixPtr(rec.Created)),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlStripeCustomer), nil
}

// initStripeCustomer resolves a customer by its id, either for a direct
// stripe.customer(id: "cus_...") query or for a typed reference from another
// resource. It fetches the customer from the API when only an id is supplied.
func initStripeCustomer(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	// Fast path: the caller already supplied a populated record.
	if len(args) > 1 {
		return args, nil, nil
	}

	idArg, ok := args["id"]
	if !ok || idArg.Value == nil {
		return nil, nil, fmt.Errorf("stripe.customer requires an id")
	}
	id, _ := idArg.Value.(string)
	if id == "" {
		return nil, nil, fmt.Errorf("stripe.customer requires an id")
	}

	c := stripeConn(runtime)
	var rec customerRecord
	if err := c.Get(context.Background(), "/v1/customers/"+id, nil, &rec); err != nil {
		if connection.IsForbidden(err) {
			return nil, nil, fmt.Errorf("access denied reading stripe.customer %q: %w", id, err)
		}
		return nil, nil, err
	}
	if rec.ID == "" {
		return nil, nil, fmt.Errorf("stripe.customer with id %q not found", id)
	}

	res, err := newMqlStripeCustomer(runtime, &rec)
	if err != nil {
		return nil, nil, err
	}
	return nil, res, nil
}
