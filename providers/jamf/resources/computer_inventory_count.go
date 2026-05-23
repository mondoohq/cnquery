// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"go.mondoo.com/mql/v13/providers/jamf/connection"
)

// totalCountResponse decodes the first page of a Jamf paginated endpoint
// when we only need the total. The Jamf Pro API returns `totalCount` in the
// first response, so a single GET with page-size=1 is enough.
type totalCountResponse struct {
	TotalCount int `json:"totalCount"`
}

func (r *mqlJamf) computerInventoryCount() (int64, error) {
	// If the full inventory was already fetched in this session, reuse it
	// instead of issuing another HTTP call.
	if r.ComputerInventory.IsSet() && r.ComputerInventory.Error == nil {
		return int64(len(r.ComputerInventory.Data)), nil
	}

	conn := r.MqlRuntime.Connection.(*connection.JamfConnection)
	var out totalCountResponse
	resp, err := conn.Client.HTTP.DoRequest("GET", "/api/v1/computers-inventory?page=0&page-size=1", nil, &out)
	if err != nil {
		return 0, err
	}
	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}
	return int64(out.TotalCount), nil
}
