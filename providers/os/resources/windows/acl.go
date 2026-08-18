// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package windows

import (
	"encoding/json"
	"io"
	"strconv"
	"strings"
)

// AclScript builds the PowerShell that reads a filesystem object's
// discretionary access control list.
//
// Two things are done here rather than left to the caller. The identity of
// each entry is translated to a SID inside a try/catch, so an entry naming a
// deleted account is still reported with its raw identity instead of taking
// the whole list down. And the access mask is widened through BitConverter
// rather than cast: FileSystemRights is a signed 32 bit enum, so a mask with
// the generic bits set (0xE0010000, say) is a negative number that
// [uint32] refuses outright with "Value was either too large or too small".
func AclScript(path string) string {
	return `$ErrorActionPreference='Stop'
$p=` + quotePowerShellString(path) + `
$a=Get-Acl -LiteralPath $p
[ordered]@{
Path=$p;Owner=[string]$a.Owner;Group=[string]$a.Group;Sddl=[string]$a.Sddl;Protected=$a.AreAccessRulesProtected
OwnerSid=$(try{[string]$a.GetOwner([System.Security.Principal.SecurityIdentifier]).Value}catch{''})
Access=@($a.Access|ForEach-Object{
$s='';try{$s=[string]$_.IdentityReference.Translate([System.Security.Principal.SecurityIdentifier]).Value}catch{}
[ordered]@{Identity=[string]$_.IdentityReference;Sid=$s;Type=[string]$_.AccessControlType;Rights=[string]$_.FileSystemRights;Mask=[int64][System.BitConverter]::ToUInt32([System.BitConverter]::GetBytes([int]$_.FileSystemRights),0);Inherited=$_.IsInherited;InheritanceFlags=[string]$_.InheritanceFlags;PropagationFlags=[string]$_.PropagationFlags}})
}|ConvertTo-Json -Depth 5 -Compress`
}

// quotePowerShellString renders a value as a PowerShell single quoted string.
// Single quoting is what makes a Windows path safe to interpolate: no escape
// sequence is recognized inside it, so a backslash stays a backslash and a $
// is not expanded. Only the quote character itself needs escaping, by
// doubling it.
func quotePowerShellString(v string) string {
	return "'" + strings.ReplaceAll(v, "'", "''") + "'"
}

// Windows file access rights, as defined by the FileSystemRights enumeration
// and the generic rights that appear in a raw access mask.
const (
	AclReadData                     int64 = 0x000001
	AclWriteData                    int64 = 0x000002
	AclAppendData                   int64 = 0x000004
	AclReadExtendedAttributes       int64 = 0x000008
	AclWriteExtendedAttributes      int64 = 0x000010
	AclExecuteFile                  int64 = 0x000020
	AclDeleteSubdirectoriesAndFiles int64 = 0x000040
	AclReadAttributes               int64 = 0x000080
	AclWriteAttributes              int64 = 0x000100
	AclDelete                       int64 = 0x010000
	AclReadPermissions              int64 = 0x020000
	AclChangePermissions            int64 = 0x040000
	AclTakeOwnership                int64 = 0x080000
	AclSynchronize                  int64 = 0x100000
	AclFullControl                  int64 = 0x1F01FF

	AclGenericAll     int64 = 0x10000000
	AclGenericExecute int64 = 0x20000000
	AclGenericWrite   int64 = 0x40000000
	AclGenericRead    int64 = 0x80000000
)

// Groupings behind the derived predicates.
const (
	aclReadMask = AclReadData | AclReadExtendedAttributes | AclReadAttributes |
		AclReadPermissions | AclGenericRead | AclGenericAll

	// Delete counts as write: a principal that can delete a file can change
	// what the path resolves to, which is the question "who can write here"
	// is really asking.
	aclWriteMask = AclWriteData | AclAppendData | AclWriteExtendedAttributes |
		AclWriteAttributes | AclDelete | AclDeleteSubdirectoriesAndFiles |
		AclGenericWrite | AclGenericAll

	aclExecuteMask = AclExecuteFile | AclGenericExecute | AclGenericAll

	aclDeleteMask = AclDelete | AclDeleteSubdirectoriesAndFiles | AclGenericAll

	// Either of these lets a principal rewrite the access control list and
	// grant itself everything else, so they are as strong as full control.
	aclPermissionChangeMask = AclChangePermissions | AclTakeOwnership | AclGenericAll
)

// WindowsAcl is a filesystem object's discretionary access control list.
type WindowsAcl struct {
	Path     string `json:"Path"`
	Owner    string `json:"Owner"`
	OwnerSid string `json:"OwnerSid"`
	Group    string `json:"Group"`
	Sddl     string `json:"Sddl"`
	// Protected is true when inheritance from the parent has been broken.
	Protected bool              `json:"Protected"`
	Access    []WindowsAclEntry `json:"Access"`
}

// WindowsAclEntry is one access control entry.
type WindowsAclEntry struct {
	Identity string `json:"Identity"`
	Sid      string `json:"Sid"`
	// Type is Allow or Deny.
	Type string `json:"Type"`
	// Rights is what .NET names the mask, which is not always a name. A mask
	// carrying generic bits has no label and arrives as a signed decimal
	// string such as "-536805376".
	Rights           string `json:"Rights"`
	Mask             int64  `json:"Mask"`
	Inherited        bool   `json:"Inherited"`
	InheritanceFlags string `json:"InheritanceFlags"`
	PropagationFlags string `json:"PropagationFlags"`
}

// AllowsRead reports whether the entry covers any form of read.
func (e WindowsAclEntry) AllowsRead() bool { return e.Mask&aclReadMask != 0 }

// AllowsWrite reports whether the entry covers any form of write, including
// appending, changing attributes, and deleting.
func (e WindowsAclEntry) AllowsWrite() bool { return e.Mask&aclWriteMask != 0 }

// AllowsExecute reports whether the entry covers executing a file or
// traversing a directory.
func (e WindowsAclEntry) AllowsExecute() bool { return e.Mask&aclExecuteMask != 0 }

// AllowsDelete reports whether the entry covers deleting the object or its
// children.
func (e WindowsAclEntry) AllowsDelete() bool { return e.Mask&aclDeleteMask != 0 }

// AllowsFullControl reports whether the entry covers every right, either as
// the full FileSystemRights set or as the generic all mask.
func (e WindowsAclEntry) AllowsFullControl() bool {
	return e.Mask&AclGenericAll != 0 || e.Mask&AclFullControl == AclFullControl
}

// AllowsPermissionChange reports whether the entry lets the principal rewrite
// the access control list or take ownership, either of which lets it grant
// itself every other right.
func (e WindowsAclEntry) AllowsPermissionChange() bool {
	return e.Mask&aclPermissionChangeMask != 0
}

// IsAllow reports whether the entry grants rather than denies.
func (e WindowsAclEntry) IsAllow() bool { return strings.EqualFold(e.Type, "Allow") }

// AllowedWritePrincipals returns the accounts named by allow entries that
// grant any form of write, in the order the entries appear and without
// repeats.
//
// Deny entries are not subtracted. A deny entry only ever removes access, so
// the result is a superset of the principals that can really write: an audit
// asserting that only certain accounts may write reports one that a deny
// entry had already stopped, rather than missing one that can.
func (a *WindowsAcl) AllowedWritePrincipals() []string {
	seen := make(map[string]struct{}, len(a.Access))
	res := []string{}
	for _, e := range a.Access {
		if !e.IsAllow() || !e.AllowsWrite() {
			continue
		}
		if _, ok := seen[e.Identity]; ok {
			continue
		}
		seen[e.Identity] = struct{}{}
		res = append(res, e.Identity)
	}
	return res
}

// AclEntryID builds the resource id of an access control entry.
//
// An entry repeats along more dimensions than the principal: the same account
// legitimately appears more than once on one object, once allowing and once
// denying, and once inherited and once set directly, with different rights and
// different propagation each time. All of them are in the id, because
// CreateResource returns the cached first instance for a repeated id, so a
// missing dimension makes the second entry report the first one's rights.
func AclEntryID(path, identity, aceType string, mask int64, inherited bool, inheritanceFlags, propagationFlags string) string {
	var b strings.Builder
	b.WriteString("windows.acl.entry/")
	b.WriteString(path)
	b.WriteString("/")
	b.WriteString(identity)
	b.WriteString("/")
	b.WriteString(aceType)
	b.WriteString("/")
	b.WriteString(strconv.FormatInt(mask, 10))
	if inherited {
		b.WriteString("/inherited")
	} else {
		b.WriteString("/direct")
	}
	b.WriteString("/")
	b.WriteString(inheritanceFlags)
	b.WriteString("/")
	b.WriteString(propagationFlags)
	return b.String()
}

// AclID builds the resource id of an access control list.
func AclID(path string) string {
	return "windows.acl/" + path
}

// ParseWindowsAcl decodes an access control list.
func ParseWindowsAcl(input io.Reader) (*WindowsAcl, error) {
	var res WindowsAcl
	if err := json.NewDecoder(input).Decode(&res); err != nil {
		return nil, err
	}
	return &res, nil
}
