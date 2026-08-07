// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

// mqlRedisdbInstanceInternal caches the CONFIG GET result fetched during init so
// the config sub-resource resolves without a second round trip. The code
// generator embeds this into mqlRedisdbInstance.
type mqlRedisdbInstanceInternal struct {
	configCache map[string]string
}
