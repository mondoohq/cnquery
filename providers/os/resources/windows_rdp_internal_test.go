// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveRdpValue(t *testing.T) {
	t.Run("group policy value wins over the effective value", func(t *testing.T) {
		policy := map[string]int64{"userauthentication": 0}
		effective := map[string]int64{"userauthentication": 1}
		assert.Equal(t, int64(0), resolveRdpValue(policy, effective, "UserAuthentication", 1))
	})

	t.Run("effective value is used when policy is unset", func(t *testing.T) {
		effective := map[string]int64{"securitylayer": 2}
		assert.Equal(t, int64(2), resolveRdpValue(nil, effective, "SecurityLayer", 1))
	})

	t.Run("default is used when neither policy nor effective is set", func(t *testing.T) {
		assert.Equal(t, int64(2), resolveRdpValue(nil, nil, "MinEncryptionLevel", 2))
	})

	t.Run("value name matching is case insensitive", func(t *testing.T) {
		policy := map[string]int64{"fdisablecdm": 1}
		assert.Equal(t, int64(1), resolveRdpValue(policy, nil, "fDisableCdm", 0))
	})

	t.Run("an explicit zero is honored, not treated as unset", func(t *testing.T) {
		policy := map[string]int64{"fencryptrpctraffic": 0}
		// default is 1, but the policy explicitly disables it
		assert.Equal(t, int64(0), resolveRdpValue(policy, nil, "fEncryptRPCTraffic", 1))
	})
}
