// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"

	"github.com/go-ldap/ldap/v3"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers/activedirectory/connection"
)

func (a *mqlActivedirectory) trusts() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.ActiveDirectoryConnection)
	baseDN := conn.BaseDN()
	searchBase := "CN=System," + baseDN

	attrs := []string{
		"cn",
		"trustPartner",
		"trustType",
		"trustDirection",
		"trustAttributes",
		"securityIdentifier",
		"whenCreated",
	}

	entries, err := connection.PagedSearch(conn.LDAPConn(), ldap.NewSearchRequest(
		searchBase,
		ldap.ScopeSingleLevel,
		ldap.NeverDerefAliases, 0, 0, false,
		"(objectClass=trustedDomain)",
		attrs,
		nil,
	))
	if err != nil {
		return nil, fmt.Errorf("failed to query trusts: %w", err)
	}

	sourceDomain := dnToDomain(baseDN)
	res := make([]interface{}, 0, len(entries))

	for _, entry := range entries {
		targetDomain := connection.GetStringAttr(entry, "trustPartner")
		trustTypeRaw := parseInt64Attr(connection.GetStringAttr(entry, "trustType"))
		trustDirRaw := parseInt64Attr(connection.GetStringAttr(entry, "trustDirection"))
		trustAttrs := parseInt64Attr(connection.GetStringAttr(entry, "trustAttributes"))
		whenCreated := parseADGeneralizedTime(connection.GetStringAttr(entry, "whenCreated"))

		trustTypeStr := parseTrustType(trustTypeRaw, trustAttrs)
		trustDirStr := parseTrustDirection(trustDirRaw)

		isTransitive := (trustAttrs & 0x1) == 0
		sidFilteringEnabled := (trustAttrs & 0x4) != 0
		selectiveAuth := (trustAttrs & 0x10) != 0
		tgtDelegation := (trustAttrs & 0x400) != 0
		isAzureADTrust := (trustAttrs & 0x200) != 0
		isIntraForest := (trustAttrs & 0x20) != 0

		// SID history is enabled when SID filtering is off and the trust
		// is not within the same forest.
		sidHistoryEnabled := !sidFilteringEnabled && !isIntraForest

		// Encryption: AD 2012+ defaults to AES for all trusts.
		// Non-forest trusts may fall back to RC4.
		isForestTrust := (trustAttrs & 0x8) != 0
		aesEncryption := true
		rc4Encryption := !isForestTrust

		resource, err := CreateResource(a.MqlRuntime, "activedirectory.trust",
			map[string]*llx.RawData{
				"targetDomain":            llx.StringData(targetDomain),
				"sourceDomain":            llx.StringData(sourceDomain),
				"trustType":               llx.StringData(trustTypeStr),
				"trustDirection":          llx.StringData(trustDirStr),
				"sidFilteringEnabled":     llx.BoolData(sidFilteringEnabled),
				"sidHistoryEnabled":       llx.BoolData(sidHistoryEnabled),
				"selectiveAuthentication": llx.BoolData(selectiveAuth),
				"aesEncryption":           llx.BoolData(aesEncryption),
				"rc4Encryption":           llx.BoolData(rc4Encryption),
				"tgtDelegation":           llx.BoolData(tgtDelegation),
				"whenCreated":             llx.TimeData(whenCreated),
				"isAzureADTrust":          llx.BoolData(isAzureADTrust),
				"isTransitive":            llx.BoolData(isTransitive),
				"trustAttributes":         llx.IntData(trustAttrs),
			})
		if err != nil {
			return nil, err
		}

		res = append(res, resource)
	}

	return res, nil
}

// parseTrustType maps the AD trustType integer attribute to a label,
// promoting Uplevel trusts to "Forest" when the forest-trust attribute
// flag (0x8) is set.
func parseTrustType(trustType, trustAttrs int64) string {
	switch trustType {
	case 1:
		return "Downlevel"
	case 2:
		if trustAttrs&0x8 != 0 {
			return "Forest"
		}
		return "Uplevel"
	case 3:
		return "MIT"
	case 4:
		return "DCE"
	default:
		return "Unknown"
	}
}

// parseTrustDirection maps the AD trustDirection integer attribute to a label.
func parseTrustDirection(dir int64) string {
	switch dir {
	case 0:
		return "Disabled"
	case 1:
		return "Inbound"
	case 2:
		return "Outbound"
	case 3:
		return "Bidirectional"
	default:
		return "Unknown"
	}
}
