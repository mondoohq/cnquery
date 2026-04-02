// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import "encoding/binary"

const (
	dnsZonePropertyType        = 0x00000001
	dnsZonePropertyAllowUpdate = 0x00000002
)

const (
	dnsZoneAllowUpdateNone = 0
	dnsZoneAllowUpdateBoth = 1
	dnsZoneAllowUpdateOnly = 2
)

type dnsZoneProperty struct {
	id   uint32
	data []byte
}

func parseDNSZoneProperty(raw []byte) (dnsZoneProperty, bool) {
	if len(raw) < 20 {
		return dnsZoneProperty{}, false
	}

	dataLength := binary.LittleEndian.Uint32(raw[0:4])
	id := binary.LittleEndian.Uint32(raw[16:20])
	// Guard against uint32 overflow: check dataLength independently before adding.
	if dataLength > uint32(len(raw)) || int(20+dataLength) > len(raw) {
		return dnsZoneProperty{}, false
	}

	data := make([]byte, dataLength)
	copy(data, raw[20:20+dataLength])
	return dnsZoneProperty{id: id, data: data}, true
}

func (p dnsZoneProperty) uint64() (uint64, bool) {
	if len(p.data) == 0 || len(p.data) > 8 {
		return 0, false
	}
	var buf [8]byte
	copy(buf[:], p.data)
	return binary.LittleEndian.Uint64(buf[:]), true
}

func classifyDNSZoneType(value uint64) string {
	switch value {
	case 0:
		return "Cache"
	case 1:
		return "Primary"
	case 2:
		return "Secondary"
	case 3:
		return "Stub"
	case 4:
		return "Forwarder"
	case 5:
		return "SecondaryCache"
	default:
		return "Unknown"
	}
}

func deriveDNSZoneSettings(rawValues [][]byte) (string, bool, bool) {
	zoneType := "Unknown"
	dynamicUpdate := false
	secureOnly := false

	for _, raw := range rawValues {
		prop, ok := parseDNSZoneProperty(raw)
		if !ok {
			continue
		}

		value, ok := prop.uint64()
		if !ok {
			continue
		}

		switch prop.id {
		case dnsZonePropertyType:
			zoneType = classifyDNSZoneType(value)
		case dnsZonePropertyAllowUpdate:
			dynamicUpdate = value != dnsZoneAllowUpdateNone
			secureOnly = value == dnsZoneAllowUpdateOnly
		}
	}

	return zoneType, dynamicUpdate, secureOnly
}
