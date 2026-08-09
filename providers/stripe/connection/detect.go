// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"context"
)

// DetectedAccount is the minimal slice of the Stripe account object used to
// stamp the asset's platform ID and display name during connect.
type DetectedAccount struct {
	ID              string `json:"id"`
	Email           string `json:"email"`
	BusinessProfile *struct {
		Name string `json:"name"`
	} `json:"business_profile"`
	Settings *struct {
		Dashboard *struct {
			DisplayName string `json:"display_name"`
		} `json:"dashboard"`
	} `json:"settings"`
}

// DisplayName picks the most human-friendly label available for the account.
func (a *DetectedAccount) DisplayName() string {
	if a.Settings != nil && a.Settings.Dashboard != nil && a.Settings.Dashboard.DisplayName != "" {
		return a.Settings.Dashboard.DisplayName
	}
	if a.BusinessProfile != nil && a.BusinessProfile.Name != "" {
		return a.BusinessProfile.Name
	}
	return a.Email
}

// DetectAccount fetches the account associated with the connection's secret
// key. It doubles as a credential check during connect.
func (c *StripeConnection) DetectAccount(ctx context.Context) (*DetectedAccount, error) {
	var account DetectedAccount
	if err := c.Get(ctx, "/v1/account", nil, &account); err != nil {
		return nil, err
	}
	return &account, nil
}
