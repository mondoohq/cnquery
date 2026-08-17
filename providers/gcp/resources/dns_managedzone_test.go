// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/api/dns/v1"
)

func TestManagedZoneServiceDirectoryNamespaceUrl(t *testing.T) {
	tests := []struct {
		name string
		cfg  *dns.ManagedZoneServiceDirectoryConfig
		want string
	}{
		{
			name: "no config",
			cfg:  nil,
			want: "",
		},
		{
			// A config block with no namespace still means the zone is not
			// backed by Service Directory. Dereferencing it would panic and
			// take the whole scan with it.
			name: "config present but namespace absent",
			cfg:  &dns.ManagedZoneServiceDirectoryConfig{},
			want: "",
		},
		{
			name: "namespace url reported",
			cfg: &dns.ManagedZoneServiceDirectoryConfig{
				Namespace: &dns.ManagedZoneServiceDirectoryConfigNamespace{
					NamespaceUrl: "https://servicedirectory.googleapis.com/v1/projects/p/locations/l/namespaces/n",
				},
			},
			want: "https://servicedirectory.googleapis.com/v1/projects/p/locations/l/namespaces/n",
		},
		{
			name: "namespace present with empty url",
			cfg: &dns.ManagedZoneServiceDirectoryConfig{
				Namespace: &dns.ManagedZoneServiceDirectoryConfigNamespace{},
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, managedZoneServiceDirectoryNamespaceUrl(tt.cfg))
		})
	}
}

func TestManagedZoneReverseLookupEnabled(t *testing.T) {
	// The API carries no boolean here: an empty but non-nil config block is how
	// it reports a reverse-lookup zone. Reading a field instead of the block's
	// presence would report every such zone as disabled.
	assert.False(t, managedZoneReverseLookupEnabled(nil))
	assert.True(t, managedZoneReverseLookupEnabled(&dns.ManagedZoneReverseLookupConfig{}))
	assert.True(t, managedZoneReverseLookupEnabled(&dns.ManagedZoneReverseLookupConfig{Kind: "dns#managedZoneReverseLookupConfig"}))
}
