// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"encoding/binary"
	"testing"
)

func buildDNSZoneProperty(id uint32, value uint64) []byte {
	buf := make([]byte, 28)
	binary.LittleEndian.PutUint32(buf[0:4], 8)  // data length
	binary.LittleEndian.PutUint32(buf[4:8], 0)  // name length
	binary.LittleEndian.PutUint32(buf[8:12], 0) // flags
	binary.LittleEndian.PutUint32(buf[12:16], 1)
	binary.LittleEndian.PutUint32(buf[16:20], id)
	binary.LittleEndian.PutUint64(buf[20:28], value)
	return buf
}

func TestParseDNSZoneProperty(t *testing.T) {
	prop, ok := parseDNSZoneProperty(buildDNSZoneProperty(dnsZonePropertyAllowUpdate, dnsZoneAllowUpdateOnly))
	if !ok {
		t.Fatal("expected property to parse")
	}
	if prop.id != dnsZonePropertyAllowUpdate {
		t.Fatalf("prop.id = %d", prop.id)
	}
	value, ok := prop.uint64()
	if !ok {
		t.Fatal("expected integer value")
	}
	if value != dnsZoneAllowUpdateOnly {
		t.Fatalf("value = %d", value)
	}
}

func TestDeriveDNSZoneSettings(t *testing.T) {
	zoneType, dynamicUpdate, secureOnly := deriveDNSZoneSettings([][]byte{
		buildDNSZoneProperty(dnsZonePropertyType, 1),
		buildDNSZoneProperty(dnsZonePropertyAllowUpdate, dnsZoneAllowUpdateOnly),
	})
	if zoneType != "Primary" {
		t.Fatalf("zoneType = %q", zoneType)
	}
	if !dynamicUpdate {
		t.Fatal("expected dynamicUpdate true")
	}
	if !secureOnly {
		t.Fatal("expected secureOnly true")
	}

	zoneType, dynamicUpdate, secureOnly = deriveDNSZoneSettings([][]byte{
		buildDNSZoneProperty(dnsZonePropertyType, 0),
		buildDNSZoneProperty(dnsZonePropertyAllowUpdate, dnsZoneAllowUpdateBoth),
	})
	if zoneType != "Cache" {
		t.Fatalf("zoneType = %q", zoneType)
	}
	if !dynamicUpdate {
		t.Fatal("expected dynamicUpdate true")
	}
	if secureOnly {
		t.Fatal("expected secureOnly false")
	}
}
