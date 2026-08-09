// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/stripe/connection"
)

type disputeRecord struct {
	ID                 string `json:"id"`
	Amount             int64  `json:"amount"`
	Currency           string `json:"currency"`
	Status             string `json:"status"`
	Reason             string `json:"reason"`
	IsChargeRefundable bool   `json:"is_charge_refundable"`
	Charge             string `json:"charge"`
	PaymentIntent      string `json:"payment_intent"`
	Livemode           bool   `json:"livemode"`
	Created            int64  `json:"created"`
	EvidenceDetails    *struct {
		DueBy int64 `json:"due_by"`
	} `json:"evidence_details"`
}

func (r disputeRecord) GetID() string { return r.ID }

func (c *mqlStripeDispute) id() (string, error) {
	return c.Id.Data, c.Id.Error
}

func listDisputes(runtime *plugin.Runtime) ([]any, error) {
	c := stripeConn(runtime)
	records, err := connection.List[disputeRecord](context.Background(), c, "/v1/disputes", nil)
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(records))
	for i := range records {
		rec := records[i]
		var dueBy int64
		if rec.EvidenceDetails != nil {
			dueBy = rec.EvidenceDetails.DueBy
		}
		mqlDispute, err := CreateResource(runtime, "stripe.dispute", map[string]*llx.RawData{
			"id":                 llx.StringData(rec.ID),
			"amount":             llx.IntData(rec.Amount),
			"currency":           llx.StringData(rec.Currency),
			"status":             llx.StringData(rec.Status),
			"reason":             llx.StringData(rec.Reason),
			"isChargeRefundable": llx.BoolData(rec.IsChargeRefundable),
			"charge":             llx.StringData(rec.Charge),
			"paymentIntent":      llx.StringData(rec.PaymentIntent),
			"evidenceDueBy":      llx.TimeDataPtr(unixPtr(dueBy)),
			"livemode":           llx.BoolData(rec.Livemode),
			"created":            llx.TimeDataPtr(unixPtr(rec.Created)),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlDispute)
	}
	return res, nil
}
