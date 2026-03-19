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

func (a *mqlActivedirectory) domain() (string, error) {
	conn := a.MqlRuntime.Connection.(*connection.ActiveDirectoryConnection)
	return dnToDomain(conn.BaseDN()), nil
}

func (a *mqlActivedirectory) distinguishedName() (string, error) {
	conn := a.MqlRuntime.Connection.(*connection.ActiveDirectoryConnection)
	return conn.BaseDN(), nil
}

func (a *mqlActivedirectory) domainSid() (string, error) {
	conn := a.MqlRuntime.Connection.(*connection.ActiveDirectoryConnection)
	return conn.DomainSID(), nil
}

func (a *mqlActivedirectory) functionalLevel() (string, error) {
	conn := a.MqlRuntime.Connection.(*connection.ActiveDirectoryConnection)
	return conn.DomainFunctionalLevel(), nil
}

func (a *mqlActivedirectory) forestFunctionalLevel() (string, error) {
	conn := a.MqlRuntime.Connection.(*connection.ActiveDirectoryConnection)
	return conn.ForestFunctionalLevel(), nil
}

func (a *mqlActivedirectory) forestName() (string, error) {
	conn := a.MqlRuntime.Connection.(*connection.ActiveDirectoryConnection)
	return dnToDomain(conn.RootDomainDN()), nil
}

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
