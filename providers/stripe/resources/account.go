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

type accountRecord struct {
	ID               string            `json:"id"`
	Type             string            `json:"type"`
	BusinessType     string            `json:"business_type"`
	Country          string            `json:"country"`
	DefaultCurrency  string            `json:"default_currency"`
	Email            string            `json:"email"`
	ChargesEnabled   bool              `json:"charges_enabled"`
	PayoutsEnabled   bool              `json:"payouts_enabled"`
	DetailsSubmitted bool              `json:"details_submitted"`
	Created          int64             `json:"created"`
	Capabilities     map[string]string `json:"capabilities"`
	BusinessProfile  *struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"business_profile"`
	Requirements *struct {
		DisabledReason string   `json:"disabled_reason"`
		CurrentlyDue   []string `json:"currently_due"`
		PastDue        []string `json:"past_due"`
		EventuallyDue  []string `json:"eventually_due"`
	} `json:"requirements"`
	Settings *struct {
		CardPayments *struct {
			DeclineOn *struct {
				AvsFailure bool `json:"avs_failure"`
				CvcFailure bool `json:"cvc_failure"`
			} `json:"decline_on"`
		} `json:"card_payments"`
		Dashboard *struct {
			Timezone string `json:"timezone"`
		} `json:"dashboard"`
		Payments *struct {
			StatementDescriptor string `json:"statement_descriptor"`
		} `json:"payments"`
		Payouts *struct {
			DebitNegativeBalances bool `json:"debit_negative_balances"`
			Schedule              *struct {
				Interval  string `json:"interval"`
				DelayDays int64  `json:"delay_days"`
			} `json:"schedule"`
		} `json:"payouts"`
	} `json:"settings"`
	TosAcceptance *struct {
		Date int64  `json:"date"`
		IP   string `json:"ip"`
	} `json:"tos_acceptance"`
}

func (c *mqlStripeAccount) id() (string, error) {
	return c.Id.Data, c.Id.Error
}

// fetchAccount retrieves the account associated with the connection's key.
func fetchAccount(runtime *plugin.Runtime) (*mqlStripeAccount, error) {
	c := stripeConn(runtime)

	var rec accountRecord
	if err := c.Get(context.Background(), "/v1/account", nil, &rec); err != nil {
		return nil, err
	}

	businessName := ""
	businessURL := ""
	if rec.BusinessProfile != nil {
		businessName = rec.BusinessProfile.Name
		businessURL = rec.BusinessProfile.URL
	}

	var (
		disabledReason string
		currentlyDue   []string
		pastDue        []string
		eventuallyDue  []string
	)
	if rec.Requirements != nil {
		disabledReason = rec.Requirements.DisabledReason
		currentlyDue = rec.Requirements.CurrentlyDue
		pastDue = rec.Requirements.PastDue
		eventuallyDue = rec.Requirements.EventuallyDue
	}

	var (
		avsDecline          bool
		cvcDecline          bool
		statementDescriptor string
		timezone            string
		payoutInterval      string
		payoutDelayDays     int64
		debitNegative       bool
	)
	if s := rec.Settings; s != nil {
		if s.CardPayments != nil && s.CardPayments.DeclineOn != nil {
			avsDecline = s.CardPayments.DeclineOn.AvsFailure
			cvcDecline = s.CardPayments.DeclineOn.CvcFailure
		}
		if s.Payments != nil {
			statementDescriptor = s.Payments.StatementDescriptor
		}
		if s.Dashboard != nil {
			timezone = s.Dashboard.Timezone
		}
		if s.Payouts != nil {
			debitNegative = s.Payouts.DebitNegativeBalances
			if s.Payouts.Schedule != nil {
				payoutInterval = s.Payouts.Schedule.Interval
				payoutDelayDays = s.Payouts.Schedule.DelayDays
			}
		}
	}

	var tosDate *int64
	tosIP := ""
	if rec.TosAcceptance != nil {
		d := rec.TosAcceptance.Date
		tosDate = &d
		tosIP = rec.TosAcceptance.IP
	}

	res, err := CreateResource(runtime, "stripe.account", map[string]*llx.RawData{
		"id":                         llx.StringData(rec.ID),
		"type":                       llx.StringData(rec.Type),
		"businessType":               llx.StringData(rec.BusinessType),
		"country":                    llx.StringData(rec.Country),
		"defaultCurrency":            llx.StringData(rec.DefaultCurrency),
		"email":                      llx.StringData(rec.Email),
		"chargesEnabled":             llx.BoolData(rec.ChargesEnabled),
		"payoutsEnabled":             llx.BoolData(rec.PayoutsEnabled),
		"detailsSubmitted":           llx.BoolData(rec.DetailsSubmitted),
		"businessName":               llx.StringData(businessName),
		"businessUrl":                llx.StringData(businessURL),
		"capabilities":               llx.MapData(mapStrToAny(rec.Capabilities), types.String),
		"requirementsDisabledReason": llx.StringData(disabledReason),
		"requirementsCurrentlyDue":   llx.ArrayData(strSliceToAny(currentlyDue), types.String),
		"requirementsPastDue":        llx.ArrayData(strSliceToAny(pastDue), types.String),
		"requirementsEventuallyDue":  llx.ArrayData(strSliceToAny(eventuallyDue), types.String),
		"declineChargeOnAvsFailure":  llx.BoolData(avsDecline),
		"declineChargeOnCvcFailure":  llx.BoolData(cvcDecline),
		"statementDescriptor":        llx.StringData(statementDescriptor),
		"payoutInterval":             llx.StringData(payoutInterval),
		"payoutDelayDays":            llx.IntData(payoutDelayDays),
		"debitNegativeBalances":      llx.BoolData(debitNegative),
		"timezone":                   llx.StringData(timezone),
		"tosAcceptanceDate":          llx.TimeDataPtr(unixPtrFromPtr(tosDate)),
		"tosAcceptanceIp":            llx.StringData(tosIP),
		"created":                    llx.TimeDataPtr(unixPtr(rec.Created)),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlStripeAccount), nil
}

// externalAccounts lists the bank accounts and debit cards a Stripe account
// pays out to. The endpoint is supported for Custom and Express connected
// accounts; for a standalone Dashboard-managed account it may be unsupported,
// in which case the list degrades to empty rather than failing the scan.
func (a *mqlStripeAccount) externalAccounts() ([]any, error) {
	accountID := a.Id.Data
	if accountID == "" {
		return []any{}, nil
	}

	c := stripeConn(a.MqlRuntime)
	records, err := connection.List[externalAccountRecord](
		context.Background(), c, "/v1/accounts/"+accountID+"/external_accounts", nil)
	if err != nil {
		// Standalone accounts return a client error for this endpoint; treat
		// that as "nothing listable" rather than failing the whole query.
		if connection.IsClientError(err) {
			return []any{}, nil
		}
		return nil, err
	}

	res := make([]any, 0, len(records))
	for i := range records {
		rec := records[i]
		mqlExt, err := CreateResource(a.MqlRuntime, "stripe.account.externalAccount", map[string]*llx.RawData{
			"id":                 llx.StringData(rec.ID),
			"type":               llx.StringData(rec.Object),
			"last4":              llx.StringData(rec.Last4),
			"country":            llx.StringData(rec.Country),
			"currency":           llx.StringData(rec.Currency),
			"bankName":           llx.StringData(rec.BankName),
			"routingNumber":      llx.StringData(rec.RoutingNumber),
			"status":             llx.StringData(rec.Status),
			"brand":              llx.StringData(rec.Brand),
			"defaultForCurrency": llx.BoolData(rec.DefaultForCurrency),
			"fingerprint":        llx.StringData(rec.Fingerprint),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlExt)
	}
	return res, nil
}

func (c *mqlStripeAccountExternalAccount) id() (string, error) {
	return c.Id.Data, c.Id.Error
}

// externalAccountRecord is the union of the bank_account and card shapes
// returned by the external accounts endpoint. Fields not present for a given
// object type stay at their zero value.
type externalAccountRecord struct {
	ID                 string `json:"id"`
	Object             string `json:"object"`
	Last4              string `json:"last4"`
	Country            string `json:"country"`
	Currency           string `json:"currency"`
	BankName           string `json:"bank_name"`
	RoutingNumber      string `json:"routing_number"`
	Status             string `json:"status"`
	Brand              string `json:"brand"`
	DefaultForCurrency bool   `json:"default_for_currency"`
	Fingerprint        string `json:"fingerprint"`
}

func (r externalAccountRecord) GetID() string { return r.ID }
