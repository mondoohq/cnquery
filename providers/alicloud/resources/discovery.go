// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

// Discovery targets recognized by the alicloud provider. `auto` and `all`
// resolve to the account asset today; per-resource asset discovery (turning
// each ECS instance, RDS instance, etc. into its own scannable asset) is layered
// on top of these constants as it is implemented.
const (
	DiscoveryAuto     = "auto"
	DiscoveryAll      = "all"
	DiscoveryAccounts = "accounts"
)
