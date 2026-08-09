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

type productRecord struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Active      bool   `json:"active"`
	Description string `json:"description"`
	Type        string `json:"type"`
	URL         string `json:"url"`
	Livemode    bool   `json:"livemode"`
	Created     int64  `json:"created"`
	Updated     int64  `json:"updated"`
}

func (r productRecord) GetID() string { return r.ID }

func (c *mqlStripeProduct) id() (string, error) {
	return c.Id.Data, c.Id.Error
}

func newMqlStripeProduct(runtime *plugin.Runtime, rec *productRecord) (*mqlStripeProduct, error) {
	res, err := CreateResource(runtime, "stripe.product", map[string]*llx.RawData{
		"id":          llx.StringData(rec.ID),
		"name":        llx.StringData(rec.Name),
		"active":      llx.BoolData(rec.Active),
		"description": llx.StringData(rec.Description),
		"type":        llx.StringData(rec.Type),
		"url":         llx.StringData(rec.URL),
		"livemode":    llx.BoolData(rec.Livemode),
		"created":     llx.TimeDataPtr(unixPtr(rec.Created)),
		"updated":     llx.TimeDataPtr(unixPtr(rec.Updated)),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlStripeProduct), nil
}

// initStripeProduct resolves a product by its id, for a direct
// stripe.product(id: "prod_...") query or for a typed reference from a price.
func initStripeProduct(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 {
		return args, nil, nil
	}

	idArg, ok := args["id"]
	if !ok || idArg.Value == nil {
		return nil, nil, fmt.Errorf("stripe.product requires an id")
	}
	id, _ := idArg.Value.(string)
	if id == "" {
		return nil, nil, fmt.Errorf("stripe.product requires an id")
	}

	c := stripeConn(runtime)
	var rec productRecord
	if err := c.Get(context.Background(), "/v1/products/"+id, nil, &rec); err != nil {
		if connection.IsForbidden(err) {
			return nil, nil, fmt.Errorf("access denied reading stripe.product %q: %w", id, err)
		}
		return nil, nil, err
	}
	if rec.ID == "" {
		return nil, nil, fmt.Errorf("stripe.product with id %q not found", id)
	}

	res, err := newMqlStripeProduct(runtime, &rec)
	if err != nil {
		return nil, nil, err
	}
	return nil, res, nil
}
