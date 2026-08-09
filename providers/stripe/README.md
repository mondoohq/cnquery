# Stripe Provider

Query and assess the configuration and security posture of your Stripe account, including account capabilities, webhook endpoints, customers, products, prices, subscriptions, and the current balance.

## Prerequisites

You need a Stripe **secret key**:

1. Go to [Stripe Dashboard > Developers > API keys](https://dashboard.stripe.com/apikeys)
2. Copy the secret key (`sk_live_...` or `sk_test_...`), or create a **restricted key** (`rk_...`) with read-only permissions

> A read-only restricted key is recommended: the provider only performs read (`GET`) requests, so it never needs write access.

## Authentication

### Environment Variable (recommended)

```bash
export STRIPE_API_KEY="<your-secret-key>"
mql shell stripe
```

### CLI Flag

```bash
mql shell stripe --token <secret-key>
```

### Stripe Connect

To scope the connection to a connected account, pass its account ID. The provider sends it as the `Stripe-Account` header on every request:

```bash
mql shell stripe --token <secret-key> --account acct_1234567890
```

## Examples

**Account activation posture**

```bash
mql> stripe.account { id type country chargesEnabled payoutsEnabled detailsSubmitted }
```

**Requested capabilities and their status**

```bash
mql> stripe.account.capabilities
```

**Audit webhook endpoints for plaintext (non-HTTPS) destinations**

```bash
mql> stripe.webhookEndpoints.where(url.downcase.contains("http://")) { url status }
```

**Find webhook endpoints subscribed to every event**

```bash
mql> stripe.webhookEndpoints.where(enabledEvents.contains("*")) { url enabledEvents }
```

**List enabled webhook endpoints**

```bash
mql> stripe.webhookEndpoints.where(status == "enabled") { url apiVersion enabledEvents }
```

**List customers**

```bash
mql> stripe.customers { id email name delinquent }
```

**Find delinquent customers**

```bash
mql> stripe.customers.where(delinquent) { id email balance }
```

**List active products**

```bash
mql> stripe.products.where(active) { id name type }
```

**Inspect prices and the product each is attached to**

```bash
mql> stripe.prices { id currency unitAmount type recurringInterval product { name } }
```

**Find recurring prices**

```bash
mql> stripe.prices.where(type == "recurring") { id currency unitAmount recurringInterval }
```

**List subscriptions with the customer being billed**

```bash
mql> stripe.subscriptions { id status currency customer { email } }
```

**Find subscriptions set to cancel at period end**

```bash
mql> stripe.subscriptions.where(cancelAtPeriodEnd) { id status currentPeriodEnd }
```

**Current account balance**

```bash
mql> stripe.balance { available pending livemode }
```

**Account overview**

```bash
mql> stripe { customers.length products.length subscriptions.length webhookEndpoints.length }
```

## Resources

| Resource | Description |
|---|---|
| `stripe.account` | Account settings, capabilities, and activation posture |
| `stripe.webhookEndpoints` | Webhook endpoints registered to receive events |
| `stripe.customers` | Customers in the account |
| `stripe.products` | Products in the catalog |
| `stripe.prices` | Prices attached to products |
| `stripe.subscriptions` | Recurring subscriptions |
| `stripe.balance` | Current available and pending balance |

The `stripe.customer` and `stripe.product` resources are also selectable directly by id, for example `stripe.customer(id: "cus_...")` and `stripe.product(id: "prod_...")`.

## Notes

The provider pins the Stripe API version (`Stripe-Version` header) so response shapes stay stable regardless of the account's default version. A restricted key that lacks read access to a given resource degrades that resource to null rather than failing the whole scan.
