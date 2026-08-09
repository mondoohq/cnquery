// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"time"

	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/stripe/connection"
)

func (r *mqlStripe) id() (string, error) {
	return "stripe", nil
}

// --- shared helpers -------------------------------------------------------

// stripeConn resolves the Stripe connection from a resource runtime.
func stripeConn(runtime *plugin.Runtime) *connection.StripeConnection {
	return runtime.Connection.(*connection.StripeConnection)
}

// unixPtr converts a Stripe unix-second timestamp into a *time.Time, returning
// nil when the timestamp is absent (zero).
func unixPtr(sec int64) *time.Time {
	if sec == 0 {
		return nil
	}
	t := time.Unix(sec, 0).UTC()
	return &t
}

// strSliceToAny widens a string slice into an any slice for llx.ArrayData.
func strSliceToAny(in []string) []any {
	out := make([]any, len(in))
	for i := range in {
		out[i] = in[i]
	}
	return out
}

// mapStrToAny widens a string map into a map[string]any for a map field.
func mapStrToAny(in map[string]string) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// --- root accessors -------------------------------------------------------

func (r *mqlStripe) account() (*mqlStripeAccount, error) {
	return fetchAccount(r.MqlRuntime)
}

func (r *mqlStripe) balance() (*mqlStripeBalance, error) {
	return fetchBalance(r.MqlRuntime)
}

func (r *mqlStripe) webhookEndpoints() ([]any, error) {
	return listWebhookEndpoints(r.MqlRuntime)
}

func (r *mqlStripe) customers() ([]any, error) {
	c := stripeConn(r.MqlRuntime)
	records, err := connection.List[customerRecord](context.Background(), c, "/v1/customers", nil)
	if err != nil {
		return nil, err
	}
	res := make([]any, 0, len(records))
	for i := range records {
		mqlCustomer, err := newMqlStripeCustomer(r.MqlRuntime, &records[i])
		if err != nil {
			return nil, err
		}
		res = append(res, mqlCustomer)
	}
	return res, nil
}

func (r *mqlStripe) products() ([]any, error) {
	c := stripeConn(r.MqlRuntime)
	records, err := connection.List[productRecord](context.Background(), c, "/v1/products", nil)
	if err != nil {
		return nil, err
	}
	res := make([]any, 0, len(records))
	for i := range records {
		mqlProduct, err := newMqlStripeProduct(r.MqlRuntime, &records[i])
		if err != nil {
			return nil, err
		}
		res = append(res, mqlProduct)
	}
	return res, nil
}

func (r *mqlStripe) prices() ([]any, error) {
	return listPrices(r.MqlRuntime)
}

func (r *mqlStripe) subscriptions() ([]any, error) {
	return listSubscriptions(r.MqlRuntime)
}
