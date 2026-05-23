// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import "time"

// daysBetween returns the number of full 24-hour periods from `from` to
// `to`, truncated toward zero. A 5-hour gap reports 0, a 25-hour gap
// reports 1, and an already-expired certificate reports a negative
// number so audits can write `daysUntilExpiry < 30` (warn) and
// `daysUntilExpiry < 0` (already expired) against the same field.
func daysBetween(from, to time.Time) int64 {
	return int64(to.Sub(from) / (24 * time.Hour))
}

func (r *mqlProxmoxCertificate) daysUntilExpiry() (int64, error) {
	if r.NotAfter.Data == nil {
		return 0, nil
	}
	return daysBetween(time.Now(), *r.NotAfter.Data), nil
}
