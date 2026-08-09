// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/vault"
	"go.mondoo.com/mql/v13/providers/stripe/connection"
)

// newTestRuntime builds a plugin runtime whose Stripe connection points at the
// given base URL (a test server), so resource functions exercise the real
// JSON->MQL mapping without live credentials.
func newTestRuntime(t *testing.T, baseURL string) *plugin.Runtime {
	t.Helper()
	conf := &inventory.Config{
		Type:    "stripe",
		Options: map[string]string{connection.OptionBaseURL: baseURL},
		Credentials: []*vault.Credential{
			vault.NewPasswordCredential("", "sk_test_dummy"),
		},
	}
	asset := &inventory.Asset{Connections: []*inventory.Config{conf}}
	conn, err := connection.NewStripeConnection(1, asset, conf)
	require.NoError(t, err)
	return plugin.NewRuntime(conn, nil, false, CreateResource, NewResource, GetData, SetData, nil)
}

// stripeMux serves canned fixtures for every endpoint the provider reads.
func stripeMux(t *testing.T) http.Handler {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("/v1/account", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, `{
			"id": "acct_test123",
			"object": "account",
			"type": "standard",
			"business_type": "company",
			"country": "US",
			"default_currency": "usd",
			"email": "owner@example.com",
			"charges_enabled": true,
			"payouts_enabled": false,
			"details_submitted": true,
			"created": 1700000000,
			"capabilities": {"card_payments": "active", "transfers": "pending"},
			"business_profile": {"name": "Acme Inc", "url": "https://acme.example.com"},
			"requirements": {
				"disabled_reason": "requirements.past_due",
				"currently_due": ["external_account"],
				"past_due": ["tos_acceptance.date"],
				"eventually_due": ["individual.verification.document"]
			},
			"settings": {
				"card_payments": {"decline_on": {"avs_failure": true, "cvc_failure": false}},
				"dashboard": {"display_name": "Acme", "timezone": "America/New_York"},
				"payments": {"statement_descriptor": "ACME"},
				"payouts": {"debit_negative_balances": true, "schedule": {"interval": "weekly", "delay_days": 7}}
			},
			"tos_acceptance": {"date": 1699990000, "ip": "203.0.113.7"}
		}`)
	})

	mux.HandleFunc("/v1/accounts/acct_test123/external_accounts", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, `{"object":"list","has_more":false,"data":[
			{"id":"ba_1","object":"bank_account","last4":"6789","country":"US","currency":"usd","bank_name":"STRIPE TEST BANK","routing_number":"110000000","status":"verified","default_for_currency":true,"fingerprint":"fp_bank"},
			{"id":"card_1","object":"card","last4":"4242","country":"US","currency":"usd","brand":"Visa","default_for_currency":false,"fingerprint":"fp_card"}
		]}`)
	})

	mux.HandleFunc("/v1/webhook_endpoints", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, `{"object":"list","has_more":false,"data":[
			{"id":"we_1","url":"https://hooks.example.com/stripe","status":"enabled","enabled_events":["*"],"api_version":"2024-06-20","description":"prod","livemode":true,"metadata":{"team":"payments"},"created":1700000100}
		]}`)
	})

	// customers paginate across two pages to exercise the starting_after cursor.
	mux.HandleFunc("/v1/customers", func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("starting_after") {
		case "":
			writeJSON(w, `{"object":"list","has_more":true,"data":[
				{"id":"cus_1","email":"a@example.com","name":"A","currency":"usd","balance":-500,"delinquent":false,"livemode":true,"created":1700000200},
				{"id":"cus_2","email":"b@example.com","name":"B","currency":"eur","balance":0,"delinquent":true,"livemode":true,"created":1700000300}
			]}`)
		case "cus_2":
			writeJSON(w, `{"object":"list","has_more":false,"data":[
				{"id":"cus_3","email":"c@example.com","name":"C","currency":"usd","balance":100,"delinquent":false,"livemode":false,"created":1700000400}
			]}`)
		default:
			t.Errorf("unexpected customers cursor %q", r.URL.Query().Get("starting_after"))
		}
	})

	mux.HandleFunc("/v1/customers/cus_typed", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, `{"id":"cus_typed","email":"typed@example.com","name":"Typed","currency":"usd","balance":0,"delinquent":false,"livemode":true,"created":1700000500}`)
	})

	mux.HandleFunc("/v1/products", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, `{"object":"list","has_more":false,"data":[
			{"id":"prod_1","name":"Widget","active":true,"description":"a widget","type":"good","url":"https://acme.example.com/widget","livemode":true,"created":1700000600,"updated":1700000700}
		]}`)
	})

	mux.HandleFunc("/v1/products/prod_typed", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, `{"id":"prod_typed","name":"Typed Product","active":true,"type":"service","livemode":true,"created":1700000800}`)
	})

	mux.HandleFunc("/v1/prices", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, `{"object":"list","has_more":false,"data":[
			{"id":"price_1","active":true,"currency":"usd","unit_amount":1999,"type":"recurring","nickname":"monthly","livemode":true,"created":1700000900,"product":"prod_typed","recurring":{"interval":"month","interval_count":1}},
			{"id":"price_2","active":true,"currency":"usd","unit_amount":5000,"type":"one_time","livemode":true,"created":1700001000,"product":"prod_typed"}
		]}`)
	})

	mux.HandleFunc("/v1/subscriptions", func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("status"); got != "all" {
			t.Errorf("expected subscriptions status=all, got %q", got)
		}
		writeJSON(w, `{"object":"list","has_more":false,"data":[
			{"id":"sub_1","status":"active","currency":"usd","cancel_at_period_end":true,"livemode":true,"created":1700001100,"current_period_start":1700001100,"current_period_end":1702593100,"canceled_at":0,"customer":"cus_typed"}
		]}`)
	})

	mux.HandleFunc("/v1/balance", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, `{"object":"balance","livemode":true,
			"available":[{"amount":12345,"currency":"usd"}],
			"pending":[{"amount":-200,"currency":"usd"}]}`)
	})

	mux.HandleFunc("/v1/disputes", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, `{"object":"list","has_more":false,"data":[
			{"id":"dp_1","amount":2500,"currency":"usd","status":"needs_response","reason":"fraudulent","is_charge_refundable":false,"charge":"ch_1","payment_intent":"pi_1","livemode":true,"created":1700001200,"evidence_details":{"due_by":1700601200}},
			{"id":"dp_2","amount":1000,"currency":"usd","status":"won","reason":"duplicate","is_charge_refundable":true,"charge":"ch_2","payment_intent":"pi_2","livemode":true,"created":1700001300},
			{"id":"dp_3","amount":750,"currency":"usd","status":"lost","reason":"general","is_charge_refundable":false,"charge":"ch_3","payment_intent":"pi_3","livemode":true,"created":1700001400,"evidence_details":{}}
		]}`)
	})

	mux.HandleFunc("/v1/reviews", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, `{"object":"list","has_more":false,"data":[
			{"id":"prv_1","open":true,"reason":"rule","opened_reason":"rule","closed_reason":"","ip_address":"198.51.100.9","charge":"ch_1","payment_intent":"pi_1","livemode":true,"created":1700001300}
		]}`)
	})

	mux.HandleFunc("/v1/events", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, `{"object":"list","has_more":false,"data":[
			{"id":"evt_1","type":"charge.succeeded","api_version":"2024-06-20","pending_webhooks":2,"livemode":true,"created":1700001400,"request":{"id":"req_1","idempotency_key":"key_1"}}
		]}`)
	})

	return mux
}

func writeJSON(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(body))
}

func newStripeServer(t *testing.T) (*plugin.Runtime, func()) {
	t.Helper()
	srv := httptest.NewServer(stripeMux(t))
	return newTestRuntime(t, srv.URL), srv.Close
}

func TestAccountMapping(t *testing.T) {
	rt, done := newStripeServer(t)
	defer done()

	acct, err := fetchAccount(rt)
	require.NoError(t, err)

	assert.Equal(t, "acct_test123", acct.Id.Data)
	assert.Equal(t, "standard", acct.Type.Data)
	assert.Equal(t, "company", acct.BusinessType.Data)
	assert.Equal(t, "US", acct.Country.Data)
	assert.Equal(t, "usd", acct.DefaultCurrency.Data)
	assert.Equal(t, "owner@example.com", acct.Email.Data)
	assert.True(t, acct.ChargesEnabled.Data)
	assert.False(t, acct.PayoutsEnabled.Data)
	assert.True(t, acct.DetailsSubmitted.Data)
	assert.Equal(t, "Acme Inc", acct.BusinessName.Data)
	assert.Equal(t, "https://acme.example.com", acct.BusinessUrl.Data)

	assert.Equal(t, map[string]any{"card_payments": "active", "transfers": "pending"}, acct.Capabilities.Data)

	// requirements
	assert.Equal(t, "requirements.past_due", acct.RequirementsDisabledReason.Data)
	assert.Equal(t, []any{"external_account"}, acct.RequirementsCurrentlyDue.Data)
	assert.Equal(t, []any{"tos_acceptance.date"}, acct.RequirementsPastDue.Data)
	assert.Equal(t, []any{"individual.verification.document"}, acct.RequirementsEventuallyDue.Data)

	// fraud + payout posture
	assert.True(t, acct.DeclineChargeOnAvsFailure.Data)
	assert.False(t, acct.DeclineChargeOnCvcFailure.Data)
	assert.Equal(t, "ACME", acct.StatementDescriptor.Data)
	assert.Equal(t, "America/New_York", acct.Timezone.Data)
	assert.Equal(t, "weekly", acct.PayoutInterval.Data)
	assert.Equal(t, int64(7), acct.PayoutDelayDays.Data)
	assert.True(t, acct.DebitNegativeBalances.Data)

	// tos + created timestamps
	require.NotNil(t, acct.TosAcceptanceDate.Data)
	assert.Equal(t, int64(1699990000), acct.TosAcceptanceDate.Data.Unix())
	assert.Equal(t, "203.0.113.7", acct.TosAcceptanceIp.Data)
	require.NotNil(t, acct.Created.Data)
	assert.Equal(t, int64(1700000000), acct.Created.Data.Unix())
}

func TestExternalAccountsMapping(t *testing.T) {
	rt, done := newStripeServer(t)
	defer done()

	acct, err := fetchAccount(rt)
	require.NoError(t, err)

	ext, err := acct.externalAccounts()
	require.NoError(t, err)
	require.Len(t, ext, 2)

	bank := ext[0].(*mqlStripeAccountExternalAccount)
	assert.Equal(t, "bank_account", bank.Type.Data)
	assert.Equal(t, "STRIPE TEST BANK", bank.BankName.Data)
	assert.Equal(t, "6789", bank.Last4.Data)
	assert.Equal(t, "110000000", bank.RoutingNumber.Data)
	assert.Equal(t, "verified", bank.Status.Data)
	assert.True(t, bank.DefaultForCurrency.Data)

	card := ext[1].(*mqlStripeAccountExternalAccount)
	assert.Equal(t, "card", card.Type.Data)
	assert.Equal(t, "Visa", card.Brand.Data)
	assert.Equal(t, "4242", card.Last4.Data)
}

// TestExternalAccountsDegradesOnClientError verifies a 4xx (e.g. the endpoint
// being unsupported for a standalone account) yields an empty list, not an error.
func TestExternalAccountsDegradesOnClientError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/account", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, `{"id":"acct_standalone","object":"account"}`)
	})
	mux.HandleFunc("/v1/accounts/acct_standalone/external_accounts", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, `{"error":{"type":"invalid_request_error","message":"external accounts can only be listed for connected accounts"}}`)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	rt := newTestRuntime(t, srv.URL)
	acct, err := fetchAccount(rt)
	require.NoError(t, err)

	ext, err := acct.externalAccounts()
	require.NoError(t, err)
	assert.Empty(t, ext)
}

// TestAccountRequirementsNullWhenUnreported verifies an account Stripe reports
// without a requirements object reads as null rather than as empty lists. An
// empty list would tell a policy that nothing is outstanding, which is a
// different claim from Stripe not having reported requirements at all.
func TestAccountRequirementsNullWhenUnreported(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/account", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, `{"id":"acct_norequirements","object":"account"}`)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	acct, err := fetchAccount(newTestRuntime(t, srv.URL))
	require.NoError(t, err)

	for name, field := range map[string]plugin.TValue[[]any]{
		"currentlyDue":  acct.RequirementsCurrentlyDue,
		"pastDue":       acct.RequirementsPastDue,
		"eventuallyDue": acct.RequirementsEventuallyDue,
	} {
		assert.Nil(t, field.Data, name)
		assert.Equal(t, plugin.StateIsNull|plugin.StateIsSet, field.State, name)
	}
}

// TestAccountRequirementsEmptyWhenReportedEmpty verifies the other half of the
// distinction: requirements reported as empty stay an empty list.
func TestAccountRequirementsEmptyWhenReportedEmpty(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/account", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, `{"id":"acct_clean","object":"account","requirements":{"disabled_reason":"","currently_due":[],"past_due":[],"eventually_due":["tos_acceptance.date"]}}`)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	acct, err := fetchAccount(newTestRuntime(t, srv.URL))
	require.NoError(t, err)

	require.NotNil(t, acct.RequirementsCurrentlyDue.Data)
	assert.Empty(t, acct.RequirementsCurrentlyDue.Data)
	assert.Equal(t, []any{"tos_acceptance.date"}, acct.RequirementsEventuallyDue.Data)
}

func TestCustomersPagination(t *testing.T) {
	rt, done := newStripeServer(t)
	defer done()

	s := &mqlStripe{MqlRuntime: rt}
	customers, err := s.customers()
	require.NoError(t, err)
	require.Len(t, customers, 3)

	c0 := customers[0].(*mqlStripeCustomer)
	assert.Equal(t, "cus_1", c0.Id.Data)
	assert.Equal(t, "a@example.com", c0.Email.Data)
	assert.Equal(t, int64(-500), c0.Balance.Data)
	assert.False(t, c0.Delinquent.Data)

	c1 := customers[1].(*mqlStripeCustomer)
	assert.True(t, c1.Delinquent.Data)

	c2 := customers[2].(*mqlStripeCustomer)
	assert.Equal(t, "cus_3", c2.Id.Data)
	assert.False(t, c2.Livemode.Data)
}

func TestPricesAndProductTypedRef(t *testing.T) {
	rt, done := newStripeServer(t)
	defer done()

	s := &mqlStripe{MqlRuntime: rt}
	prices, err := s.prices()
	require.NoError(t, err)
	require.Len(t, prices, 2)

	recurring := prices[0].(*mqlStripePrice)
	assert.Equal(t, "price_1", recurring.Id.Data)
	assert.Equal(t, int64(1999), recurring.UnitAmount.Data)
	assert.Equal(t, "recurring", recurring.Type.Data)
	assert.Equal(t, "month", recurring.RecurringInterval.Data)
	assert.Equal(t, int64(1), recurring.RecurringIntervalCount.Data)
	assert.Equal(t, "prod_typed", recurring.cacheProductID)

	oneTime := prices[1].(*mqlStripePrice)
	assert.Equal(t, "one_time", oneTime.Type.Data)
	assert.Empty(t, oneTime.RecurringInterval.Data)

	// typed reference resolves the product via NewResource + init
	prod, err := recurring.product()
	require.NoError(t, err)
	require.NotNil(t, prod)
	assert.Equal(t, "prod_typed", prod.Id.Data)
	assert.Equal(t, "Typed Product", prod.Name.Data)
	assert.Equal(t, "service", prod.Type.Data)
}

func TestSubscriptionsAndCustomerTypedRef(t *testing.T) {
	rt, done := newStripeServer(t)
	defer done()

	s := &mqlStripe{MqlRuntime: rt}
	subs, err := s.subscriptions()
	require.NoError(t, err)
	require.Len(t, subs, 1)

	sub := subs[0].(*mqlStripeSubscription)
	assert.Equal(t, "sub_1", sub.Id.Data)
	assert.Equal(t, "active", sub.Status.Data)
	assert.True(t, sub.CancelAtPeriodEnd.Data)
	require.NotNil(t, sub.CurrentPeriodEnd.Data)
	assert.Equal(t, int64(1702593100), sub.CurrentPeriodEnd.Data.Unix())
	assert.Nil(t, sub.CanceledAt.Data, "canceled_at of 0 must map to nil, not epoch zero")
	assert.Equal(t, "cus_typed", sub.cacheCustomerID)

	cust, err := sub.customer()
	require.NoError(t, err)
	require.NotNil(t, cust)
	assert.Equal(t, "cus_typed", cust.Id.Data)
	assert.Equal(t, "typed@example.com", cust.Email.Data)
}

// TestSubscriptionCustomerNullWhenEmpty verifies the typed accessor marks the
// field null (rather than panicking or re-fetching) when there is no customer id.
func TestSubscriptionCustomerNullWhenEmpty(t *testing.T) {
	rt, done := newStripeServer(t)
	defer done()

	sub := &mqlStripeSubscription{MqlRuntime: rt}
	sub.cacheCustomerID = ""

	cust, err := sub.customer()
	require.NoError(t, err)
	assert.Nil(t, cust)
	assert.True(t, sub.Customer.State&plugin.StateIsNull != 0, "expected StateIsNull to be set")
	assert.True(t, sub.Customer.State&plugin.StateIsSet != 0, "expected StateIsSet to be set")
}

func TestWebhookEndpointsMapping(t *testing.T) {
	rt, done := newStripeServer(t)
	defer done()

	endpoints, err := listWebhookEndpoints(rt)
	require.NoError(t, err)
	require.Len(t, endpoints, 1)

	we := endpoints[0].(*mqlStripeWebhookEndpoint)
	assert.Equal(t, "we_1", we.Id.Data)
	assert.Equal(t, "https://hooks.example.com/stripe", we.Url.Data)
	assert.Equal(t, "enabled", we.Status.Data)
	assert.Equal(t, []any{"*"}, we.EnabledEvents.Data)
	assert.Equal(t, map[string]any{"team": "payments"}, we.Metadata.Data)
}

func TestBalanceMapping(t *testing.T) {
	rt, done := newStripeServer(t)
	defer done()

	bal, err := fetchBalance(rt)
	require.NoError(t, err)

	assert.True(t, bal.Livemode.Data)
	require.Len(t, bal.Available.Data, 1)
	assert.Equal(t, map[string]any{"amount": int64(12345), "currency": "usd"}, bal.Available.Data[0])
	require.Len(t, bal.Pending.Data, 1)
	// negative pending amount must be preserved
	assert.Equal(t, map[string]any{"amount": int64(-200), "currency": "usd"}, bal.Pending.Data[0])
	assert.Equal(t, "stripe.balance", bal.MqlID())
}

func TestDisputesMapping(t *testing.T) {
	rt, done := newStripeServer(t)
	defer done()

	disputes, err := listDisputes(rt)
	require.NoError(t, err)
	require.Len(t, disputes, 3)

	d := disputes[0].(*mqlStripeDispute)
	assert.Equal(t, "dp_1", d.Id.Data)
	assert.Equal(t, int64(2500), d.Amount.Data)
	assert.Equal(t, "needs_response", d.Status.Data)
	assert.Equal(t, "fraudulent", d.Reason.Data)
	assert.False(t, d.IsChargeRefundable.Data)
	assert.Equal(t, "ch_1", d.Charge.Data)
	require.NotNil(t, d.EvidenceDueBy.Data)
	assert.Equal(t, int64(1700601200), d.EvidenceDueBy.Data.Unix())
}

// A dispute the API reports without an evidence deadline must read as null,
// not as the Unix epoch. Stripe omits evidence_details entirely on some
// disputes and returns it without a due_by on others, so cover both.
func TestDisputeEvidenceDueByNullWhenAbsent(t *testing.T) {
	rt, done := newStripeServer(t)
	defer done()

	disputes, err := listDisputes(rt)
	require.NoError(t, err)
	require.Len(t, disputes, 3)

	for _, tc := range []struct {
		idx  int
		id   string
		desc string
	}{
		{1, "dp_2", "evidence_details omitted"},
		{2, "dp_3", "evidence_details present without due_by"},
	} {
		d := disputes[tc.idx].(*mqlStripeDispute)
		require.Equal(t, tc.id, d.Id.Data, tc.desc)
		assert.Nil(t, d.EvidenceDueBy.Data, tc.desc)
		assert.Equal(t, plugin.StateIsNull|plugin.StateIsSet, d.EvidenceDueBy.State, tc.desc)
	}
}

func TestReviewsMapping(t *testing.T) {
	rt, done := newStripeServer(t)
	defer done()

	reviews, err := listReviews(rt)
	require.NoError(t, err)
	require.Len(t, reviews, 1)

	rv := reviews[0].(*mqlStripeReview)
	assert.Equal(t, "prv_1", rv.Id.Data)
	assert.True(t, rv.Open.Data)
	assert.Equal(t, "rule", rv.Reason.Data)
	assert.Equal(t, "198.51.100.9", rv.IpAddress.Data)
}

func TestEventsMapping(t *testing.T) {
	rt, done := newStripeServer(t)
	defer done()

	events, err := listEvents(rt)
	require.NoError(t, err)
	require.Len(t, events, 1)

	ev := events[0].(*mqlStripeEvent)
	assert.Equal(t, "evt_1", ev.Id.Data)
	assert.Equal(t, "charge.succeeded", ev.Type.Data)
	assert.Equal(t, int64(2), ev.PendingWebhooks.Data)
	assert.Equal(t, "req_1", ev.RequestId.Data)
	assert.Equal(t, "key_1", ev.RequestIdempotencyKey.Data)
}

func TestProductsMapping(t *testing.T) {
	rt, done := newStripeServer(t)
	defer done()

	s := &mqlStripe{MqlRuntime: rt}
	products, err := s.products()
	require.NoError(t, err)
	require.Len(t, products, 1)

	p := products[0].(*mqlStripeProduct)
	assert.Equal(t, "prod_1", p.Id.Data)
	assert.Equal(t, "Widget", p.Name.Data)
	assert.True(t, p.Active.Data)
	assert.Equal(t, "good", p.Type.Data)
	require.NotNil(t, p.Updated.Data)
	assert.Equal(t, int64(1700000700), p.Updated.Data.Unix())
}

// TestInitRequiresID verifies the init functions reject a missing/empty id with
// an error instead of falling through to a blank resource.
func TestInitRequiresID(t *testing.T) {
	rt, done := newStripeServer(t)
	defer done()

	_, err := NewResource(rt, "stripe.customer", map[string]*llx.RawData{})
	require.Error(t, err, "stripe.customer with no id must error")

	_, err = NewResource(rt, "stripe.customer", map[string]*llx.RawData{"id": llx.StringData("")})
	require.Error(t, err, "stripe.customer with empty id must error")

	_, err = NewResource(rt, "stripe.product", map[string]*llx.RawData{})
	require.Error(t, err, "stripe.product with no id must error")
}
