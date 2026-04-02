// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"reflect"
	"testing"
)

func TestClassifyNamingContexts(t *testing.T) {
	namingContexts := []string{
		"DC=mini,DC=lab",
		"CN=Configuration,DC=mini,DC=lab",
		"CN=Schema,CN=Configuration,DC=mini,DC=lab",
		"DC=DomainDnsZones,DC=mini,DC=lab",
		"DC=ForestDnsZones,DC=mini,DC=lab",
		"DC=root,DC=example,DC=com",
		"DC=child,DC=root,DC=example,DC=com",
		"DC=mini,DC=lab",
	}

	domainDNSZonesDN, forestDNSZonesDN, domainNCs := classifyNamingContexts(namingContexts)

	if domainDNSZonesDN != "DC=DomainDnsZones,DC=mini,DC=lab" {
		t.Fatalf("domainDnsZonesDN = %q", domainDNSZonesDN)
	}
	if forestDNSZonesDN != "DC=ForestDnsZones,DC=mini,DC=lab" {
		t.Fatalf("forestDnsZonesDN = %q", forestDNSZonesDN)
	}

	wantDomainNCs := []string{
		"DC=mini,DC=lab",
		"DC=root,DC=example,DC=com",
		"DC=child,DC=root,DC=example,DC=com",
	}
	if !reflect.DeepEqual(domainNCs, wantDomainNCs) {
		t.Fatalf("domainNCs = %#v, want %#v", domainNCs, wantDomainNCs)
	}
}
