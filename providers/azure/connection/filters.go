// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/filteropts"
)

// DiscoveryFilters narrows what the Azure provider discovers. It mirrors the
// AWS provider's DiscoveryFilters so the two share the same shape: the raw
// --filters key/value options carried on inventory.Discovery.Filter are parsed
// into this typed struct once, at connection time, and read from conn.Filters
// during discovery.
type DiscoveryFilters struct {
	Subscriptions SubscriptionsFilter
}

// SubscriptionsFilter selects which subscriptions to discover. An empty Include
// means "all subscriptions"; a non-empty Include restricts discovery to exactly
// those subscriptions. Exclude removes matches when Include is empty.
type SubscriptionsFilter struct {
	Exclude []string
	Include []string
}

// DiscoveryFiltersFromOpts parses the raw --filters key/value options into the
// typed DiscoveryFilters. It is nil-safe: a nil opts map yields empty filters.
func DiscoveryFiltersFromOpts(opts map[string]string) DiscoveryFilters {
	return DiscoveryFilters{
		Subscriptions: SubscriptionsFilter{
			Include: filteropts.ParseCsvSliceOpt(opts, "subscriptions"),
			Exclude: filteropts.ParseCsvSliceOpt(opts, "subscriptions-exclude"),
		},
	}
}
