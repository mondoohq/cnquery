// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-ldap/ldap/v3"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers/activedirectory/connection"
)

// fileTimeToTime converts a Windows FILETIME (100ns intervals since 1601-01-01)
// to a Go time.Time. Sentinel values 0 and 0x7FFFFFFFFFFFFFFF (never) return
// the zero time.
func fileTimeToTime(ft int64) time.Time {
	if ft <= 0 || ft == 0x7FFFFFFFFFFFFFFF {
		return time.Time{}
	}
	// Windows epoch starts 116444736000000000 100ns intervals before Unix epoch.
	const epochDiff = 116_444_736_000_000_000
	unixNano := (ft - epochDiff) * 100
	return time.Unix(0, unixNano).UTC()
}

// extractSiteFromServerRef extracts the AD site name from a serverReferenceBL DN.
// Expected format: CN=<DC>,CN=Servers,CN=<SiteName>,CN=Sites,CN=Configuration,...
func extractSiteFromServerRef(dn string) string {
	if dn == "" {
		return ""
	}
	parts := strings.Split(dn, ",")
	for i := 0; i < len(parts)-2; i++ {
		if strings.EqualFold(strings.TrimSpace(parts[i]), "CN=Servers") &&
			strings.HasPrefix(strings.TrimSpace(parts[i+1]), "CN=") &&
			strings.EqualFold(strings.TrimSpace(parts[i+2]), "CN=Sites") {
			return strings.TrimSpace(parts[i+1])[3:]
		}
	}
	return ""
}

func (a *mqlActivedirectoryDomainController) id() (string, error) {
	return a.DistinguishedName.Data, nil
}

func (a *mqlActivedirectory) domainControllers() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.ActiveDirectoryConnection)
	baseDN := conn.BaseDN()

	filter := fmt.Sprintf("(userAccountControl:1.2.840.113556.1.4.803:=%d)", UACServerTrustAccount)
	attrs := []string{
		"dNSHostName",
		"distinguishedName",
		"operatingSystem",
		"operatingSystemVersion",
		"userAccountControl",
		"pwdLastSet",
		"lastLogonTimestamp",
		"serverReferenceBL",
	}

	result, err := connection.PagedSearch(conn.LDAPConn(), ldap.NewSearchRequest(
		baseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases, 0, 0, false,
		filter,
		attrs,
		nil,
	))
	if err != nil {
		return nil, fmt.Errorf("failed to query domain controllers: %w", err)
	}

	res := make([]interface{}, 0, len(result))
	for _, entry := range result {
		name := entry.GetAttributeValue("dNSHostName")
		dn := entry.GetAttributeValue("distinguishedName")
		operatingSystem := entry.GetAttributeValue("operatingSystem")
		osVersion := entry.GetAttributeValue("operatingSystemVersion")
		uac := parseInt64Attr(entry.GetAttributeValue("userAccountControl"))

		pwdLastSet := fileTimeToTime(parseInt64Attr(entry.GetAttributeValue("pwdLastSet")))
		lastLogon := fileTimeToTime(parseInt64Attr(entry.GetAttributeValue("lastLogonTimestamp")))

		serverRef := entry.GetAttributeValue("serverReferenceBL")
		site := extractSiteFromServerRef(serverRef)

		isRODC := uacHasFlag(uac, UACPartialSecretsAccount)
		isGlobalCatalog, err := isGlobalCatalogServer(conn, name, serverRef)
		if err != nil {
			return nil, fmt.Errorf("failed to determine GC state for %s: %w", name, err)
		}

		resource, err := CreateResource(a.MqlRuntime, "activedirectory.domainController",
			map[string]*llx.RawData{
				"name":                   llx.StringData(name),
				"distinguishedName":      llx.StringData(dn),
				"operatingSystem":        llx.StringData(operatingSystem),
				"operatingSystemVersion": llx.StringData(osVersion),
				"isGlobalCatalog":        llx.BoolData(isGlobalCatalog),
				"isRODC":                 llx.BoolData(isRODC),
				"pwdLastSet":             llx.TimeData(pwdLastSet),
				"lastLogonTimestamp":     llx.TimeData(lastLogon),
				"site":                   llx.StringData(site),
			})
		if err != nil {
			return nil, err
		}

		res = append(res, resource)
	}

	return res, nil
}
