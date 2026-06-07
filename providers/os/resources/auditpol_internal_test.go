// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAuditpolInclusionAudits(t *testing.T) {
	cases := []struct {
		setting          string
		success, failure bool
	}{
		{"Success", true, false},
		{"Failure", false, true},
		{"Success and Failure", true, true},
		{"No Auditing", false, false},
		{"", false, false},
	}

	for _, c := range cases {
		t.Run(c.setting, func(t *testing.T) {
			assert.Equal(t, c.success, auditpolInclusionAudits(c.setting, "success"))
			assert.Equal(t, c.failure, auditpolInclusionAudits(c.setting, "failure"))
		})
	}
}
