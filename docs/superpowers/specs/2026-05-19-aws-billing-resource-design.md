# AWS Billing Resource — Design

**Date:** 2026-05-19
**Provider:** `aws` (currently at `13.26.1`; new entries land at `13.26.2`)
**Status:** Design approved; ready for implementation plan.

## Goal

Add an MQL resource family for AWS billing so users can audit:

- Current and recent spend (month-to-date, last month, last 30 days)
- Forecasted spend for the current month
- Per-service spend breakdowns
- Configured AWS Budgets, including thresholds, actuals, forecasts, filters, and notification subscribers

Out of scope for v1 (could be follow-ups): cost anomalies, Savings Plans inventory and utilization, Reserved Instance utilization/coverage, Cost Optimization Hub recommendations, Free Tier usage, BCM Data Exports / legacy CUR.

## Why these APIs, why this scope

Two AWS APIs cover v1:

- **Cost Explorer (`ce`)** — `GetCostAndUsage`, `GetCostForecast`. us-east-1-only endpoint. **AWS bills $0.01 per paginated request**, which shapes the design (see Caching).
- **Budgets (`budgets`)** — `DescribeBudgets`, `DescribeNotificationsForBudget`, `DescribeSubscribersForNotification`. us-east-1-only endpoint. No per-call charge.

Both are account-global (not regional). Neither requires the AWS provider to enumerate regions.

## Resource shape

Two resources:

```
aws.billing                # account-global namespace
aws.billing.budget         # private sub-resource, one per configured AWS Budget
```

Placed alongside other root-level service namespaces (`aws.iam`, `aws.ec2`, `aws.s3`, …). Not under `aws.account` — that resource is for account identity/metadata, not service data.

### `aws.billing`

```
// AWS Billing
//
// Examine the AWS account's billing posture: month-to-date and recent
// historical spend (from AWS Cost Explorer), service-level breakdowns,
// forecasts for the current month, and the inventory of configured
// AWS Budgets. Cost Explorer must be enabled in the account; if not,
// cost fields return zero. Each Cost Explorer API call is billed by
// AWS at $0.01, so cost-related fields are fetched lazily and cached.
aws.billing @defaults("monthToDateCost currency budgets.length") {
  // Currency the account is billed in (e.g., USD)
  currency() string
  // Month-to-date unblended cost in `currency`
  monthToDateCost() float
  // Total unblended cost for the previous full calendar month
  lastMonthCost() float
  // Total unblended cost for the trailing 30 days
  last30DaysCost() float
  // Forecasted total unblended cost for the current calendar month:
  // month-to-date actuals plus a Cost Explorer forecast for the
  // remainder of the month.
  forecastedMonthCost() float
  // Month-to-date unblended cost broken down by AWS service name
  costsByService() map[string]float
  // Previous full calendar month unblended cost broken down by AWS service name
  costsByServiceLastMonth() map[string]float
  // All AWS Budgets configured on the account
  budgets() []aws.billing.budget
}
```

`__id`: the AWS account ID.

### `aws.billing.budget`

```
// AWS Budget
//
// Examine a single AWS Budgets entry. The `name` field selects the budget
// (unique per account), e.g. `aws.billing.budgets.where(name == "monthly-cap")`.
// Surfaces the budget's type, time unit, limit, actual and forecasted
// spend, cost filters, and the configured notifications and subscribers.
private aws.billing.budget @defaults("name budgetType budgetLimit actualSpend") {
  // Budget name (unique within an account)
  name string
  // Account ID that owns the budget
  accountId string
  // Budget type: COST, USAGE, RI_UTILIZATION, RI_COVERAGE,
  // SAVINGS_PLANS_UTILIZATION, or SAVINGS_PLANS_COVERAGE
  budgetType string
  // Reporting period: DAILY, MONTHLY, QUARTERLY, or ANNUALLY
  timeUnit string
  // Configured budget ceiling
  budgetLimit float
  // Unit of `budgetLimit` (e.g., USD, GB, hours)
  budgetLimitUnit string
  // Actual spend/usage in the current period
  actualSpend float
  // Forecasted spend/usage for the current period
  forecastedSpend float
  // Cost filter dimensions applied to the budget
  // (e.g., "Service" -> ["Amazon EC2"], "LinkedAccount" -> ["123..."])
  costFilters map[string][]string
  // Start time of the current budget period
  periodStart time
  // End time of the current budget period
  periodEnd time
  // Last time the budget definition was updated
  lastUpdatedTime time
  // Notifications configured on the budget. Each entry has keys:
  // notificationType (ACTUAL|FORECASTED), comparisonOperator, threshold,
  // thresholdType (PERCENTAGE|ABSOLUTE_VALUE), notificationState
  // (OK|ALARM), and subscribers ([{type, address}]).
  notifications() []dict
}
```

`__id`: `<accountId>/<budgetName>`.

#### Why `notifications` is `[]dict`, not a sub-resource

Per CLAUDE.md's "When to create a sub-resource" rule, budget notifications have no stable identifier (they're inline children of a budget) and don't nest typed refs to other modeled resources. They're a small heterogeneous struct — `[]dict` keeps the schema lean while preserving queryability (`notifications.where(notificationType == "ACTUAL" && threshold > 80)`). Subscribers nest inside each notification dict as `[{type: "EMAIL"|"SNS", address: "..."}]` for the same reason.

## Caching strategy

Cost Explorer charges per paginated API request, so v1 is built around minimizing CE calls.

Single `GetCostAndUsage` calls with `GroupBy: SERVICE` return both the per-service breakdown *and* the totals. So:

- `monthToDateCost` and `costsByService` share one CE call (period: month-to-date).
- `lastMonthCost` and `costsByServiceLastMonth` share one CE call (period: previous full month).
- `last30DaysCost` is one CE call (period: trailing 30 days, total only).
- `forecastedMonthCost` is one `GetCostForecast` call (start = tomorrow, end = first day of next month) summed with the cached `monthToDateCost` from the first call above. If today is the last day of the month, the forecast call is skipped and the field equals MTD.

So a "fetch everything" query is **4 CE calls = $0.04**. Within a single scan, the runtime resource cache (`__id`-keyed) prevents repeats.

Implementation: an `mqlAwsBillingInternal` struct holds the cached `*costAndUsageResult` for each of the three windows plus a `sync.Mutex` + `fetched` flag for each, following the double-check locking pattern documented in CLAUDE.md.

```go
type mqlAwsBillingInternal struct {
    mtdLock        sync.Mutex
    mtdFetched     bool
    mtdResult      *costAndUsageResult

    lastMonthLock    sync.Mutex
    lastMonthFetched bool
    lastMonthResult  *costAndUsageResult

    last30Lock    sync.Mutex
    last30Fetched bool
    last30Total   float64

    forecastLock    sync.Mutex
    forecastFetched bool
    forecastAmount  float64

    currency string // populated by the first CE call that returns a unit
}
```

## Error handling

- **Cost Explorer not enabled** — first CE call returns `OptInRequired` / `AccessDenied`. Detect via existing `Is400AccessDeniedError` helper, log a debug message, return `0` for cost fields and `{}` for breakdowns. The rest of the scan proceeds.
- **`budgets:Describe*` denied** — return an empty `[]aws.billing.budget` rather than erroring the whole resource.
- **Pagination** — both `DescribeBudgets` and `DescribeNotificationsForBudget` paginate; follow the existing provider pagination pattern from CLAUDE.md.
- **Cost Explorer values are strings in the SDK** (e.g. `"123.4567890"`); parse to `float64` once at fetch time.

## Region + auth

- All clients pinned to `us-east-1`.
- Reuses existing `connection.AwsConnection` config/credential plumbing — same pattern as other `aws_*.go` resource files.
- New IAM permissions to record in `aws.permissions.json` (regenerated by build):
  - `ce:GetCostAndUsage`
  - `ce:GetCostForecast`
  - `budgets:DescribeBudgets`
  - `budgets:DescribeNotificationsForBudget`
  - `budgets:DescribeSubscribersForNotification`

## File-level changes

1. `providers/aws/go.mod` — add `github.com/aws/aws-sdk-go-v2/service/costexplorer` and `github.com/aws/aws-sdk-go-v2/service/budgets`.
2. `providers/aws/resources/aws.lr` — add the two resources above.
3. `providers/aws/resources/aws.lr.versions` — add new entries at `13.26.2`.
4. `providers/aws/resources/aws.lr.go` — regenerated via `./mqlr generate`.
5. `providers/aws/resources/aws_billing.go` — new, implementation.
6. `providers/aws/resources/aws.permissions.json` — regenerated by `make providers/build/aws`.

Two `./mqlr generate` passes are required (per CLAUDE.md): first to generate the resource scaffolding, then again after `mqlAwsBillingInternal` is added so the generator can embed it.

## Verification plan

1. `make providers/build/aws && make providers/install/aws`.
2. `mql shell aws` against a real account with Cost Explorer enabled and at least one budget, run:
   ```
   aws.billing { monthToDateCost lastMonthCost last30DaysCost forecastedMonthCost currency }
   aws.billing.costsByService
   aws.billing.budgets { name budgetType budgetLimit actualSpend forecastedSpend notifications }
   ```
3. Confirm graceful degradation on an account with Cost Explorer disabled (zero values, no scan failure).
4. Confirm graceful degradation with budgets read denied (empty list).

## Out of scope for v1 (potential follow-ups)

- Cost anomalies (`ce:GetAnomalies`, monitors, subscriptions).
- Savings Plans inventory (`savingsplans:DescribeSavingsPlans`) and utilization (`ce:GetSavingsPlansUtilization`).
- Reserved Instance coverage/utilization.
- Cost Optimization Hub recommendations.
- Free Tier usage (`freetier:GetFreeTierUsage`).
- BCM Data Exports / legacy CUR inventory.
- Custom date-range query method (escape hatch for ad-hoc audits).
- Breakdowns by linked account and/or region.
