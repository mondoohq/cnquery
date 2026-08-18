// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package windows

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers/os/resources/powershell"
)

// The fixtures are the verbatim output of AclScript captured from a Windows
// Server 2022 host: a directory deliberately given a Modify grant to
// BUILTIN\Users and a FullControl deny to ANONYMOUS LOGON, and the stock
// C:\Windows\System32\dns directory.
func loadAcl(t *testing.T, name string) *WindowsAcl {
	t.Helper()
	f, err := os.Open("testdata/" + name)
	require.NoError(t, err)
	defer f.Close()

	res, err := ParseWindowsAcl(f)
	require.NoError(t, err)
	return res
}

func TestParseWindowsAcl(t *testing.T) {
	acl := loadAcl(t, "acl-directory.json")

	assert.Equal(t, `C:\acltest`, acl.Path)
	assert.Equal(t, `BUILTIN\Administrators`, acl.Owner)
	assert.Equal(t, "S-1-5-32-544", acl.OwnerSid)
	assert.Equal(t, `NT AUTHORITY\SYSTEM`, acl.Group)
	assert.False(t, acl.Protected, "the directory still inherits from its parent")
	assert.Contains(t, acl.Sddl, "O:BAG:SY")
	require.Len(t, acl.Access, 7)

	// The deny entry comes first, as Windows orders them.
	deny := acl.Access[0]
	assert.Equal(t, `NT AUTHORITY\ANONYMOUS LOGON`, deny.Identity)
	assert.Equal(t, "S-1-5-7", deny.Sid)
	assert.Equal(t, "Deny", deny.Type)
	assert.False(t, deny.IsAllow())
	assert.True(t, deny.AllowsFullControl(), "the entry covers full control, it just denies it")
}

// The rights label cannot be used for assertions, and this is the evidence.
// Two of the shapes in these fixtures defeat a substring match on it.
func TestWindowsAclRightsLabelIsNotAssertable(t *testing.T) {
	acl := loadAcl(t, "acl-directory.json")
	sys := loadAcl(t, "acl-system-directory.json")

	// A principal granted Modify is a writer, but the label never says Write.
	var users WindowsAclEntry
	for _, e := range acl.Access {
		if e.Identity == `BUILTIN\Users` && e.Rights == "Modify, Synchronize" {
			users = e
			break
		}
	}
	require.NotEmpty(t, users.Identity)
	assert.NotContains(t, users.Rights, "Write")
	assert.True(t, users.AllowsWrite(), "Modify grants write even though its label does not say so")

	// A mask with the generic bits set has no label at all and renders as a
	// bare number, so any name-based match misses it entirely.
	var generic WindowsAclEntry
	for _, e := range sys.Access {
		if e.Mask == AclGenericAll {
			generic = e
			break
		}
	}
	require.NotEmpty(t, generic.Identity)
	assert.Equal(t, "268435456", generic.Rights, "GenericAll has no label")
	assert.True(t, generic.AllowsWrite())
	assert.True(t, generic.AllowsFullControl())
	assert.True(t, generic.AllowsPermissionChange())
}

// The whole point of the resource: naming who can change a path.
func TestAllowedWritePrincipals(t *testing.T) {
	acl := loadAcl(t, "acl-directory.json")

	got := acl.AllowedWritePrincipals()
	// BUILTIN\Users and Authenticated Users both hold Modify here, which is
	// exactly the finding an audit of this directory should produce.
	assert.Contains(t, got, `BUILTIN\Users`)
	assert.Contains(t, got, `NT AUTHORITY\Authenticated Users`)
	assert.Contains(t, got, `NT AUTHORITY\SYSTEM`)
	assert.Contains(t, got, `BUILTIN\Administrators`)
	// The deny entry names a principal that cannot write, and a deny is never
	// a grant, so it must not appear.
	assert.NotContains(t, got, `NT AUTHORITY\ANONYMOUS LOGON`)

	// Authenticated Users holds two allow entries on this directory; the
	// principal is reported once.
	count := 0
	for _, p := range got {
		if p == `NT AUTHORITY\Authenticated Users` {
			count++
		}
	}
	assert.Equal(t, 1, count, "a principal with several write entries is listed once")

	// The stock C:\Windows\System32\dns directory. Note CREATOR OWNER: its
	// entry is GenericAll with InheritOnly, meaning whoever creates a child
	// object owns it outright. It is a real write grant and the resource is
	// right to name it, which is the kind of thing a mode-bit view of Windows
	// permissions cannot express at all.
	sys := loadAcl(t, "acl-system-directory.json")
	assert.Equal(t, []string{
		`NT SERVICE\TrustedInstaller`,
		`NT AUTHORITY\SYSTEM`,
		`BUILTIN\Administrators`,
		`CREATOR OWNER`,
	}, sys.AllowedWritePrincipals())

	// BUILTIN\Users holds two entries on that directory, ReadAndExecute and
	// the generic mask 0xA0000000 (read and execute). Neither is a write, so
	// it must not be named. Widening the write mask by one bit too many would
	// report every readable directory as writable.
	assert.NotContains(t, sys.AllowedWritePrincipals(), `BUILTIN\Users`)
}

func TestWindowsAclEntryPredicates(t *testing.T) {
	tests := []struct {
		name             string
		mask             int64
		read             bool
		write            bool
		execute          bool
		del              bool
		full             bool
		permissionChange bool
	}{
		{
			name: "FullControl", mask: AclFullControl,
			read: true, write: true, execute: true, del: true, full: true, permissionChange: true,
		},
		{
			name: "GenericAll with no label", mask: AclGenericAll,
			read: true, write: true, execute: true, del: true, full: true, permissionChange: true,
		},
		{
			// 1245631 = Modify | Synchronize, the value in the fixture.
			name: "Modify", mask: 1245631,
			read: true, write: true, execute: true, del: true, full: false, permissionChange: false,
		},
		{
			// 1179817 = ReadAndExecute | Synchronize, the value in the fixture.
			name: "ReadAndExecute", mask: 1179817,
			read: true, write: false, execute: true, del: false, full: false, permissionChange: false,
		},
		{
			name: "ReadData only", mask: AclReadData,
			read: true,
		},
		{
			name: "WriteData only", mask: AclWriteData,
			write: true,
		},
		{
			// Appending is a write even though nothing existing is replaced.
			name: "AppendData only", mask: AclAppendData,
			write: true,
		},
		{
			// Changing attributes is a write, and is easy to miss.
			name: "WriteAttributes only", mask: AclWriteAttributes,
			write: true,
		},
		{
			// Delete is a write: it changes what the path resolves to.
			name: "Delete only", mask: AclDelete,
			write: true, del: true,
		},
		{
			// Neither of these says "write" but both let the principal grant
			// itself write, so they must be visible.
			name: "ChangePermissions only", mask: AclChangePermissions,
			permissionChange: true,
		},
		{
			name: "TakeOwnership only", mask: AclTakeOwnership,
			permissionChange: true,
		},
		{
			// Synchronize appears on almost every entry and grants nothing.
			name: "Synchronize only", mask: AclSynchronize,
		},
		{
			name: "no rights at all", mask: 0,
		},
		{
			// 3758161920 = 0xE0010000: the generic read, write and execute
			// bits plus Delete. This is the mask that arrives as a negative
			// int32 and must not have been truncated on the way in.
			name: "generic read write execute with delete", mask: 3758161920,
			read: true, write: true, execute: true, del: true, full: false, permissionChange: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := WindowsAclEntry{Mask: tc.mask}
			assert.Equal(t, tc.read, e.AllowsRead(), "AllowsRead")
			assert.Equal(t, tc.write, e.AllowsWrite(), "AllowsWrite")
			assert.Equal(t, tc.execute, e.AllowsExecute(), "AllowsExecute")
			assert.Equal(t, tc.del, e.AllowsDelete(), "AllowsDelete")
			assert.Equal(t, tc.full, e.AllowsFullControl(), "AllowsFullControl")
			assert.Equal(t, tc.permissionChange, e.AllowsPermissionChange(), "AllowsPermissionChange")
		})
	}
}

// A mask carrying the generic bits is a negative signed 32 bit value on the
// wire. It must arrive widened rather than truncated, or every predicate over
// it reads false and the most privileged entries look like the least.
func TestWindowsAclHighBitMaskSurvives(t *testing.T) {
	acl, err := ParseWindowsAcl(strings.NewReader(`{
		"Path":"C:\\x","Owner":"BUILTIN\\Administrators","Group":"","Sddl":"","Protected":false,"OwnerSid":"",
		"Access":[{"Identity":"NT AUTHORITY\\Authenticated Users","Sid":"S-1-5-11","Type":"Allow",
		"Rights":"-536805376","Mask":3758161920,"Inherited":true,"InheritanceFlags":"ContainerInherit, ObjectInherit","PropagationFlags":"InheritOnly"}]
	}`))
	require.NoError(t, err)
	require.Len(t, acl.Access, 1)

	e := acl.Access[0]
	assert.Equal(t, int64(3758161920), e.Mask)
	assert.Greater(t, e.Mask, int64(0), "the mask must not arrive as a negative int32")
	assert.True(t, e.AllowsWrite())
	assert.Equal(t, []string{`NT AUTHORITY\Authenticated Users`}, acl.AllowedWritePrincipals())
}

// An entry repeats along more dimensions than its principal. The system
// directory fixture contains the case directly: TrustedInstaller appears twice
// as an allow entry, once with FullControl and once with GenericAll. If the id
// missed the mask, the second would be dropped into the first's cache slot and
// report FullControl's rights.
func TestAclEntryIDCarriesEveryDimension(t *testing.T) {
	sys := loadAcl(t, "acl-system-directory.json")

	ids := map[string]int{}
	for _, e := range sys.Access {
		ids[AclEntryID(sys.Path, e.Identity, e.Type, e.Mask, e.Inherited, e.InheritanceFlags, e.PropagationFlags)]++
	}
	assert.Len(t, ids, len(sys.Access), "every entry must get a distinct id")

	base := AclEntryID(`C:\x`, `BUILTIN\Users`, "Allow", AclFullControl, false, "None", "None")
	assert.NotEqual(t, base, AclEntryID(`C:\y`, `BUILTIN\Users`, "Allow", AclFullControl, false, "None", "None"), "path")
	assert.NotEqual(t, base, AclEntryID(`C:\x`, `BUILTIN\Guests`, "Allow", AclFullControl, false, "None", "None"), "identity")
	assert.NotEqual(t, base, AclEntryID(`C:\x`, `BUILTIN\Users`, "Deny", AclFullControl, false, "None", "None"), "allow vs deny")
	assert.NotEqual(t, base, AclEntryID(`C:\x`, `BUILTIN\Users`, "Allow", AclReadData, false, "None", "None"), "mask")
	assert.NotEqual(t, base, AclEntryID(`C:\x`, `BUILTIN\Users`, "Allow", AclFullControl, true, "None", "None"), "inherited")
	assert.NotEqual(t, base, AclEntryID(`C:\x`, `BUILTIN\Users`, "Allow", AclFullControl, false, "ObjectInherit", "None"), "inheritance flags")
	assert.NotEqual(t, base, AclEntryID(`C:\x`, `BUILTIN\Users`, "Allow", AclFullControl, false, "None", "InheritOnly"), "propagation flags")
}

// A path is interpolated into the script, so it has to survive being a Windows
// path and must not be able to close the string it sits in.
func TestAclScriptQuotesThePath(t *testing.T) {
	script := AclScript(`C:\Program Files\App`)
	assert.Contains(t, script, `$p='C:\Program Files\App'`)
	// A backslash keeps its literal meaning inside a single quoted string, so
	// nothing about the path is re-interpreted.
	assert.NotContains(t, script, `C:\\Program`)

	// A quote in the path is doubled rather than ending the string.
	quoted := AclScript(`C:\it's`)
	assert.Contains(t, quoted, `$p='C:\it''s'`)

	// A path that tries to close the string and append a command stays one
	// string literal.
	injected := AclScript(`C:\x'; Remove-Item C:\ -Recurse; '`)
	assert.NotContains(t, injected, `$p='C:\x'; Remove-Item`)
	assert.Contains(t, injected, `''; Remove-Item`)

	// The script has to fit on a Windows command line once encoded.
	assert.Less(t, len(powershell.Encode(AclScript(strings.Repeat("a", 260)))), 8191)
}

// A directory with no entries at all is not the same as a directory anyone can
// write, and an empty list must not be produced from a failed read. The
// resource errors on a failed read; this covers the shape only.
func TestParseWindowsAclEmptyAccess(t *testing.T) {
	acl, err := ParseWindowsAcl(strings.NewReader(
		`{"Path":"C:\\x","Owner":"BUILTIN\\Administrators","Group":"","Sddl":"","Protected":true,"OwnerSid":"S-1-5-32-544","Access":[]}`))
	require.NoError(t, err)

	assert.Empty(t, acl.Access)
	assert.Empty(t, acl.AllowedWritePrincipals())
	assert.True(t, acl.Protected)
}

// An entry whose account cannot be translated to a SID, an orphaned entry left
// behind by a deleted account, is still reported with its raw identity rather
// than being dropped from the list.
func TestParseWindowsAclUnresolvableIdentity(t *testing.T) {
	acl, err := ParseWindowsAcl(strings.NewReader(`{
		"Path":"C:\\x","Owner":"BUILTIN\\Administrators","Group":"","Sddl":"","Protected":false,"OwnerSid":"",
		"Access":[
		{"Identity":"S-1-5-21-1111111111-2222222222-3333333333-1013","Sid":"","Type":"Allow","Rights":"FullControl","Mask":2032127,"Inherited":false,"InheritanceFlags":"None","PropagationFlags":"None"},
		{"Identity":"BUILTIN\\Administrators","Sid":"S-1-5-32-544","Type":"Allow","Rights":"FullControl","Mask":2032127,"Inherited":false,"InheritanceFlags":"None","PropagationFlags":"None"}]
	}`))
	require.NoError(t, err)

	require.Len(t, acl.Access, 2, "an unresolvable account must not shorten the list")
	assert.Empty(t, acl.Access[0].Sid)
	assert.Equal(t, "S-1-5-21-1111111111-2222222222-3333333333-1013", acl.Access[0].Identity)
	// It still counts as a writer, so an audit sees the orphaned grant.
	assert.Contains(t, acl.AllowedWritePrincipals(), "S-1-5-21-1111111111-2222222222-3333333333-1013")
}
