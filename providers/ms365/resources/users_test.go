// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// job()/contact() resolve from the bulk user list response (userSelectFields)
// instead of an N+1 per-user Get. That optimization only holds if every field
// the job/contact dicts read is also requested by the bulk list. This guards
// against silently reintroducing the N+1 (or dropping data) by removing a field
// from userSelectFields.
func TestUserSelectFieldsCoverJobContactFields(t *testing.T) {
	selected := make(map[string]bool, len(userSelectFields))
	for _, f := range userSelectFields {
		selected[f] = true
	}
	for _, f := range userJobContactFields {
		assert.Truef(t, selected[f],
			"userJobContactFields[%q] is not in userSelectFields; job()/contact() "+
				"would fall back to an N+1 per-user Get (or lose this field)", f)
	}
}
