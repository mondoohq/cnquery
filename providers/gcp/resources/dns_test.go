// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/api/dns/v1"
)

func TestManagedZoneDnssecNonExistence(t *testing.T) {
	t.Run("nil config returns empty", func(t *testing.T) {
		assert.Equal(t, "", managedZoneDnssecNonExistence(nil))
	})
	t.Run("value is returned", func(t *testing.T) {
		assert.Equal(t, "nsec3", managedZoneDnssecNonExistence(&dns.ManagedZoneDnsSecConfig{NonExistence: "nsec3"}))
	})
}
