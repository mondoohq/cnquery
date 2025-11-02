// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v12/providers/os/connection/mock"
)

func TestParseRichRule(t *testing.T) {
	input := `rule family="ipv4" source address="127.0.0.1" destination not address="127.0.0.1" drop`

	rule := parseRichRule(input)

	require.Equal(t, "ipv4", rule.Family)
	require.Equal(t, "drop", rule.Action)
	require.Equal(t, "127.0.0.1", rule.Source.Address)
	require.Equal(t, "", rule.Source.Not.Address)
	require.True(t, rule.Dest.HasNot)
	require.Equal(t, "127.0.0.1", rule.Dest.Not.Address)
}

func TestFirewalldZonesFromConfig(t *testing.T) {

	publicZone, err := os.ReadFile("./firewalld/testdata/public.xml")
	require.NoError(t, err)
	trustedZone, err := os.ReadFile("./firewalld/testdata/trusted.xml")
	require.NoError(t, err)

	conn, err := mock.New(0, &inventory.Asset{}, mock.WithData(&mock.TomlData{
		Files: map[string]*mock.MockFileData{
			"/etc/firewalld/firewalld.conf": {
				Data: []byte(`# firewalld config file
DefaultZone=public
`),
				StatData: mock.FileInfo{
					Mode: 0o644,
				},
			},
			"/etc/firewalld/zones": {
				StatData: mock.FileInfo{
					Mode:  0o755,
					IsDir: true,
				},
			},
			"/etc/firewalld/zones/public.xml": {
				Data: publicZone,
				StatData: mock.FileInfo{
					Mode: 0o644,
				},
			},
			"/etc/firewalld/zones/trusted.xml": {
				Data: trustedZone,
				StatData: mock.FileInfo{
					Mode: 0o644,
				},
			},
		},
	}))
	require.NoError(t, err)

	runtime := plugin.NewRuntime(conn, nil, false, CreateResource, NewResource, GetData, SetData, nil)

	_, err = loadFirewalldFromConfig(runtime)
	require.NoError(t, err)

	res, err := CreateResource(runtime, ResourceFirewalld, map[string]*llx.RawData{})
	require.NoError(t, err)

	firewalld := res.(*mqlFirewalld)
	zones, err := firewalld.zones()
	require.NoError(t, err)
	require.Len(t, zones, 2)

	require.Equal(t, plugin.StateIsSet, firewalld.DefaultZone.State)
	require.Equal(t, "public", firewalld.DefaultZone.Data)
	require.Equal(t, plugin.StateIsSet, firewalld.ActiveZones.State)
	require.ElementsMatch(t, []any{"public", "trusted"}, firewalld.ActiveZones.Data)

	public := findZoneByName(t, zones, "public")
	require.True(t, public.Active.Data)
	require.Equal(t, []any{"eth0"}, public.Interfaces.Data)
	require.Len(t, public.RichRules.Data, 1)

	rule := public.RichRules.Data[0].(*mqlFirewalldRule)
	require.Equal(t, "ipv4", rule.Family.Data)
	require.Equal(t, "drop", rule.Action.Data)
	require.NotNil(t, rule.Destination.Data)
	require.NotNil(t, rule.Destination.Data.Not.Data)
	require.Equal(t, "127.0.0.1", rule.Destination.Data.Not.Data.Address.Data)

	trusted := findZoneByName(t, zones, "trusted")
	require.True(t, trusted.Active.Data)
	require.True(t, trusted.Masquerade.Data)
	require.Equal(t, []any{"192.168.1.0/24"}, trusted.Sources.Data)
	require.Equal(t, []any{"echo-request"}, trusted.IcmpBlocks.Data)
}

func TestFirewalldZones(t *testing.T) {
	conn, err := mock.New(0, &inventory.Asset{}, mock.WithData(&mock.TomlData{
		Commands: map[string]*mock.Command{
			"firewall-cmd --get-default-zone": {Stdout: "public\n", ExitStatus: 0},
			"firewall-cmd --get-active-zones": {Stdout: "public\n  interfaces: eth0\n", ExitStatus: 0},
			"firewall-cmd --get-zones":        {Stdout: "public trusted\n", ExitStatus: 0},
			"firewall-cmd --zone=public --list-all": {
				Stdout: `public (active)
  target: default
  icmp-block-inversion: no
  interfaces: eth0
  sources:
  services: ssh dhcpv6-client
  ports: 8080/tcp
  protocols:
  masquerade: no
  forward-ports:
  source-ports:
  icmp-blocks:
  rich rules:
`,
				ExitStatus: 0,
			},
			"firewall-cmd --zone=public --list-rich-rules": {
				Stdout: `rule family="ipv4" source address="127.0.0.1" destination not address="127.0.0.1" drop
rule family="ipv6" source address="::1" destination not address="::1" drop
`,
				ExitStatus: 0,
			},
			"firewall-cmd --zone=trusted --list-all": {
				Stdout: `trusted
  target: ACCEPT
  icmp-block-inversion: no
  interfaces:
  sources: 192.168.1.0/24
  services:
  ports:
  protocols:
  masquerade: yes
  forward-ports:
  source-ports:
  icmp-blocks: echo-request
  rich rules:
`,
				ExitStatus: 0,
			},
			"firewall-cmd --zone=trusted --list-rich-rules": {Stdout: "\n", ExitStatus: 0},
		},
	}))
	require.NoError(t, err)

	runtime := plugin.NewRuntime(conn, nil, false, CreateResource, NewResource, GetData, SetData, nil)

	res, err := CreateResource(runtime, ResourceFirewalld, map[string]*llx.RawData{})
	require.NoError(t, err)

	firewalld := res.(*mqlFirewalld)
	zones, err := firewalld.zones()
	require.NoError(t, err)
	require.Len(t, zones, 2)

	require.Equal(t, plugin.StateIsSet, firewalld.DefaultZone.State)
	require.Equal(t, "public", firewalld.DefaultZone.Data)
	require.Equal(t, plugin.StateIsSet, firewalld.ActiveZones.State)
	require.Equal(t, []any{"public"}, firewalld.ActiveZones.Data)

	publicZone := findZoneByName(t, zones, "public")
	require.True(t, publicZone.Active.Data)
	require.Equal(t, []any{"eth0"}, publicZone.Interfaces.Data)
	require.Len(t, publicZone.RichRules.Data, 2)

	rule := publicZone.RichRules.Data[0].(*mqlFirewalldRule)
	require.Equal(t, "ipv4", rule.Family.Data)
	require.Equal(t, "drop", rule.Action.Data)

	source := rule.Source.Data
	require.NotNil(t, source)
	require.Equal(t, "127.0.0.1", source.Address.Data)

	destination := rule.Destination.Data
	require.NotNil(t, destination)
	require.NotNil(t, destination.Not.Data)
	require.Equal(t, "127.0.0.1", destination.Not.Data.Address.Data)

	trustedZone := findZoneByName(t, zones, "trusted")
	require.False(t, trustedZone.Active.Data)
	require.True(t, trustedZone.Masquerade.Data)
	require.Equal(t, []any{"192.168.1.0/24"}, trustedZone.Sources.Data)
	require.Empty(t, trustedZone.RichRules.Data)
}

func findZoneByName(t *testing.T, zones []any, name string) *mqlFirewalldZone {
	t.Helper()
	for _, z := range zones {
		mqlZone := z.(*mqlFirewalldZone)
		if mqlZone.Name.Data == name {
			return mqlZone
		}
	}
	t.Fatalf("zone %q not found", name)
	return nil
}
