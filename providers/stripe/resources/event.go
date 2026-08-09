// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/stripe/connection"
)

type eventRecord struct {
	ID              string `json:"id"`
	Type            string `json:"type"`
	APIVersion      string `json:"api_version"`
	PendingWebhooks int64  `json:"pending_webhooks"`
	Livemode        bool   `json:"livemode"`
	Created         int64  `json:"created"`
	Request         *struct {
		ID             string `json:"id"`
		IdempotencyKey string `json:"idempotency_key"`
	} `json:"request"`
}

func (r eventRecord) GetID() string { return r.ID }

func (c *mqlStripeEvent) id() (string, error) {
	return c.Id.Data, c.Id.Error
}

func listEvents(runtime *plugin.Runtime) ([]any, error) {
	c := stripeConn(runtime)
	records, err := connection.List[eventRecord](context.Background(), c, "/v1/events", nil)
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(records))
	for i := range records {
		rec := records[i]
		requestID := ""
		idempotencyKey := ""
		if rec.Request != nil {
			requestID = rec.Request.ID
			idempotencyKey = rec.Request.IdempotencyKey
		}
		mqlEvent, err := CreateResource(runtime, "stripe.event", map[string]*llx.RawData{
			"id":                    llx.StringData(rec.ID),
			"type":                  llx.StringData(rec.Type),
			"apiVersion":            llx.StringData(rec.APIVersion),
			"pendingWebhooks":       llx.IntData(rec.PendingWebhooks),
			"requestId":             llx.StringData(requestID),
			"requestIdempotencyKey": llx.StringData(idempotencyKey),
			"livemode":              llx.BoolData(rec.Livemode),
			"created":               llx.TimeDataPtr(unixPtr(rec.Created)),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlEvent)
	}
	return res, nil
}
