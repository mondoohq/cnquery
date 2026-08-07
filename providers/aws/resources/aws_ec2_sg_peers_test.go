// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPeerIsResolvable(t *testing.T) {
	const scanning = "111111111111"

	tests := []struct {
		name                string
		groupID             string
		accountID           string
		peeringConnectionID string
		want                bool
	}{
		{
			name:    "same account, no peering",
			groupID: "sg-123", accountID: scanning,
			want: true,
		},
		{
			// AWS omits UserId for references inside the scanned account.
			name:    "no owner named means same account",
			groupID: "sg-123", accountID: "",
			want: true,
		},
		{
			// The group lives in another account and cannot be described with
			// these credentials. Resolving it anyway would yield a resource that
			// fails to load, which reads as an absent group rather than a
			// cross-account reference.
			name:    "other account is not resolvable",
			groupID: "sg-123", accountID: "999999999999",
			want: false,
		},
		{
			// Peering may place the group in another region, and the rule
			// carries no region of its own to build an ARN from.
			name:    "peering connection is not resolvable",
			groupID: "sg-123", accountID: scanning, peeringConnectionID: "pcx-abc",
			want: false,
		},
		{
			name:    "cross account across peering is not resolvable",
			groupID: "sg-123", accountID: "999999999999", peeringConnectionID: "pcx-abc",
			want: false,
		},
		{
			name:    "missing group id is not resolvable",
			groupID: "", accountID: scanning,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := peerIsResolvable(tt.groupID, tt.accountID, tt.peeringConnectionID, scanning)
			assert.Equal(t, tt.want, got)
		})
	}
}
