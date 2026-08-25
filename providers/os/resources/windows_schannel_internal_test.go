// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/llx"
	"go.mondoo.com/mql/providers/os/registry"
)

// TestPqcKeyExchangeEnabled locks in the ML-KEM detection: a supported group is
// post-quantum iff its name contains "MLKEM" (case-insensitive), matching the
// Windows group strings such as secp256r1_mlkem768.
func TestPqcKeyExchangeEnabled(t *testing.T) {
	cases := []struct {
		name   string
		curves []string
		want   bool
	}{
		{
			name:   "mixed list with an ML-KEM group",
			curves: []string{"SecP256r1MLKEM768", "x25519"},
			want:   true,
		},
		{
			name:   "windows lowercase mlkem group string",
			curves: []string{"curve25519", "secp384r1_mlkem1024"},
			want:   true,
		},
		{
			name:   "only classical curves",
			curves: []string{"curve25519", "NistP256"},
			want:   false,
		},
		{
			name:   "empty list",
			curves: []string{},
			want:   false,
		},
		{
			name:   "nil list",
			curves: nil,
			want:   false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, pqcKeyExchangeEnabled(c.curves))
		})
	}
}

// TestSchannelEnabledDecode pins the DWORD decode, including the 0xFFFFFFFF
// "on" spelling. The two registry backends disagree on its sign (PowerShell
// hands back the signed -1, the native API the unsigned 4294967295), so both
// have to read as enabled. An absent value must stay nil, because false would
// claim the administrator turned the algorithm off.
func TestSchannelEnabledDecode(t *testing.T) {
	cases := []struct {
		name  string
		items map[string]registry.RegistryKeyItem
		want  *bool
	}{
		{"explicit 1 is enabled", regItems(map[string]int64{"Enabled": 1}), boolPtr(true)},
		{"0xFFFFFFFF as signed -1 is enabled", regItems(map[string]int64{"Enabled": -1}), boolPtr(true)},
		{"0xFFFFFFFF as unsigned is enabled", regItems(map[string]int64{"Enabled": 4294967295}), boolPtr(true)},
		{"explicit 0 is disabled", regItems(map[string]int64{"Enabled": 0}), boolPtr(false)},
		{"absent value stays nil", regItems(map[string]int64{"DisabledByDefault": 1}), nil},
		{"empty key stays nil", map[string]registry.RegistryKeyItem{}, nil},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := schannelEnabled(c.items, schannelEnabledValue)
			if c.want == nil {
				assert.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			assert.Equal(t, *c.want, *got)
		})
	}
}

// TestSchannelEnabledIsCaseInsensitive covers the registry not enforcing a
// spelling for value names: the lookup map is lower-cased, so a key stored as
// "enabled" must resolve the same as "Enabled".
func TestSchannelEnabledIsCaseInsensitive(t *testing.T) {
	items := regItems(map[string]int64{"enabled": 1})
	got := schannelEnabled(items, "Enabled")
	require.NotNil(t, got)
	assert.True(t, *got)
}

// TestSchannelConfigured covers the tri-state that the whole resource turns on:
// configured says whether the administrator wrote anything at all, separately
// from what they wrote. A key holding only DisabledByDefault is configured even
// though Enabled is absent, and an explicit Enabled=0 is configured too.
func TestSchannelConfigured(t *testing.T) {
	cases := []struct {
		name  string
		items map[string]registry.RegistryKeyItem
		names []string
		want  bool
	}{
		{
			name:  "no values at all",
			items: map[string]registry.RegistryKeyItem{},
			names: []string{schannelEnabledValue, schannelDisabledByDefaultValue},
			want:  false,
		},
		{
			name:  "explicit disable counts as configured",
			items: regItems(map[string]int64{"Enabled": 0}),
			names: []string{schannelEnabledValue, schannelDisabledByDefaultValue},
			want:  true,
		},
		{
			name:  "DisabledByDefault alone counts as configured",
			items: regItems(map[string]int64{"DisabledByDefault": 1}),
			names: []string{schannelEnabledValue, schannelDisabledByDefaultValue},
			want:  true,
		},
		{
			name:  "unrelated value does not count",
			items: regItems(map[string]int64{"SomethingElse": 1}),
			names: []string{schannelEnabledValue},
			want:  false,
		},
		{
			name:  "key exchange minimum key length alone counts as configured",
			items: regItems(map[string]int64{"ServerMinKeyBitLength": 2048}),
			names: []string{schannelEnabledValue, schannelServerMinKeyBitLength, schannelClientMinKeyBitLength},
			want:  true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, schannelConfigured(c.items, c.names...))
		})
	}
}

// TestSchannelProtocolArgsUnconfiguredIsNull is the absent-is-not-disabled
// guarantee stated as a test: a protocol with no registry subkey must report
// configured=false and null (not false) enablement on both sides. If these
// became llx.BoolData(false) the resource would report "TLS 1.0 is disabled"
// on a host where the Windows default leaves it on.
func TestSchannelProtocolArgsUnconfiguredIsNull(t *testing.T) {
	args := schannelProtocolArgs("TLS 1.0", nil, nil)

	assert.Equal(t, "TLS 1.0", args["name"].Value)
	assert.Equal(t, schannelProtocolsPath+`\TLS 1.0`, args["__id"].Value)

	for _, side := range []string{"client", "server"} {
		assert.Equal(t, false, args[side+"Configured"].Value, side+"Configured")
		assert.Same(t, llx.NilData, args[side+"Enabled"], side+"Enabled must be null, not false")
		assert.Same(t, llx.NilData, args[side+"DisabledByDefault"],
			side+"DisabledByDefault must be null, not false")
	}
}

// TestSchannelProtocolArgsPerSide checks that the two sides stay independent: a
// server-side disable must not be reported on the client side, and vice versa.
func TestSchannelProtocolArgsPerSide(t *testing.T) {
	client := regItems(map[string]int64{"Enabled": 0xFFFFFFFF, "DisabledByDefault": 0})
	server := regItems(map[string]int64{"Enabled": 0})

	args := schannelProtocolArgs("TLS 1.2", client, server)

	assert.Equal(t, true, args["clientConfigured"].Value)
	assert.Equal(t, true, args["clientEnabled"].Value)
	assert.Equal(t, false, args["clientDisabledByDefault"].Value)

	assert.Equal(t, true, args["serverConfigured"].Value)
	assert.Equal(t, false, args["serverEnabled"].Value)
	// the server side never wrote DisabledByDefault, so it stays null
	assert.Same(t, llx.NilData, args["serverDisabledByDefault"])
}

// TestSchannelEnablementArgs covers the cipher and hash shape, including that a
// cipher with no subkey reports a null rather than a false enablement.
func TestSchannelEnablementArgs(t *testing.T) {
	off := schannelEnablementArgs(schannelCiphersPath, "RC4 128/128", regItems(map[string]int64{"Enabled": 0}))
	assert.Equal(t, schannelCiphersPath+`\RC4 128/128`, off["__id"].Value)
	assert.Equal(t, "RC4 128/128", off["name"].Value)
	assert.Equal(t, true, off["configured"].Value)
	assert.Equal(t, false, off["enabled"].Value)

	absent := schannelEnablementArgs(schannelHashesPath, "SHA256", map[string]registry.RegistryKeyItem{})
	assert.Equal(t, false, absent["configured"].Value)
	assert.Same(t, llx.NilData, absent["enabled"], "an absent hash override must read null, not false")
}

// TestSchannelKeyExchangeArgs pins the minimum key bit length parsing. The
// Diffie-Hellman entry is commonly configured with a minimum key size and no
// Enabled value at all, so configured has to account for the length values or
// a real hardening setting would report as unconfigured.
func TestSchannelKeyExchangeArgs(t *testing.T) {
	dh := schannelKeyExchangeArgs("Diffie-Hellman", regItems(map[string]int64{
		"ServerMinKeyBitLength": 2048,
		"ClientMinKeyBitLength": 2048,
	}))
	assert.Equal(t, schannelKeyExchangePath+`\Diffie-Hellman`, dh["__id"].Value)
	assert.Equal(t, true, dh["configured"].Value)
	assert.Same(t, llx.NilData, dh["enabled"], "no Enabled value means null, not false")
	assert.Equal(t, int64(2048), dh["serverMinKeyBitLength"].Value)
	assert.Equal(t, int64(2048), dh["clientMinKeyBitLength"].Value)

	// a client-only minimum leaves the server side null rather than 0
	clientOnly := schannelKeyExchangeArgs("PKCS", regItems(map[string]int64{"ClientMinKeyBitLength": 1024}))
	assert.Equal(t, int64(1024), clientOnly["clientMinKeyBitLength"].Value)
	assert.Same(t, llx.NilData, clientOnly["serverMinKeyBitLength"])

	absent := schannelKeyExchangeArgs("ECDH", map[string]registry.RegistryKeyItem{})
	assert.Equal(t, false, absent["configured"].Value)
	assert.Same(t, llx.NilData, absent["enabled"])
	assert.Same(t, llx.NilData, absent["serverMinKeyBitLength"])
	assert.Same(t, llx.NilData, absent["clientMinKeyBitLength"])
}

// TestSchannelChildNames covers both registry backends. The native enumeration
// is one level deep and repeats the parent in Path while the leaf is in Name;
// the PowerShell fallback enumerates recursively and puts the child's full path
// in Path, so its Name is the leaf of a grandchild (Client / Server), not of the
// protocol. Reading the leaf out of Name in that case would report the protocol
// list as {Client, Server}.
func TestSchannelChildNames(t *testing.T) {
	cases := []struct {
		name     string
		parent   string
		children []registry.RegistryKeyChild
		want     []string
	}{
		{
			name:   "native shape: leaf in Name, Path repeats the parent",
			parent: schannelProtocolsPath,
			children: []registry.RegistryKeyChild{
				{Name: "TLS 1.2", Path: schannelProtocolsPath},
				{Name: "SSL 3.0", Path: schannelProtocolsPath},
			},
			want: []string{"TLS 1.2", "SSL 3.0"},
		},
		{
			name:   "powershell shape: recursive, full path in Path",
			parent: schannelProtocolsPath,
			children: []registry.RegistryKeyChild{
				{Name: "TLS 1.2", Path: schannelProtocolsPath + `\TLS 1.2`},
				{Name: "Client", Path: schannelProtocolsPath + `\TLS 1.2\Client`},
				{Name: "Server", Path: schannelProtocolsPath + `\TLS 1.2\Server`},
				{Name: "SSL 3.0", Path: schannelProtocolsPath + `\SSL 3.0`},
				{Name: "Server", Path: schannelProtocolsPath + `\SSL 3.0\Server`},
			},
			want: []string{"TLS 1.2", "SSL 3.0"},
		},
		{
			name:   "cipher names keep their slashes",
			parent: schannelCiphersPath,
			children: []registry.RegistryKeyChild{
				{Name: "RC4 128/128", Path: schannelCiphersPath + `\RC4 128/128`},
				{Name: "Triple DES 168", Path: schannelCiphersPath + `\Triple DES 168`},
			},
			want: []string{"RC4 128/128", "Triple DES 168"},
		},
		{
			name:   "duplicates are folded case-insensitively, first spelling wins",
			parent: schannelHashesPath,
			children: []registry.RegistryKeyChild{
				{Name: "SHA256", Path: schannelHashesPath + `\SHA256`},
				{Name: "sha256", Path: schannelHashesPath + `\sha256`},
			},
			want: []string{"SHA256"},
		},
		{
			name:     "no children",
			parent:   schannelCiphersPath,
			children: nil,
			want:     []string{},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, schannelChildNames(c.parent, c.children))
		})
	}
}

// TestMergeSchannelNames is the other half of the absent-is-not-disabled
// guarantee: the well-known names are always present, in a fixed order, whether
// or not the registry mentions them, so an audit that selects "TLS 1.0" finds a
// row to assert on instead of matching nothing and passing vacuously.
func TestMergeSchannelNames(t *testing.T) {
	t.Run("well-known names survive an empty registry", func(t *testing.T) {
		got := mergeSchannelNames(schannelWellKnownProtocols, nil)
		assert.Equal(t, schannelWellKnownProtocols, got)
		assert.Contains(t, got, "TLS 1.0")
		assert.Contains(t, got, "Multi-Protocol Unified Hello")
	})

	t.Run("discovered extras are appended and sorted", func(t *testing.T) {
		got := mergeSchannelNames(
			[]string{"MD5", "SHA"},
			[]string{"zzz-custom", "SHA", "aaa-custom"},
		)
		assert.Equal(t, []string{"MD5", "SHA", "aaa-custom", "zzz-custom"}, got)
	})

	t.Run("a differently-cased discovery keeps the well-known spelling", func(t *testing.T) {
		got := mergeSchannelNames([]string{"TLS 1.2"}, []string{"tls 1.2"})
		assert.Equal(t, []string{"TLS 1.2"}, got)
	})

	t.Run("duplicate discoveries are folded", func(t *testing.T) {
		got := mergeSchannelNames([]string{"PKCS"}, []string{"Custom", "custom"})
		assert.Equal(t, []string{"PKCS", "Custom"}, got)
	})

	t.Run("every well-known list is non-empty and unique", func(t *testing.T) {
		for _, list := range [][]string{
			schannelWellKnownProtocols,
			schannelWellKnownCiphers,
			schannelWellKnownHashes,
			schannelWellKnownKeyExchangeAlgorithms,
		} {
			require.NotEmpty(t, list)
			seen := map[string]struct{}{}
			for _, n := range list {
				k := strings.ToLower(n)
				_, dup := seen[k]
				assert.False(t, dup, "duplicate well-known name %q", n)
				seen[k] = struct{}{}
			}
		}
	})
}

func boolPtr(v bool) *bool { return &v }
