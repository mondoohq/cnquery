// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/go-ldap/ldap/v3"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/activedirectory/connection"
)

// parseGroupType interprets the AD groupType bitmask and returns a
// human-readable label (e.g. "Security - Global") plus the raw int64 value.
// groupType is stored as a decimal string representing a signed 32-bit integer.
func parseGroupType(raw int64) (string, int64) {
	// Signed 32-bit interpretation: bit 31 → security group.
	isSecurity := (raw & 0x80000000) != 0

	var kind string
	if isSecurity {
		kind = "Security"
	} else {
		kind = "Distribution"
	}

	scope := "Unknown"
	switch {
	case raw&0x02 != 0:
		scope = "Global"
	case raw&0x04 != 0:
		scope = "DomainLocal"
	case raw&0x08 != 0:
		scope = "Universal"
	}

	return kind + " - " + scope, raw
}

// privilegedGroupSIDs builds the set of well-known privileged group SIDs
// from the shared privilegedGroups definitions in privileged.go.
func privilegedGroupSIDs(domainSID, rootDomainSID string) map[string]bool {
	sids := make(map[string]bool, len(privilegedGroups))
	for _, pg := range privilegedGroups {
		var sid string
		switch pg.Base {
		case "domain":
			sid = domainSID + "-" + pg.RID
		case "forest":
			sid = rootDomainSID + "-" + pg.RID
		case "builtin":
			sid = "S-1-5-32-" + pg.RID
		default:
			continue
		}
		sids[sid] = true
	}
	return sids
}

// computeMemberCount determines the accurate member count for a group.
// If the member attribute was returned with range markers (indicating
// truncation at MaxValRange), it performs full range retrieval to count
// all members. Otherwise it uses the directly returned member values.
func computeMemberCount(l *ldap.Conn, entry *ldap.Entry, groupDN string) (int64, bool) {
	// Check if the server returned a range-limited member attribute.
	for _, attr := range entry.Attributes {
		if strings.HasPrefix(strings.ToLower(attr.Name), "member;range=") {
			// Range-limited: do full retrieval for accurate count.
			allMembers, err := rangeRetrieveMembers(l, groupDN)
			if err != nil {
				// Fall back to what we have.
				count := int64(len(attr.Values))
				return count, count == 0
			}
			count := int64(len(allMembers))
			return count, count == 0
		}
	}

	// No range limitation: use the direct member attribute.
	members := connection.GetStringSliceAttr(entry, "member")
	count := int64(len(members))
	return count, count == 0
}

func (a *mqlActivedirectory) groups() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.ActiveDirectoryConnection)
	baseDN := conn.BaseDN()

	attrs := []string{
		"sAMAccountName",
		"distinguishedName",
		"displayName",
		"objectSid",
		"groupType",
		"description",
		"adminCount",
		"whenCreated",
		"member",
	}

	entries, err := connection.PagedSearch(conn.LDAPConn(), ldap.NewSearchRequest(
		baseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases, 0, 0, false,
		"(objectClass=group)",
		attrs,
		nil,
	))
	if err != nil {
		return nil, fmt.Errorf("failed to query groups: %w", err)
	}

	privSIDs := privilegedGroupSIDs(conn.DomainSID(), conn.RootDomainSID())

	res := make([]interface{}, 0, len(entries))
	for _, entry := range entries {
		samName := connection.GetStringAttr(entry, "sAMAccountName")
		dn := connection.GetStringAttr(entry, "distinguishedName")
		displayName := connection.GetStringAttr(entry, "displayName")
		desc := connection.GetStringAttr(entry, "description")

		// SID
		sidRaw := connection.GetBinaryAttr(entry, "objectSid")
		sid, _ := connection.DecodeSID(sidRaw)

		// groupType: stored as signed 32-bit decimal string
		groupTypeRaw := parseInt64Attr(connection.GetStringAttr(entry, "groupType"))
		// Sign-extend from 32-bit: AD stores negative values like -2147483646.
		// parseInt64Attr already parses negative strings correctly, but if the value
		// came as unsigned 32-bit, force sign extension.
		groupTypeLabel, groupTypeVal := parseGroupType(groupTypeRaw)

		// adminCount: AD stores as "1" when set; 0 or absent otherwise
		adminCount := connection.GetStringAttr(entry, "adminCount") == "1"

		// whenCreated: AD generalized time "20060102150405.0Z"
		whenCreated := parseADGeneralizedTime(connection.GetStringAttr(entry, "whenCreated"))

		// Get initial member count. AD may truncate at MaxValRange (1500),
		// so check for range-limited responses and do full retrieval if needed.
		memberCount, isEmpty := computeMemberCount(conn.LDAPConn(), entry, dn)

		ouPath := extractOU(dn)

		isPrivileged := privSIDs[sid]

		resource, err := CreateResource(a.MqlRuntime, "activedirectory.group",
			map[string]*llx.RawData{
				"sAMAccountName":   llx.StringData(samName),
				"distinguishedName": llx.StringData(dn),
				"displayName":      llx.StringData(displayName),
				"sid":              llx.StringData(sid),
				"groupType":        llx.StringData(groupTypeLabel),
				"groupTypeRaw":     llx.IntData(groupTypeVal),
				"description":      llx.StringData(desc),
				"adminCount":       llx.BoolData(adminCount),
				"memberCount":      llx.IntData(memberCount),
				"isPrivileged":     llx.BoolData(isPrivileged),
				"isEmpty":          llx.BoolData(isEmpty),
				"whenCreated":      llx.TimeData(whenCreated),
				"ouPath":           llx.StringData(ouPath),
			})
		if err != nil {
			return nil, err
		}

		res = append(res, resource)
	}

	return res, nil
}

func (a *mqlActivedirectoryGroup) id() (string, error) {
	return a.DistinguishedName.Data, nil
}

// members performs full range retrieval of the group's member attribute and
// resolves each member DN to an activedirectory.groupMember resource.
func (a *mqlActivedirectoryGroup) members() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.ActiveDirectoryConnection)
	groupDN := a.DistinguishedName.Data

	memberDNs, err := rangeRetrieveMembers(conn.LDAPConn(), groupDN)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve members for group %s: %w", groupDN, err)
	}

	res := make([]interface{}, 0, len(memberDNs))
	for _, memberDN := range memberDNs {
		member, err := resolveMember(a.MqlRuntime, conn, memberDN)
		if err != nil {
			// Skip orphaned or deleted objects gracefully.
			continue
		}
		res = append(res, member)
	}

	return res, nil
}

// rangeRetrieveMembers collects all member DNs from a group using AD's
// range retrieval mechanism. AD returns at most MaxValRange (typically 1500)
// values per request; this function pages through until all are collected.
func rangeRetrieveMembers(l *ldap.Conn, groupDN string) ([]string, error) {
	var allMembers []string
	rangeStart := 0

	for {
		rangeAttr := fmt.Sprintf("member;range=%d-*", rangeStart)
		entries, err := connection.PagedSearch(l, ldap.NewSearchRequest(
			groupDN,
			ldap.ScopeBaseObject,
			ldap.NeverDerefAliases, 0, 0, false,
			"(objectClass=*)",
			[]string{rangeAttr},
			nil,
		))
		if err != nil {
			return nil, err
		}
		if len(entries) == 0 {
			break
		}

		entry := entries[0]
		var found bool
		var isTerminal bool
		for _, attr := range entry.Attributes {
			if !strings.HasPrefix(strings.ToLower(attr.Name), "member;range=") {
				continue
			}
			found = true
			allMembers = append(allMembers, attr.Values...)

			// Parse range header to determine if this is the last page.
			// Format: "member;range=X-Y" where Y="*" on the terminal page.
			rangePart := attr.Name[len("member;range="):]
			parts := strings.SplitN(rangePart, "-", 2)
			if len(parts) == 2 && parts[1] == "*" {
				isTerminal = true
			} else if len(parts) == 2 {
				endVal, err := strconv.Atoi(parts[1])
				if err != nil {
					return allMembers, nil
				}
				rangeStart = endVal + 1
			}
			break
		}

		if !found || isTerminal {
			// No member attribute returned (empty group) or terminal page reached.
			break
		}
	}

	return allMembers, nil
}

// resolveMember queries a single member DN to determine its type and
// creates an activedirectory.groupMember resource.
func resolveMember(runtime *plugin.Runtime, conn *connection.ActiveDirectoryConnection, memberDN string) (interface{}, error) {
	entries, err := connection.PagedSearch(conn.LDAPConn(), ldap.NewSearchRequest(
		memberDN,
		ldap.ScopeBaseObject,
		ldap.NeverDerefAliases, 0, 0, false,
		"(objectClass=*)",
		[]string{"sAMAccountName", "objectSid", "objectClass"},
		nil,
	))
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("member not found: %s", memberDN)
	}

	entry := entries[0]
	name := connection.GetStringAttr(entry, "sAMAccountName")
	sidRaw := connection.GetBinaryAttr(entry, "objectSid")
	sid, _ := connection.DecodeSID(sidRaw)

	classes := connection.GetStringSliceAttr(entry, "objectClass")
	memberType := classifyMember(classes)

	return CreateResource(runtime, "activedirectory.groupMember",
		map[string]*llx.RawData{
			"name":              llx.StringData(name),
			"distinguishedName": llx.StringData(memberDN),
			"sid":               llx.StringData(sid),
			"type":              llx.StringData(memberType),
		})
}

// classifyMember determines the member type from its objectClass list.
func classifyMember(classes []string) string {
	for _, c := range classes {
		lower := strings.ToLower(c)
		if lower == "computer" {
			return "computer"
		}
	}
	for _, c := range classes {
		if strings.ToLower(c) == "group" {
			return "group"
		}
	}
	return "user"
}

func (a *mqlActivedirectoryGroupMember) id() (string, error) {
	return a.DistinguishedName.Data, nil
}
