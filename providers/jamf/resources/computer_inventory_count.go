// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

func (r *mqlJamf) computerInventoryCount() (int64, error) {
	// Derive the count from the cached inventory rather than issuing a
	// second paginated fetch — the Jamf SDK paginates through every record
	// regardless of the requested page-size, so a "count only" call would
	// still load the full dataset.
	inv := r.GetComputerInventory()
	if inv.Error != nil {
		return 0, inv.Error
	}
	return int64(len(inv.Data)), nil
}
