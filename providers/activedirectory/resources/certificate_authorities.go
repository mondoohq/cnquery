// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"crypto/x509"
	"fmt"
	"time"

	"github.com/go-ldap/ldap/v3"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers/activedirectory/connection"
	"go.mondoo.com/mql/v13/types"
)

func (a *mqlActivedirectory) certificateAuthorities() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.ActiveDirectoryConnection)
	baseDN := fmt.Sprintf("CN=Enrollment Services,CN=Public Key Services,CN=Services,%s", conn.ConfigDN())

	attrs := []string{
		"cn",
		"distinguishedName",
		"dNSHostName",
		"certificateTemplates",
		"cACertificate",
	}

	entries, err := connection.PagedSearch(conn.LDAPConn(), ldap.NewSearchRequest(
		baseDN,
		ldap.ScopeSingleLevel,
		ldap.NeverDerefAliases, 0, 0, false,
		"(objectClass=pKIEnrollmentService)",
		attrs,
		nil,
	))
	if err != nil {
		if ldap.IsErrorWithCode(err, ldap.LDAPResultNoSuchObject) {
			log.Warn().Msg("ADCS Enrollment Services container not found, ADCS may not be installed")
			return []interface{}{}, nil
		}
		return nil, fmt.Errorf("failed to query certificate authorities: %w", err)
	}

	res := make([]interface{}, 0, len(entries))
	for _, entry := range entries {
		name := connection.GetStringAttr(entry, "cn")
		dn := connection.GetStringAttr(entry, "distinguishedName")
		dnsHostname := connection.GetStringAttr(entry, "dNSHostName")

		// Parse certificate expiration and infer whether the Enterprise CA is root
		// or subordinate from the published CA certificate.
		caType := "Enterprise"
		var certExpiration time.Time
		rawCert := entry.GetRawAttributeValue("cACertificate")
		if len(rawCert) > 0 {
			cert, parseErr := x509.ParseCertificate(rawCert)
			if parseErr != nil {
				log.Warn().Err(parseErr).Str("ca", name).Msg("failed to parse cACertificate, using zero time for expiration")
			} else {
				certExpiration = cert.NotAfter.UTC()
				if cert.CheckSignatureFrom(cert) == nil {
					caType = "Enterprise Root"
				} else {
					caType = "Enterprise Subordinate"
				}
			}
		}

		// Template names are CN values, not full DNs.
		templates := connection.GetStringSliceAttr(entry, "certificateTemplates")
		templatesRaw := make([]interface{}, len(templates))
		for i, v := range templates {
			templatesRaw[i] = v
		}

		resource, err := CreateResource(a.MqlRuntime, "activedirectory.certificateAuthority",
			map[string]*llx.RawData{
				"name":                  llx.StringData(name),
				"distinguishedName":     llx.StringData(dn),
				"dnsHostname":           llx.StringData(dnsHostname),
				"caType":                llx.StringData(caType),
				"certificateTemplates":  llx.ArrayData(templatesRaw, types.String),
				"certificateExpiration": llx.TimeData(certExpiration),
			})
		if err != nil {
			return nil, err
		}

		res = append(res, resource)
	}

	return res, nil
}
