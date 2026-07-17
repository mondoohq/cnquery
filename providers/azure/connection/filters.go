// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"slices"
	"strings"

	"go.mondoo.com/mql/v13/providers-sdk/v1/util/filteropts"
)

// DiscoveryFilters narrows what the Azure provider discovers. It mirrors the
// AWS provider's DiscoveryFilters so the two share the same shape: the raw
// --filters key/value options carried on inventory.Discovery.Filter are parsed
// into this typed struct once, at connection time, and read from conn.Filters
// during discovery.
type DiscoveryFilters struct {
	Subscriptions SubscriptionsFilter
	// PropagateSubscriptionTags merges each subscription's tags into every asset
	// discovered under that subscription (an asset's own labels win on collision).
	// Off by default. Mirrors the GCP provider's PropagateProjectLabels and the
	// AWS provider's PropagateAccountTags.
	PropagateSubscriptionTags bool
	// SubscriptionTags is an optional injected override. When non-empty it is used
	// instead of fetching each subscription's tags from the API, and applies to
	// every discovered subscription. Mirrors the AWS provider's AccountTags.
	SubscriptionTags map[string]string
}

// SubscriptionsFilter selects which subscriptions to discover. An empty Include
// means "all subscriptions"; a non-empty Include restricts discovery to exactly
// those subscriptions. Exclude removes matches when Include is empty.
type SubscriptionsFilter struct {
	Exclude []string
	Include []string
}

// IsFilteredOut reports whether the subscription with the given ID should be
// skipped during discovery. A non-empty Include list short-circuits: only
// subscriptions in it are kept, and Exclude is ignored. When Include is empty,
// a subscription is skipped only if it appears in Exclude.
//
// note: if this function returns `true`, it means that the subscription should
// be skipped.
func (f SubscriptionsFilter) IsFilteredOut(subscriptionID string) bool {
	if len(f.Include) > 0 {
		return !slices.Contains(f.Include, subscriptionID)
	}
	return slices.Contains(f.Exclude, subscriptionID)
}

// DiscoveryFiltersFromOpts parses the raw --filters key/value options into the
// typed DiscoveryFilters. It is nil-safe: a nil opts map yields empty filters.
func DiscoveryFiltersFromOpts(opts map[string]string) DiscoveryFilters {
	return DiscoveryFilters{
		Subscriptions: SubscriptionsFilter{
			Include: filteropts.ParseCsvSliceOpt(opts, "subscriptions"),
			Exclude: filteropts.ParseCsvSliceOpt(opts, "subscriptions-exclude"),
		},
		PropagateSubscriptionTags: filteropts.ParseBoolOpt(opts, "propagate-subscription-tags", false),
		SubscriptionTags:          parseMapOpt(opts, "subscription-tag:"),
	}
}

// parseMapOpt collects all opts whose key starts with keyPrefix into a map,
// trimming the prefix from each key. Empty keys or values are skipped. Returns a
// non-nil empty map when nothing matches.
func parseMapOpt(opts map[string]string, keyPrefix string) map[string]string {
	res := map[string]string{}
	for k, v := range opts {
		if k == "" || v == "" {
			continue
		}
		if !strings.HasPrefix(k, keyPrefix) {
			continue
		}
		res[strings.TrimPrefix(k, keyPrefix)] = v
	}
	return res
}
