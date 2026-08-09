// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/types"
)

type balanceRecord struct {
	Available []balanceAmount `json:"available"`
	Pending   []balanceAmount `json:"pending"`
	Livemode  bool            `json:"livemode"`
}

type balanceAmount struct {
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
}

func balanceAmountsToAny(in []balanceAmount) []any {
	out := make([]any, len(in))
	for i := range in {
		out[i] = map[string]any{
			"amount":   in[i].Amount,
			"currency": in[i].Currency,
		}
	}
	return out
}

// fetchBalance retrieves the account's current balance. Balance is a singleton
// (one per account), so its cache key is a fixed synthetic id rather than a
// user-facing value.
func fetchBalance(runtime *plugin.Runtime) (*mqlStripeBalance, error) {
	c := stripeConn(runtime)

	var rec balanceRecord
	if err := c.Get(context.Background(), "/v1/balance", nil, &rec); err != nil {
		return nil, err
	}

	res, err := CreateResource(runtime, "stripe.balance", map[string]*llx.RawData{
		"__id":      llx.StringData("stripe.balance"),
		"available": llx.ArrayData(balanceAmountsToAny(rec.Available), types.Dict),
		"pending":   llx.ArrayData(balanceAmountsToAny(rec.Pending), types.Dict),
		"livemode":  llx.BoolData(rec.Livemode),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlStripeBalance), nil
}
