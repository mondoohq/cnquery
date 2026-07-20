// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Exchange domain collections (e.g. AllowedSenderDomains on hosted content
// filter policies, Domains on sharing policies) are MultiValuedProperty<...>
// values. Depending on the cmdlet and ConvertTo-Json serialization each entry
// is emitted as a bare string or as an object with a "Domain" field, and a
// single-element collection may be a scalar rather than an array. Decoding any
// of these into exchangeDomainList must succeed and normalize to []string.
func TestExchangeDomainList_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{"null", `null`, nil},
		{"empty array", `[]`, []string{}},
		{"string array", `["contoso.com","fabrikam.com"]`, []string{"contoso.com", "fabrikam.com"}},
		{"single string", `"contoso.com"`, []string{"contoso.com"}},
		{
			"object array",
			`[{"Domain":"contoso.com","IncludeSubDomains":false},{"Domain":"fabrikam.com"}]`,
			[]string{"contoso.com", "fabrikam.com"},
		},
		{"single object", `{"Domain":"contoso.com","IncludeSubDomains":true}`, []string{"contoso.com"}},
		{"object without domain", `[{"Foo":"bar"}]`, []string{}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var got exchangeDomainList
			require.NoError(t, json.Unmarshal([]byte(tc.in), &got))
			assert.Equal(t, tc.want, []string(got))
		})
	}
}

// Regression: a hosted content filter policy whose AllowedSenderDomains is an
// array of SmtpDomainWithSubdomains objects previously aborted the whole
// resource with "cannot unmarshal object ... of type string".
func TestExchangeHostedContentFilterPolicy_ObjectAllowedSenderDomains(t *testing.T) {
	raw := `{
		"Identity": "Default",
		"Name": "Default",
		"AllowedSenderDomains": [{"Domain":"contoso.com","IncludeSubDomains":false}],
		"BlockedSenderDomains": []
	}`
	var p ExchangeHostedContentFilterPolicy
	require.NoError(t, json.Unmarshal([]byte(raw), &p))
	assert.Equal(t, []string{"contoso.com"}, []string(p.AllowedSenderDomains))
	assert.Empty(t, []string(p.BlockedSenderDomains))
}

// Regression: sharing policy Domains are SharingPolicyDomain objects and must
// decode without aborting the resource.
func TestExchangeSharingPolicy_ObjectDomains(t *testing.T) {
	raw := `{
		"Identity": "Default Sharing Policy",
		"Name": "Default Sharing Policy",
		"Enabled": true,
		"Default": true,
		"Domains": [{"Domain":"contoso.com","DomainType":"CalendarSharingFreeBusySimple"}]
	}`
	var p ExchangeSharingPolicy
	require.NoError(t, json.Unmarshal([]byte(raw), &p))
	assert.Equal(t, []string{"contoso.com"}, []string(p.Domains))
}
