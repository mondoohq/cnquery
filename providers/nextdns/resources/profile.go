// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

func (r *mqlNextdnsProfile) id() (string, error) {
	return "nextdns.profile/" + r.Id.Data, nil
}
