// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEventFilterID(t *testing.T) {
	// Eventarc triggers can declare multiple filters sharing an attribute
	// name but matching different values. Without the value in the id,
	// siblings collide in the runtime cache and one wins.
	cases := []struct {
		name string
		want string
		got  string
	}{
		{
			name: "happy path",
			want: "trigger1/eventFilters/type/google.cloud.storage.object.v1.finalized",
			got:  eventFilterID("trigger1", "type", "google.cloud.storage.object.v1.finalized"),
		},
		{
			name: "same attribute, different value",
			want: "trigger1/eventFilters/type/google.cloud.audit.log.v1.written",
			got:  eventFilterID("trigger1", "type", "google.cloud.audit.log.v1.written"),
		},
		{
			name: "different attribute, same value",
			want: "trigger1/eventFilters/serviceName/storage.googleapis.com",
			got:  eventFilterID("trigger1", "serviceName", "storage.googleapis.com"),
		},
	}
	seen := make(map[string]string, len(cases))
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.got)
			if prev, exists := seen[tc.got]; exists {
				t.Fatalf("id collision: %q produced by both %q and %q", tc.got, prev, tc.name)
			}
			seen[tc.got] = tc.name
		})
	}
}
