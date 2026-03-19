// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"
	"strings"

	"github.com/go-ldap/ldap/v3"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/activedirectory/connection"
)

func initActivedirectory(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	conn := runtime.Connection.(*connection.ActiveDirectoryConnection)
	// Populate fields that are directly available from the connection (no extra LDAP queries).
	if args == nil {
		args = make(map[string]*llx.RawData)
	}
	if _, ok := args["domain"]; !ok {
		args["domain"] = llx.StringData(dnToDomain(conn.BaseDN()))
	}
	if _, ok := args["distinguishedName"]; !ok {
		args["distinguishedName"] = llx.StringData(conn.BaseDN())
	}
	if _, ok := args["domainSid"]; !ok {
		args["domainSid"] = llx.StringData(conn.DomainSID())
	}
	if _, ok := args["functionalLevel"]; !ok {
		args["functionalLevel"] = llx.StringData(conn.DomainFunctionalLevel())
	}
	if _, ok := args["forestFunctionalLevel"]; !ok {
		args["forestFunctionalLevel"] = llx.StringData(conn.ForestFunctionalLevel())
	}
	if _, ok := args["forestName"]; !ok {
		args["forestName"] = llx.StringData(dnToDomain(conn.RootDomainDN()))
	}
	return args, nil, nil
}

func (a *mqlActivedirectory) id() (string, error) {
	return "activedirectory", nil
}

// dnToDomain converts a distinguished name to a DNS domain name.
// DC=corp,DC=example,DC=com → corp.example.com
func dnToDomain(dn string) string {
	parts := strings.Split(dn, ",")
	var labels []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if strings.HasPrefix(strings.ToUpper(p), "DC=") {
			labels = append(labels, p[3:])
		}
	}
	return strings.Join(labels, ".")
}

// The following methods are now computed fields (defined with () in .lr):
// netbiosName, lapsEnabled, schemaVersion. The plain fields (domain, distinguishedName,
// domainSid, functionalLevel, forestFunctionalLevel, forestName) are set in init above.

func (a *mqlActivedirectory) netbiosName() (string, error) {
	conn := a.MqlRuntime.Connection.(*connection.ActiveDirectoryConnection)
	baseDN := conn.BaseDN()
	configDN := conn.ConfigDN()

	searchBase := "CN=Partitions," + configDN
	filter := fmt.Sprintf("(&(objectClass=crossRef)(nCName=%s))", ldap.EscapeFilter(baseDN))

	result, err := connection.PagedSearch(conn.LDAPConn(), ldap.NewSearchRequest(
		searchBase,
		ldap.ScopeSingleLevel,
		ldap.NeverDerefAliases, 0, 0, false,
		filter,
		[]string{"nETBIOSName"},
		nil,
	))
	if err != nil {
		return "", fmt.Errorf("failed to query NetBIOS name: %w", err)
	}

	if len(result) == 0 {
		return "", nil
	}

	return result[0].GetAttributeValue("nETBIOSName"), nil
}

func (a *mqlActivedirectory) lapsEnabled() (bool, error) {
	conn := a.MqlRuntime.Connection.(*connection.ActiveDirectoryConnection)
	schemaDN := conn.SchemaDN()

	// Search for LAPS schema attributes: legacy ms-Mcs-AdmPwd or new msLAPS-Password.
	filter := "(|(lDAPDisplayName=ms-Mcs-AdmPwd)(lDAPDisplayName=msLAPS-Password))"
	result, err := connection.PagedSearch(conn.LDAPConn(), ldap.NewSearchRequest(
		schemaDN,
		ldap.ScopeSingleLevel,
		ldap.NeverDerefAliases, 0, 0, false,
		filter,
		[]string{"lDAPDisplayName"},
		nil,
	))
	if err != nil {
		// Schema query may fail on restricted permissions; treat as not enabled.
		return false, nil
	}

	return len(result) > 0, nil
}

func (a *mqlActivedirectory) schemaVersion() (int64, error) {
	conn := a.MqlRuntime.Connection.(*connection.ActiveDirectoryConnection)
	schemaDN := conn.SchemaDN()

	result, err := connection.PagedSearch(conn.LDAPConn(), ldap.NewSearchRequest(
		schemaDN,
		ldap.ScopeBaseObject,
		ldap.NeverDerefAliases, 0, 0, false,
		"(objectClass=*)",
		[]string{"objectVersion"},
		nil,
	))
	if err != nil {
		return 0, fmt.Errorf("failed to query schema version: %w", err)
	}

	if len(result) == 0 {
		return 0, nil
	}

	v := result[0].GetAttributeValue("objectVersion")
	if v == "" {
		return 0, nil
	}

	var version int64
	_, err = fmt.Sscanf(v, "%d", &version)
	if err != nil {
		return 0, fmt.Errorf("failed to parse schema version %q: %w", v, err)
	}

	return version, nil
}
