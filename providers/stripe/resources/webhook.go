// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/stripe/connection"
	"go.mondoo.com/mql/v13/types"
)

type webhookEndpointRecord struct {
	ID            string            `json:"id"`
	URL           string            `json:"url"`
	Status        string            `json:"status"`
	EnabledEvents []string          `json:"enabled_events"`
	APIVersion    string            `json:"api_version"`
	Description   string            `json:"description"`
	Livemode      bool              `json:"livemode"`
	Metadata      map[string]string `json:"metadata"`
	Created       int64             `json:"created"`
}

func (r webhookEndpointRecord) GetID() string { return r.ID }

func (c *mqlStripeWebhookEndpoint) id() (string, error) {
	return c.Id.Data, c.Id.Error
}

func listWebhookEndpoints(runtime *plugin.Runtime) ([]any, error) {
	c := stripeConn(runtime)
	records, err := connection.List[webhookEndpointRecord](context.Background(), c, "/v1/webhook_endpoints", nil)
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(records))
	for i := range records {
		rec := records[i]
		mqlEndpoint, err := CreateResource(runtime, "stripe.webhookEndpoint", map[string]*llx.RawData{
			"id":            llx.StringData(rec.ID),
			"url":           llx.StringData(rec.URL),
			"status":        llx.StringData(rec.Status),
			"enabledEvents": llx.ArrayData(strSliceToAny(rec.EnabledEvents), types.String),
			"apiVersion":    llx.StringData(rec.APIVersion),
			"description":   llx.StringData(rec.Description),
			"livemode":      llx.BoolData(rec.Livemode),
			"metadata":      llx.MapData(mapStrToAny(rec.Metadata), types.String),
			"created":       llx.TimeDataPtr(unixPtr(rec.Created)),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlEndpoint)
	}
	return res, nil
}
