// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/stripe/connection"
)

type reviewRecord struct {
	ID            string `json:"id"`
	Open          bool   `json:"open"`
	Reason        string `json:"reason"`
	OpenedReason  string `json:"opened_reason"`
	ClosedReason  string `json:"closed_reason"`
	IPAddress     string `json:"ip_address"`
	Charge        string `json:"charge"`
	PaymentIntent string `json:"payment_intent"`
	Livemode      bool   `json:"livemode"`
	Created       int64  `json:"created"`
}

func (r reviewRecord) GetID() string { return r.ID }

func (c *mqlStripeReview) id() (string, error) {
	return c.Id.Data, c.Id.Error
}

func listReviews(runtime *plugin.Runtime) ([]any, error) {
	c := stripeConn(runtime)
	records, err := connection.List[reviewRecord](context.Background(), c, "/v1/reviews", nil)
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(records))
	for i := range records {
		rec := records[i]
		mqlReview, err := CreateResource(runtime, "stripe.review", map[string]*llx.RawData{
			"id":            llx.StringData(rec.ID),
			"open":          llx.BoolData(rec.Open),
			"reason":        llx.StringData(rec.Reason),
			"openedReason":  llx.StringData(rec.OpenedReason),
			"closedReason":  llx.StringData(rec.ClosedReason),
			"ipAddress":     llx.StringData(rec.IPAddress),
			"charge":        llx.StringData(rec.Charge),
			"paymentIntent": llx.StringData(rec.PaymentIntent),
			"livemode":      llx.BoolData(rec.Livemode),
			"created":       llx.TimeDataPtr(unixPtr(rec.Created)),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlReview)
	}
	return res, nil
}
