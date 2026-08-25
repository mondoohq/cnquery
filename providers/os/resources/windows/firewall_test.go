// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package windows

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWindowsFirewallSettings(t *testing.T) {
	r, err := os.Open("./testdata/firewall-settings.json")
	require.NoError(t, err)

	settings, err := ParseWindowsFirewallSettings(r)
	assert.Nil(t, err)
	assert.Equal(t, int64(65535), settings.ActiveProfile)
}

func TestWindowsFirewallProfiles(t *testing.T) {
	r, err := os.Open("./testdata/firewall-profiles.json")
	require.NoError(t, err)

	items, err := ParseWindowsFirewallProfiles(r)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(items))
}

func TestWindowsFirewallRules(t *testing.T) {
	r, err := os.Open("./testdata/firewall-rules.json")
	require.NoError(t, err)

	items, err := ParseWindowsFirewallRules(r)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(items))
}

func TestFirewallRuleFiltersScriptFitsCommandLine(t *testing.T) {
	// Encode widens the script to UTF-16 and base64 encodes it, roughly
	// tripling its length against a command-line cap that depends on the
	// transport. Over the cap the script is rejected before PowerShell runs
	// and the non-zero exit reads as if the cmdlets were missing.
	assert.LessOrEqual(t, len(FIREWALL_RULE_FILTERS), PSMaxScriptLength)
	assert.LessOrEqual(t, len(FIREWALL_RULES), PSMaxScriptLength)
}

// stockFilterPayload is shaped like the combined filter payload a Windows
// host returns: a rule whose ports are Any, a rule with a real port range
// and a program path, an ICMP rule, and a rule that appears in the port
// collection but in no application, service or security collection.
const stockFilterPayload = `{
  "Port": [
    {"InstanceID":"CoreNet-DHCP-In","Protocol":"UDP","LocalPort":["68"],"RemotePort":["67"],"IcmpType":["Any"]},
    {"InstanceID":"RemoteDesktop-UserMode-In-TCP","Protocol":"TCP","LocalPort":["3389"],"RemotePort":{"value":["Any"],"Count":1},"IcmpType":null},
    {"InstanceID":"WINRM-HTTP-In-TCP","Protocol":"TCP","LocalPort":{"value":["5985","49152-65535"],"Count":2},"RemotePort":["Any"],"IcmpType":null},
    {"InstanceID":"CoreNet-ICMP4-DU-In","Protocol":"ICMPv4","LocalPort":["Any"],"RemotePort":["Any"],"IcmpType":["3:4"]},
    {"InstanceID":"Orphan-Port-Only","Protocol":"47","LocalPort":["Any"],"RemotePort":["Any"],"IcmpType":["Any"]}
  ],
  "Address": [
    {"InstanceID":"CoreNet-DHCP-In","LocalAddress":["Any"],"RemoteAddress":["Any"]},
    {"InstanceID":"RemoteDesktop-UserMode-In-TCP","LocalAddress":["Any"],"RemoteAddress":{"value":["LocalSubnet","10.0.0.0/8"],"Count":2}}
  ],
  "Application": [
    {"InstanceID":"CoreNet-DHCP-In","Program":"%SystemRoot%\\system32\\svchost.exe"},
    {"InstanceID":"RemoteDesktop-UserMode-In-TCP","Program":"Any"},
    {"InstanceID":"WINRM-HTTP-In-TCP","Program":"System"}
  ],
  "Service": [
    {"InstanceID":"CoreNet-DHCP-In","Service":"dhcp"},
    {"InstanceID":"RemoteDesktop-UserMode-In-TCP","Service":"termservice"}
  ],
  "InterfaceType": [
    {"InstanceID":"CoreNet-DHCP-In","InterfaceType":"Any"},
    {"InstanceID":"RemoteDesktop-UserMode-In-TCP","InterfaceType":"Wired, Wireless"}
  ],
  "Security": [
    {"InstanceID":"CoreNet-DHCP-In","RemoteUser":"Any","RemoteMachine":"Any"},
    {"InstanceID":"RemoteDesktop-UserMode-In-TCP","RemoteUser":"D:(A;;CC;;;S-1-5-21-1-2-3-1001)","RemoteMachine":"Any"}
  ]
}`

func TestParseWindowsFirewallRuleFiltersJoinsOnInstanceID(t *testing.T) {
	filters, err := ParseWindowsFirewallRuleFilters(strings.NewReader(stockFilterPayload))
	require.NoError(t, err)
	require.Len(t, filters, 5)

	dhcp := filters["CoreNet-DHCP-In"]
	require.NotNil(t, dhcp)
	require.NotNil(t, dhcp.Port)
	assert.Equal(t, PSFlexString("UDP"), dhcp.Port.Protocol)
	assert.Equal(t, PSStringArray{"68"}, dhcp.Port.LocalPort)
	require.NotNil(t, dhcp.Application)
	assert.Equal(t, PSFlexString(`%SystemRoot%\system32\svchost.exe`), dhcp.Application.Program)
	require.NotNil(t, dhcp.Service)
	assert.Equal(t, PSFlexString("dhcp"), dhcp.Service.Service)
	require.NotNil(t, dhcp.InterfaceType)
	assert.Equal(t, PSFlagList{"Any"}, dhcp.InterfaceType.InterfaceType)

	// Every collection joins onto the same rule, and none of them bleeds
	// into a neighbouring rule.
	rdp := filters["RemoteDesktop-UserMode-In-TCP"]
	require.NotNil(t, rdp)
	assert.Equal(t, PSStringArray{"3389"}, rdp.Port.LocalPort)
	assert.Equal(t, PSStringArray{"LocalSubnet", "10.0.0.0/8"}, rdp.Address.RemoteAddress)
	assert.Equal(t, PSFlexString("Any"), rdp.Application.Program)
	assert.Equal(t, PSFlagList{"Wired", "Wireless"}, rdp.InterfaceType.InterfaceType)
	assert.Equal(t, PSFlexString("D:(A;;CC;;;S-1-5-21-1-2-3-1001)"), rdp.Security.RemoteUser)
}

func TestParseWindowsFirewallRuleFiltersBothListShapes(t *testing.T) {
	filters, err := ParseWindowsFirewallRuleFilters(strings.NewReader(stockFilterPayload))
	require.NoError(t, err)

	// A bare JSON array and the {"value":[...],"Count":n} wrapper carry the
	// same meaning. A plain []string tag decodes the second to empty, which
	// reports "no ports" on a rule that listens on two.
	winrm := filters["WINRM-HTTP-In-TCP"]
	require.NotNil(t, winrm)
	assert.Equal(t, PSStringArray{"5985", "49152-65535"}, winrm.Port.LocalPort)
	assert.Equal(t, PSStringArray{"Any"}, winrm.Port.RemotePort)

	rdp := filters["RemoteDesktop-UserMode-In-TCP"]
	assert.Equal(t, PSStringArray{"Any"}, rdp.Port.RemotePort)
	assert.Equal(t, PSStringArray{"LocalSubnet", "10.0.0.0/8"}, rdp.Address.RemoteAddress)
}

func TestParseWindowsFirewallRuleFiltersAbsentFilters(t *testing.T) {
	filters, err := ParseWindowsFirewallRuleFilters(strings.NewReader(stockFilterPayload))
	require.NoError(t, err)

	// A rule that appears in the port collection and nowhere else must not
	// report an empty program, service or SDDL as if it were a fact about
	// the rule. Nil is what the resource layer turns into a null field.
	orphan := filters["Orphan-Port-Only"]
	require.NotNil(t, orphan)
	require.NotNil(t, orphan.Port)
	assert.Nil(t, orphan.Application)
	assert.Nil(t, orphan.Service)
	assert.Nil(t, orphan.Security)
	assert.Nil(t, orphan.Address)
	assert.Nil(t, orphan.InterfaceType)

	// An ICMP rule carries no application filter either, and its ICMP types
	// are read while its neighbours' stay empty.
	icmp := filters["CoreNet-ICMP4-DU-In"]
	require.NotNil(t, icmp)
	assert.Equal(t, PSStringArray{"3:4"}, icmp.Port.IcmpType)
	assert.Equal(t, PSStringArray{"Any"}, icmp.Port.LocalPort)
	assert.Nil(t, icmp.Application)

	// A rule absent from every collection is absent from the join, so the
	// resource layer reports every condition as null.
	assert.Nil(t, filters["Never-Filtered"])
}

func TestParseWindowsFirewallRuleFiltersEmptyCollections(t *testing.T) {
	// Windows PowerShell 5.1 serializes an empty collection assigned into a
	// PSCustomObject as "" rather than as [].
	filters, err := ParseWindowsFirewallRuleFilters(strings.NewReader(
		`{"Port":"","Address":"","Application":"","Service":"","InterfaceType":"","Security":""}`))
	require.NoError(t, err)
	assert.Empty(t, filters)

	// An empty payload is not an error: it means the host returned nothing.
	filters, err = ParseWindowsFirewallRuleFilters(strings.NewReader(""))
	require.NoError(t, err)
	assert.Empty(t, filters)
}

func TestParseWindowsFirewallRuleFiltersSingleElementFlattens(t *testing.T) {
	// A one-element collection can flatten to a bare object.
	filters, err := ParseWindowsFirewallRuleFilters(strings.NewReader(
		`{"Port":{"InstanceID":"Solo-In","Protocol":"TCP","LocalPort":"445","RemotePort":["Any"]},
		  "Application":{"InstanceID":"Solo-In","Program":"Any"}}`))
	require.NoError(t, err)
	require.Len(t, filters, 1)

	solo := filters["Solo-In"]
	require.NotNil(t, solo)
	// A single port can arrive unwrapped from its array too.
	assert.Equal(t, PSStringArray{"445"}, solo.Port.LocalPort)
	assert.Equal(t, PSFlexString("Any"), solo.Application.Program)
	assert.Nil(t, solo.Security)
}

func TestPSFlexString(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   string
		want PSFlexString
	}{
		{"string", `"TCP"`, "TCP"},
		{"empty string", `""`, ""},
		{"null", `null`, ""},
		// A calculated property yielding nothing serializes as {}, which
		// fails a plain string tag and takes the whole payload with it.
		{"empty object", `{}`, ""},
		// A protocol with no well-known name keeps its number rather than
		// decoding to an empty string.
		{"number", `47`, "47"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got PSFlexString
			require.NoError(t, json.Unmarshal([]byte(tc.in), &got))
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestPSFlagList(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   string
		want PSFlagList
	}{
		{"single", `"Any"`, PSFlagList{"Any"}},
		{"several", `"Domain, Private"`, PSFlagList{"Domain", "Private"}},
		{"no spaces", `"Domain,Public"`, PSFlagList{"Domain", "Public"}},
		{"empty", `""`, nil},
		{"null", `null`, nil},
		// A bit mask is reported as its literal text: an empty list would
		// read as "this rule applies to no profile at all".
		{"number", `4`, PSFlagList{"4"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got PSFlagList
			require.NoError(t, json.Unmarshal([]byte(tc.in), &got))
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestParseWindowsFirewallRulesProfiles(t *testing.T) {
	rules, err := ParseWindowsFirewallRules(strings.NewReader(
		`[{"InstanceID":"A","Name":"A","Profiles":"Domain, Private"},
		  {"InstanceID":"B","Name":"B","Profiles":"Any"},
		  {"InstanceID":"C","Name":"C"}]`))
	require.NoError(t, err)
	require.Len(t, rules, 3)
	assert.Equal(t, PSFlagList{"Domain", "Private"}, rules[0].Profiles)
	assert.Equal(t, PSFlagList{"Any"}, rules[1].Profiles)
	// An absent value stays empty so the resource layer can report null
	// rather than claiming the rule applies to no profile.
	assert.Nil(t, rules[2].Profiles)
}
