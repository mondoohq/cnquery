// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"bytes"
	"errors"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v12/providers/os/connection/shared"
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
	conn := newFakeConnection(nil)

	require.NoError(t, conn.fs.MkdirAll("/etc/firewalld/zones", 0o755))
	require.NoError(t, afero.WriteFile(conn.fs, "/etc/firewalld/firewalld.conf", []byte(`# firewalld config file
DefaultZone=public
`), 0o644))

	publicZone := `<?xml version="1.0" encoding="utf-8"?>
<zone target="default">
  <interface name="eth0"/>
  <service name="ssh"/>
  <service name="dhcpv6-client"/>
  <port protocol="tcp" port="8080"/>
  <rule family="ipv4">
    <source address="127.0.0.1"/>
    <destination invert="yes" address="127.0.0.1"/>
    <drop/>
  </rule>
</zone>
`

	trustedZone := `<?xml version="1.0" encoding="utf-8"?>
<zone target="ACCEPT">
  <masquerade/>
  <source address="192.168.1.0/24"/>
  <icmp-block name="echo-request"/>
</zone>
`

	require.NoError(t, afero.WriteFile(conn.fs, "/etc/firewalld/zones/public.xml", []byte(publicZone), 0o644))
	require.NoError(t, afero.WriteFile(conn.fs, "/etc/firewalld/zones/trusted.xml", []byte(trustedZone), 0o644))

	runtime := plugin.NewRuntime(conn, nil, false, CreateResource, NewResource, GetData, SetData, nil)

	_, err := loadFirewalldFromConfig(runtime)
	require.NoError(t, err)

	res, err := CreateResource(runtime, "firewalld", map[string]*llx.RawData{})
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
	conn := newFakeConnection(map[string]commandResult{
		"firewall-cmd --get-default-zone": {stdout: "public\n", exitStatus: 0},
		"firewall-cmd --get-active-zones": {stdout: "public\n  interfaces: eth0\n", exitStatus: 0},
		"firewall-cmd --get-zones":        {stdout: "public trusted\n", exitStatus: 0},
		"firewall-cmd --zone=public --list-all": {
			stdout: `public (active)
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
			exitStatus: 0,
		},
		"firewall-cmd --zone=public --list-rich-rules": {
			stdout: `rule family="ipv4" source address="127.0.0.1" destination not address="127.0.0.1" drop
rule family="ipv6" source address="::1" destination not address="::1" drop
`,
			exitStatus: 0,
		},
		"firewall-cmd --zone=trusted --list-all": {
			stdout: `trusted
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
			exitStatus: 0,
		},
		"firewall-cmd --zone=trusted --list-rich-rules": {stdout: "\n", exitStatus: 0},
	})

	runtime := plugin.NewRuntime(conn, nil, false, CreateResource, NewResource, GetData, SetData, nil)

	res, err := CreateResource(runtime, "firewalld", map[string]*llx.RawData{})
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

type commandResult struct {
	stdout     string
	stderr     string
	exitStatus int
}

type fakeConnection struct {
	id       uint32
	asset    *inventory.Asset
	commands map[string]commandResult
	fs       afero.Fs
}

func newFakeConnection(commands map[string]commandResult) *fakeConnection {
	return &fakeConnection{
		id:       1,
		asset:    &inventory.Asset{},
		commands: commands,
		fs:       afero.NewMemMapFs(),
	}
}

func (c *fakeConnection) ID() uint32 {
	return c.id
}

func (c *fakeConnection) ParentID() uint32 {
	return 0
}

func (c *fakeConnection) RunCommand(command string) (*shared.Command, error) {
	res, ok := c.commands[command]
	if !ok {
		return &shared.Command{
			Command:    command,
			Stdout:     bytes.NewBuffer(nil),
			Stderr:     bytes.NewBufferString("command not found"),
			ExitStatus: 127,
		}, nil
	}

	return &shared.Command{
		Command:    command,
		Stdout:     bytes.NewBufferString(res.stdout),
		Stderr:     bytes.NewBufferString(res.stderr),
		ExitStatus: res.exitStatus,
	}, nil
}

func (c *fakeConnection) FileInfo(string) (shared.FileInfoDetails, error) {
	return shared.FileInfoDetails{}, errors.New("not implemented")
}

func (c *fakeConnection) FileSystem() afero.Fs {
	return c.fs
}

func (c *fakeConnection) Name() string {
	return "fake"
}

func (c *fakeConnection) Type() shared.ConnectionType {
	return shared.Type_Local
}

func (c *fakeConnection) Asset() *inventory.Asset {
	return c.asset
}

func (c *fakeConnection) UpdateAsset(asset *inventory.Asset) {
	c.asset = asset
}

func (c *fakeConnection) Capabilities() shared.Capabilities {
	return shared.Capability_RunCommand
}
