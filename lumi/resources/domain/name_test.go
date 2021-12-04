package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDomainName(t *testing.T) {
	tests := map[string]*Name{
		"https://google.com": {
			Host:                "google.com",
			EffectiveTLDPlusOne: "google.com",
			TLD:                 "com",
			IcannManagedTLD:     true,
			Labels:              []string{"google", "com"},
		},
		"https://mail.google.com": {
			Host:                "mail.google.com",
			EffectiveTLDPlusOne: "google.com",
			TLD:                 "com",
			IcannManagedTLD:     true,
			Labels:              []string{"mail", "google", "com"},
		},
		"https://blog.google": {
			Host:                "blog.google",
			EffectiveTLDPlusOne: "blog.google",
			TLD:                 "google",
			IcannManagedTLD:     true,
			Labels:              []string{"blog", "google"},
		},
		"https://hello.example": {
			Host:                "hello.example",
			EffectiveTLDPlusOne: "hello.example",
			TLD:                 "example",
			IcannManagedTLD:     false,
			Labels:              []string{"hello", "example"},
		},
		"https://hello.notpublicsuffix": {
			Host:                "hello.notpublicsuffix",
			EffectiveTLDPlusOne: "hello.notpublicsuffix",
			TLD:                 "notpublicsuffix",
			IcannManagedTLD:     false,
			Labels:              []string{"hello", "notpublicsuffix"},
		},
		"https://mondoo.io": {
			Host:                "mondoo.io",
			EffectiveTLDPlusOne: "mondoo.io",
			TLD:                 "io",
			IcannManagedTLD:     true,
			Labels:              []string{"mondoo", "io"},
		},
		"mondoo.io": {
			Host:                "mondoo.io",
			EffectiveTLDPlusOne: "mondoo.io",
			TLD:                 "io",
			IcannManagedTLD:     true,
			Labels:              []string{"mondoo", "io"},
		},
	}
	for k, expected := range tests {
		dn, err := Parse(k)
		require.NoError(t, err)
		assert.Equal(t, expected, dn, k)
	}
}
