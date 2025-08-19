// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	vmwaretypes "github.com/vmware/govmomi/vim25/types"
)

// extractTagKeys extracts tag keys from vmware Tag slice
func extractTagKeys(tags []vmwaretypes.Tag) []string {
	tagKeys := make([]string, len(tags))
	for i, tag := range tags {
		tagKeys[i] = tag.Key
	}
	return tagKeys
}
