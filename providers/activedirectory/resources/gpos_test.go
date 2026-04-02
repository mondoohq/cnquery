// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import "testing"

func TestParseGPLinks(t *testing.T) {
	tests := []struct {
		name   string
		gplink string
		want   []gpLinkEntry
	}{
		{
			name:   "empty string",
			gplink: "",
			want:   nil,
		},
		{
			name:   "single enabled link",
			gplink: "[LDAP://CN={31B2F340-016D-11D2-945F-00C04FB984F9},CN=Policies,CN=System,DC=corp,DC=com;0]",
			want: []gpLinkEntry{
				{
					gpoDN:    "cn={31b2f340-016d-11d2-945f-00c04fb984f9},cn=policies,cn=system,dc=corp,dc=com",
					rawDN:    "CN={31B2F340-016D-11D2-945F-00C04FB984F9},CN=Policies,CN=System,DC=corp,DC=com",
					status:   0,
					order:    1,
					enabled:  true,
					enforced: false,
				},
			},
		},
		{
			name:   "single disabled link",
			gplink: "[LDAP://CN={GUID},CN=Policies,CN=System,DC=corp,DC=com;1]",
			want: []gpLinkEntry{
				{
					gpoDN:    "cn={guid},cn=policies,cn=system,dc=corp,dc=com",
					rawDN:    "CN={GUID},CN=Policies,CN=System,DC=corp,DC=com",
					status:   1,
					order:    1,
					enabled:  false,
					enforced: false,
				},
			},
		},
		{
			name:   "single enforced link",
			gplink: "[LDAP://CN={GUID},CN=Policies,CN=System,DC=corp,DC=com;2]",
			want: []gpLinkEntry{
				{
					gpoDN:    "cn={guid},cn=policies,cn=system,dc=corp,dc=com",
					rawDN:    "CN={GUID},CN=Policies,CN=System,DC=corp,DC=com",
					status:   2,
					order:    1,
					enabled:  true,
					enforced: true,
				},
			},
		},
		{
			name:   "disabled and enforced",
			gplink: "[LDAP://CN={GUID},CN=Policies,CN=System,DC=corp,DC=com;3]",
			want: []gpLinkEntry{
				{
					gpoDN:    "cn={guid},cn=policies,cn=system,dc=corp,dc=com",
					rawDN:    "CN={GUID},CN=Policies,CN=System,DC=corp,DC=com",
					status:   3,
					order:    1,
					enabled:  false,
					enforced: true,
				},
			},
		},
		{
			name: "multiple links with order assignment",
			gplink: "[LDAP://CN={AAA},CN=Policies,CN=System,DC=corp,DC=com;0]" +
				"[LDAP://CN={BBB},CN=Policies,CN=System,DC=corp,DC=com;0]" +
				"[LDAP://CN={CCC},CN=Policies,CN=System,DC=corp,DC=com;0]",
			want: []gpLinkEntry{
				{
					gpoDN: "cn={aaa},cn=policies,cn=system,dc=corp,dc=com",
					rawDN: "CN={AAA},CN=Policies,CN=System,DC=corp,DC=com",
					order: 3, enabled: true,
				},
				{
					gpoDN: "cn={bbb},cn=policies,cn=system,dc=corp,dc=com",
					rawDN: "CN={BBB},CN=Policies,CN=System,DC=corp,DC=com",
					order: 2, enabled: true,
				},
				{
					gpoDN: "cn={ccc},cn=policies,cn=system,dc=corp,dc=com",
					rawDN: "CN={CCC},CN=Policies,CN=System,DC=corp,DC=com",
					order: 1, enabled: true,
				},
			},
		},
		{
			name: "mixed enabled disabled enforced",
			gplink: "[LDAP://CN={A},CN=Policies,CN=System,DC=x,DC=com;0]" +
				"[LDAP://CN={B},CN=Policies,CN=System,DC=x,DC=com;1]" +
				"[LDAP://CN={C},CN=Policies,CN=System,DC=x,DC=com;2]",
			want: []gpLinkEntry{
				{gpoDN: "cn={a},cn=policies,cn=system,dc=x,dc=com", rawDN: "CN={A},CN=Policies,CN=System,DC=x,DC=com", status: 0, order: 3, enabled: true, enforced: false},
				{gpoDN: "cn={b},cn=policies,cn=system,dc=x,dc=com", rawDN: "CN={B},CN=Policies,CN=System,DC=x,DC=com", status: 1, order: 2, enabled: false, enforced: false},
				{gpoDN: "cn={c},cn=policies,cn=system,dc=x,dc=com", rawDN: "CN={C},CN=Policies,CN=System,DC=x,DC=com", status: 2, order: 1, enabled: true, enforced: true},
			},
		},
		{
			name:   "malformed entry without semicolon skipped",
			gplink: "[LDAP://CN={GUID},CN=Policies,CN=System,DC=corp,DC=com]",
			want:   []gpLinkEntry{},
		},
		{
			name:   "whitespace padded",
			gplink: "  [LDAP://CN={GUID},CN=Policies,CN=System,DC=corp,DC=com;0]  ",
			want: []gpLinkEntry{
				{
					gpoDN: "cn={guid},cn=policies,cn=system,dc=corp,dc=com",
					rawDN: "CN={GUID},CN=Policies,CN=System,DC=corp,DC=com",
					order: 1, enabled: true,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseGPLinks(tt.gplink)

			if tt.want == nil {
				if got != nil {
					t.Fatalf("parseGPLinks(%q) = %v, want nil", tt.gplink, got)
				}
				return
			}

			if len(got) != len(tt.want) {
				t.Fatalf("parseGPLinks(%q) returned %d entries, want %d", tt.gplink, len(got), len(tt.want))
			}

			for i, w := range tt.want {
				g := got[i]
				if g.gpoDN != w.gpoDN {
					t.Errorf("[%d] gpoDN = %q, want %q", i, g.gpoDN, w.gpoDN)
				}
				if g.rawDN != w.rawDN {
					t.Errorf("[%d] rawDN = %q, want %q", i, g.rawDN, w.rawDN)
				}
				if g.order != w.order {
					t.Errorf("[%d] order = %d, want %d", i, g.order, w.order)
				}
				if g.enabled != w.enabled {
					t.Errorf("[%d] enabled = %v, want %v", i, g.enabled, w.enabled)
				}
				if g.enforced != w.enforced {
					t.Errorf("[%d] enforced = %v, want %v", i, g.enforced, w.enforced)
				}
				if g.status != w.status {
					t.Errorf("[%d] status = %d, want %d", i, g.status, w.status)
				}
			}
		})
	}
}
