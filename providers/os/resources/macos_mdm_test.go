// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// macos.mdm — parseMdmEnrollment
// =============================================================================

func TestParseMdmEnrollment_NotEnrolled(t *testing.T) {
	out := `Enrolled via DEP: No
MDM enrollment: No
`
	s := parseMdmEnrollment(out)
	assert.False(t, s.enrolled)
	assert.False(t, s.dep)
	assert.False(t, s.userApproved)
	assert.Equal(t, "", s.serverUrl)
}

func TestParseMdmEnrollment_DepUserApproved(t *testing.T) {
	out := `Enrolled via DEP: Yes
MDM enrollment: Yes (User Approved)
MDM server: https://mdm.example.com/MDMServiceConfig?id=ABC-123
`
	s := parseMdmEnrollment(out)
	assert.True(t, s.enrolled)
	assert.True(t, s.dep)
	assert.True(t, s.userApproved)
	assert.Equal(t, "https://mdm.example.com/MDMServiceConfig?id=ABC-123", s.serverUrl)
}

func TestParseMdmEnrollment_ManualEnrollment(t *testing.T) {
	// User-initiated enrollment without DEP, not (yet) user-approved.
	out := `Enrolled via DEP: No
MDM enrollment: Yes
MDM server: https://mdm.example.com/enrollment
`
	s := parseMdmEnrollment(out)
	assert.True(t, s.enrolled)
	assert.False(t, s.dep)
	assert.False(t, s.userApproved)
	assert.Equal(t, "https://mdm.example.com/enrollment", s.serverUrl)
}

func TestParseMdmEnrollment_CaseInsensitive(t *testing.T) {
	// Some macOS versions use slightly different capitalization;
	// parsing must not care.
	out := `enrolled via dep: Yes
MDM Enrollment: Yes (User Approved)
mdm server: https://example.com
`
	s := parseMdmEnrollment(out)
	assert.True(t, s.enrolled)
	assert.True(t, s.dep)
	assert.True(t, s.userApproved)
	assert.Equal(t, "https://example.com", s.serverUrl)
}

func TestParseMdmEnrollment_ServerUrlWithColon(t *testing.T) {
	// The server URL contains a colon (the `https://`), so the parser
	// must split on the FIRST colon only.
	out := `Enrolled via DEP: Yes
MDM enrollment: Yes
MDM server: https://mdm.example.com:8443/path
`
	s := parseMdmEnrollment(out)
	assert.Equal(t, "https://mdm.example.com:8443/path", s.serverUrl)
}

func TestParseMdmEnrollment_GarbageLinesIgnored(t *testing.T) {
	out := `
Some banner line that has no colon
Enrolled via DEP: No

MDM enrollment: No
`
	s := parseMdmEnrollment(out)
	assert.False(t, s.enrolled)
	assert.False(t, s.dep)
}

// =============================================================================
// macos.profiles — parseMacosProfilesXML
// =============================================================================

const profilesXMLBoth = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>_computerlevel</key>
    <array>
        <dict>
            <key>ProfileIdentifier</key>
            <string>com.example.firewall</string>
            <key>ProfileUUID</key>
            <string>11111111-1111-1111-1111-111111111111</string>
            <key>ProfileDisplayName</key>
            <string>Example Firewall</string>
            <key>ProfileDescription</key>
            <string>Locks down the firewall</string>
            <key>ProfileOrganization</key>
            <string>Example Inc.</string>
            <key>ProfileType</key>
            <string>Configuration</string>
            <key>ProfileRemovalDisallowed</key>
            <string>true</string>
            <key>ProfileItems</key>
            <array>
                <dict>
                    <key>PayloadUUID</key>
                    <string>22222222-2222-2222-2222-222222222222</string>
                    <key>PayloadType</key>
                    <string>com.apple.security.firewall</string>
                    <key>PayloadIdentifier</key>
                    <string>com.example.firewall.payload</string>
                    <key>PayloadDisplayName</key>
                    <string>Firewall Settings</string>
                    <key>PayloadContent</key>
                    <dict>
                        <key>EnableFirewall</key>
                        <true/>
                        <key>BlockAllIncoming</key>
                        <true/>
                    </dict>
                </dict>
                <dict>
                    <key>PayloadUUID</key>
                    <string>33333333-3333-3333-3333-333333333333</string>
                    <key>PayloadType</key>
                    <string>com.apple.MCXFileVault2</string>
                    <key>PayloadIdentifier</key>
                    <string>com.example.fv.payload</string>
                    <key>PayloadDisplayName</key>
                    <string>FileVault Policy</string>
                </dict>
            </array>
        </dict>
    </array>
    <key>501</key>
    <array>
        <dict>
            <key>ProfileIdentifier</key>
            <string>com.example.user.wifi</string>
            <key>ProfileUUID</key>
            <string>44444444-4444-4444-4444-444444444444</string>
            <key>ProfileDisplayName</key>
            <string>Corporate Wi-Fi</string>
            <key>ProfileType</key>
            <string>Configuration</string>
            <key>ProfileItems</key>
            <array/>
        </dict>
    </array>
</dict>
</plist>
`

func TestParseMacosProfilesXML_BothScopes(t *testing.T) {
	parsed, err := parseMacosProfilesXML([]byte(profilesXMLBoth))
	require.NoError(t, err)
	require.Len(t, parsed, 2)

	// Order between scopes isn't deterministic (map iteration); index
	// by identifier instead.
	byID := map[string]parsedMacosProfile{}
	for _, p := range parsed {
		byID[p.identifier] = p
	}

	sys := byID["com.example.firewall"]
	require.NotEmpty(t, sys.identifier, "system profile missing")
	assert.Equal(t, "system", sys.scope)
	assert.Equal(t, "11111111-1111-1111-1111-111111111111", sys.uuid)
	assert.Equal(t, "Example Firewall", sys.displayName)
	assert.Equal(t, "Locks down the firewall", sys.description)
	assert.Equal(t, "Example Inc.", sys.organization)
	assert.Equal(t, "Configuration", sys.profileType)
	assert.True(t, sys.removalDisallowed, "string 'true' must parse as bool true")
	require.Len(t, sys.payloads, 2)

	// First payload — firewall settings with parsed PayloadContent.
	fw := sys.payloads[0]
	assert.Equal(t, "22222222-2222-2222-2222-222222222222", fw.uuid)
	assert.Equal(t, "com.apple.security.firewall", fw.payloadType)
	assert.Equal(t, "com.example.firewall.payload", fw.identifier)
	assert.Equal(t, "Firewall Settings", fw.displayName)
	assert.Equal(t, true, fw.content["EnableFirewall"])
	assert.Equal(t, true, fw.content["BlockAllIncoming"])

	// Second payload — no PayloadContent at all. Must default to empty map,
	// not nil, so downstream `.content["foo"]` queries don't NPE.
	fv := sys.payloads[1]
	assert.Equal(t, "com.apple.MCXFileVault2", fv.payloadType)
	assert.NotNil(t, fv.content)
	assert.Empty(t, fv.content)

	// User-scope profile (uid 501).
	usr := byID["com.example.user.wifi"]
	require.NotEmpty(t, usr.identifier, "user profile missing")
	assert.Equal(t, "user", usr.scope)
	assert.Equal(t, "Corporate Wi-Fi", usr.displayName)
	assert.Empty(t, usr.payloads, "empty <array/> for ProfileItems yields no payloads")
}

func TestParseMacosProfilesXML_Empty(t *testing.T) {
	parsed, err := parseMacosProfilesXML([]byte(""))
	require.NoError(t, err)
	assert.Empty(t, parsed)
}

func TestParseMacosProfilesXML_NoProfiles(t *testing.T) {
	// macOS returns the top-level dict with empty arrays when no
	// profiles are installed.
	in := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>_computerlevel</key>
    <array/>
</dict>
</plist>
`
	parsed, err := parseMacosProfilesXML([]byte(in))
	require.NoError(t, err)
	assert.Empty(t, parsed)
}

func TestParseMacosProfilesXML_BoolRemovalDisallowed(t *testing.T) {
	// Older versions of `profiles` emit the value as a real plist bool
	// (<true/>) rather than a string. Both forms must parse.
	in := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>_computerlevel</key>
    <array>
        <dict>
            <key>ProfileIdentifier</key>
            <string>com.example.bool</string>
            <key>ProfileUUID</key>
            <string>55555555-5555-5555-5555-555555555555</string>
            <key>ProfileRemovalDisallowed</key>
            <true/>
        </dict>
    </array>
</dict>
</plist>
`
	parsed, err := parseMacosProfilesXML([]byte(in))
	require.NoError(t, err)
	require.Len(t, parsed, 1)
	assert.True(t, parsed[0].removalDisallowed, "plist <true/> must parse as bool true")
}

func TestStringFromDict(t *testing.T) {
	d := map[string]any{
		"a": "hello",
		"b": 42,
		"c": nil,
	}
	assert.Equal(t, "hello", stringFromDict(d, "a"))
	assert.Equal(t, "", stringFromDict(d, "b"), "non-string values yield empty string")
	assert.Equal(t, "", stringFromDict(d, "c"))
	assert.Equal(t, "", stringFromDict(d, "missing"))
}
