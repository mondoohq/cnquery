// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build windows

package networki

import (
	"fmt"
	"net"
	"syscall"
	"unsafe"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"golang.org/x/sys/windows"
)

const (
	// Windows address family constants
	AF_INET  = 2  // IPv4
	AF_INET6 = 23 // IPv6
)

var (
	iphlpapi                 = windows.NewLazySystemDLL("iphlpapi.dll")
	procGetIpForwardTable2   = iphlpapi.NewProc("GetIpForwardTable2")
	procFreeMibTable         = iphlpapi.NewProc("FreeMibTable")
	procGetAdaptersAddresses = iphlpapi.NewProc("GetAdaptersAddresses")
)

// detectWindowsRoutes detects network routes on Windows using native IP Helper API
func (n *neti) detectWindowsRoutes() ([]Route, error) {
	// Use GetIpForwardTable2 (supports both IPv4 and IPv6, Windows Vista+)
	routes, err := n.detectWindowsRoutesViaGetIpForwardTable()
	if err == nil && len(routes) > 0 {
		return routes, nil
	}
	log.Debug().Err(err).Msg("native Windows API failed")
	return nil, err
	// // Fallback to PowerShell if native APIs fail
	// routes, err = n.detectWindowsRoutesViaPowerShell()
	// if err == nil && len(routes) > 0 {
	// 	return routes, nil
	// }
	// log.Debug().Err(err).Int("routeCount", len(routes)).Msg("PowerShell Get-NetRoute failed, trying netstat")

	// // fallback to netstat
	// return n.detectWindowsRoutesViaNetstat()
}

// SOCKADDR_IN represents an IPv4 socket address
// https://docs.microsoft.com/en-us/windows/win32/winsock/sockaddr-in-2
type sockaddrIn struct {
	SinFamily uint16
	SinPort   [2]byte
	SinAddr   [4]byte
	SinZero   [8]byte
}

// SOCKADDR_IN6 represents an IPv6 socket address
// https://docs.microsoft.com/en-us/windows/win32/winsock/sockaddr-in6-2
type sockaddrIn6 struct {
	Sin6Family   uint16
	Sin6Port     [2]byte
	Sin6Flowinfo uint32
	Sin6Addr     [16]byte
	Sin6ScopeId  uint32
}

// SOCKADDR_INET is a union that can represent either IPv4 or IPv6 address
// https://docs.microsoft.com/en-us/windows/win32/winsock/sockaddr-inet
type sockaddrInet struct {
	// Union: either Ipv4 or Ipv6, determined by Family field
	// We'll use a byte array and cast based on Family
	Data [28]byte // Max size of SOCKADDR_IN6
}

// MIB_IPFORWARD_ROW2 represents a route entry (supports both IPv4 and IPv6)
// https://docs.microsoft.com/en-us/windows/win32/api/netioapi/ns-netioapi-mib_ipforward_row2
// Note: The actual Windows structure has NET_LUID InterfaceLuid (8 bytes) and NET_IFINDEX InterfaceIndex (4 bytes)
// before IP_ADDRESS_PREFIX. IP_ADDRESS_PREFIX contains SOCKADDR_INET Prefix and PrefixLength.
// For simplicity, we'll read the family from DestinationPrefix.Data[0] (first 2 bytes of sockaddr)
type mibIpForwardRow2 struct {
	// NET_LUID InterfaceLuid (8 bytes) - we skip this
	_              [8]byte
	InterfaceIndex uint32
	// IP_ADDRESS_PREFIX DestinationPrefix - contains SOCKADDR_INET (28 bytes) + PrefixLength (4 bytes)
	DestinationPrefix    sockaddrInet
	_                    [4]byte // PrefixLength union
	NextHop              sockaddrInet
	SitePrefixLength     uint8
	ValidLifetime        uint32
	PreferredLifetime    uint32
	Metric               uint32
	Protocol             uint32
	Loopback             uint8
	AutoconfigureAddress uint8
	Publish              uint8
	Immortal             uint8
	Age                  uint32
	Origin               uint32
}

// MIB_IPFORWARD_TABLE2 contains an array of route entries (IPv4 and IPv6)
type mibIpForwardTable2 struct {
	NumEntries uint32
	Table      [1]mibIpForwardRow2 // Variable length array
}

// detectWindowsRoutesViaGetIpForwardTable uses GetIpForwardTable2 to get routes (IPv4 + IPv6)
func (n *neti) detectWindowsRoutesViaGetIpForwardTable() ([]Route, error) {
	// Get interface map for name lookup
	interfaceMap, err := n.getWindowsInterfaceMap()
	if err != nil {
		log.Debug().Err(err).Msg("failed to get interface map, continuing without interface names")
		interfaceMap = make(map[uint32]string)
	}

	var table *mibIpForwardTable2
	ret, _, _ := procGetIpForwardTable2.Call(
		uintptr(0), // AddressFamily: 0 = both IPv4 and IPv6
		uintptr(unsafe.Pointer(&table)),
	)

	if ret != 0 {
		return nil, errors.Errorf("GetIpForwardTable2 failed with error code: %d", ret)
	}
	// Early return if no routes
	if table == nil || table.NumEntries == 0 {
		return []Route{}, nil
	}
	defer procFreeMibTable.Call(uintptr(unsafe.Pointer(table)))

	// Dump raw memory for testing - calculate size of the table structure
	tableSize := unsafe.Sizeof(mibIpForwardTable2{}) +
		uintptr(table.NumEntries-1)*unsafe.Sizeof(mibIpForwardRow2{})
	rawBytes := (*[1 << 20]byte)(unsafe.Pointer(table))[:tableSize:tableSize]
	log.Debug().
		Int("rawBytesLen", len(rawBytes)).
		Hex("rawBytes", rawBytes).
		Msg("dumping raw table bytes for testing")

	var routes []Route

	// Access the variable-length array of route entries
	// Calculate the actual size of each entry to ensure proper alignment
	rowSize := unsafe.Sizeof(mibIpForwardRow2{})
	tableStart := uintptr(unsafe.Pointer(&table.Table[0]))

	log.Debug().
		Uint32("numEntries", table.NumEntries).
		Uint64("rowSize", uint64(rowSize)).
		Uint64("tableStart", uint64(tableStart)).
		Int("interfaceMapSize", len(interfaceMap)).
		Msg("GetIpForwardTable2: processing route entries")

	// Manually iterate through entries to ensure correct alignment
	for i := uint32(0); i < table.NumEntries; i++ {
		// Calculate offset for this entry
		entryOffset := tableStart + uintptr(i)*rowSize

		// Manually read fields from correct offsets
		// Structure layout (from raw bytes analysis):
		// - Offset 0-7: InterfaceLuid (8 bytes)
		// - Offset 8-11: Unknown/padding (4 bytes)
		// - Offset 12-15: InterfaceIndex (4 bytes) - confirmed from raw bytes showing 0d000000 = 13
		// - Offset 16-43: DestinationPrefix SOCKADDR_INET (28 bytes)
		// - Offset 44-47: PrefixLength (4 bytes)
		// - Offset 48-75: NextHop SOCKADDR_INET (28 bytes)
		// - Offset 76: SitePrefixLength (1 byte)
		// - Offset 77-79: Padding (3 bytes)
		// - Offset 80+: Rest of fields

		interfaceIndexOffset := entryOffset + 12 // InterfaceLuid (8) + padding (4) = 12
		interfaceIndex := *(*uint32)(unsafe.Pointer(interfaceIndexOffset))

		destPrefixOffset := entryOffset + 16 // After InterfaceLuid (8) + padding (4) + InterfaceIndex (4) = 16
		actualFamily := *(*uint16)(unsafe.Pointer(destPrefixOffset))

		// Read DestinationPrefix (28 bytes)
		destPrefixData := (*[28]byte)(unsafe.Pointer(destPrefixOffset))
		destPrefix := sockaddrInet{Data: *destPrefixData}

		// Read PrefixLength from IP_ADDRESS_PREFIX (4 bytes) - at offset 44
		prefixLengthOffset := entryOffset + 44
		prefixLength := *(*uint32)(unsafe.Pointer(prefixLengthOffset))

		// Read NextHop (28 bytes) - starts at offset 48
		nextHopOffset := entryOffset + 48
		nextHopData := (*[28]byte)(unsafe.Pointer(nextHopOffset))
		nextHop := sockaddrInet{Data: *nextHopData}

		// Read SitePrefixLength (1 byte) - at offset 76
		sitePrefixLengthOffset := entryOffset + 76
		sitePrefixLength := *(*uint8)(unsafe.Pointer(sitePrefixLengthOffset))

		rawEntrySize := 64
		if rowSize < 64 {
			rawEntrySize = int(rowSize)
		}

		// Validate family value
		if actualFamily != AF_INET && actualFamily != AF_INET6 {
			log.Debug().
				Int("index", int(i)).
				Uint16("family", actualFamily).
				Hex("destPrefix", destPrefix.Data[:8]).
				Hex("rawEntry", (*[64]byte)(unsafe.Pointer(entryOffset))[:rawEntrySize]).
				Msg("invalid family in destination prefix, skipping route")
			continue
		}

		log.Debug().
			Int("index", int(i)).
			Uint16("family", actualFamily).
			Uint32("interfaceIndex", interfaceIndex).
			Uint32("prefixLength", prefixLength).
			Uint8("sitePrefixLength", sitePrefixLength).
			Hex("destPrefix", destPrefix.Data[:8]).
			Hex("nextHop", nextHop.Data[:8]).
			Hex("rawEntry", (*[64]byte)(unsafe.Pointer(entryOffset))[:rawEntrySize]).
			Msg("GetIpForwardTable2: processing route entry")

		// Parse destination address - use actualFamily to determine address type
		destIP, _, err := n.parseSockaddrInet(destPrefix, actualFamily)
		if err != nil {
			log.Debug().
				Int("index", int(i)).
				Err(err).
				Uint16("family", actualFamily).
				Hex("destPrefix", destPrefix.Data[:]).
				Msg("failed to parse destination address, skipping route")
			continue
		}

		// Parse gateway address - use actualFamily
		gatewayIP, _, err := n.parseSockaddrInet(nextHop, actualFamily)
		if err != nil {
			log.Debug().
				Int("index", int(i)).
				Err(err).
				Uint16("family", actualFamily).
				Hex("nextHop", nextHop.Data[:]).
				Msg("failed to parse gateway address, using empty")
			gatewayIP = nil
		}

		// Use the prefix length from IP_ADDRESS_PREFIX for the destination
		destPrefixLen := int(prefixLength)

		var dest string
		if destIP == nil {
			log.Debug().
				Int("index", int(i)).
				Msg("destination IP is nil, skipping route")
			continue
		}

		if destIP.To4() != nil {
			// IPv4
			if destIP.Equal(net.IPv4zero) {
				dest = "0.0.0.0/0"
			} else {
				dest = fmt.Sprintf("%s/%d", destIP.String(), destPrefixLen)
			}
		} else {
			// IPv6
			if destIP.Equal(net.IPv6unspecified) {
				dest = "::/0"
			} else {
				dest = fmt.Sprintf("%s/%d", destIP.String(), destPrefixLen)
			}
		}

		var gateway string
		if gatewayIP == nil {
			if actualFamily == AF_INET {
				gateway = "0.0.0.0"
			} else {
				gateway = "::"
			}
		} else if gatewayIP.To4() != nil {
			if gatewayIP.IsUnspecified() {
				gateway = "0.0.0.0"
			} else {
				gateway = gatewayIP.String()
			}
		} else {
			if gatewayIP.Equal(net.IPv6unspecified) {
				gateway = "::"
			} else {
				gateway = gatewayIP.String()
			}
		}

		iface := fmt.Sprintf("%d", interfaceIndex)
		if name, ok := interfaceMap[interfaceIndex]; ok {
			iface = name
			log.Debug().
				Int("index", int(i)).
				Uint32("interfaceIndex", interfaceIndex).
				Str("interfaceName", name).
				Msg("found interface name in map")
		} else {
			log.Debug().
				Int("index", int(i)).
				Uint32("interfaceIndex", interfaceIndex).
				Interface("interfaceMapKeys", func() []uint32 {
					keys := make([]uint32, 0, len(interfaceMap))
					for k := range interfaceMap {
						keys = append(keys, k)
					}
					return keys
				}()).
				Msg("interface index not found in map, using index as string")
		}

		log.Debug().
			Int("index", int(i)).
			Str("destination", dest).
			Str("gateway", gateway).
			Str("interface", iface).
			Uint32("prefixLength", prefixLength).
			Msg("GetIpForwardTable2: parsed route successfully")

		routes = append(routes, Route{
			Destination: dest,
			Gateway:     gateway,
			Flags:       []string{},
			Interface:   iface,
		})
	}

	log.Debug().
		Int("totalRoutes", len(routes)).
		Uint32("totalEntries", table.NumEntries).
		Msg("GetIpForwardTable2: completed processing")

	return routes, nil
}

// parseSockaddrInet parses a SOCKADDR_INET union into a net.IP
// The family parameter should be trusted (from row.Family which is set correctly by Windows)
func (n *neti) parseSockaddrInet(addr sockaddrInet, family uint16) (net.IP, int, error) {
	switch family {
	case AF_INET:
		// Parse as SOCKADDR_IN
		// Structure layout: family (2 bytes), port (2 bytes), IP (4 bytes), zero padding (8 bytes)
		sa := (*sockaddrIn)(unsafe.Pointer(&addr.Data[0]))
		// Don't check SinFamily - it might be 0 or uninitialized in some cases
		// Trust the family parameter from row.Family instead
		ip := net.IPv4(sa.SinAddr[0], sa.SinAddr[1], sa.SinAddr[2], sa.SinAddr[3])
		return ip, 0, nil

	case AF_INET6:
		// Parse as SOCKADDR_IN6
		// Structure layout: family (2 bytes), port (2 bytes), flowinfo (4 bytes), IP (16 bytes), scope (4 bytes)
		sa6 := (*sockaddrIn6)(unsafe.Pointer(&addr.Data[0]))
		// Don't check Sin6Family - it might be 0 or uninitialized in some cases
		// Trust the family parameter from row.Family instead
		ip := make(net.IP, 16)
		copy(ip, sa6.Sin6Addr[:])
		return ip, 0, nil

	default:
		return nil, 0, errors.Errorf("unsupported address family: %d", family)
	}
}

// IP_ADAPTER_ADDRESSES represents a network adapter
// https://docs.microsoft.com/en-us/windows/win32/api/iptypes/ns-iptypes-ip_adapter_addresses_lh
type ipAdapterAddresses struct {
	Length                uint32
	IfIndex               uint32
	Next                  *ipAdapterAddresses
	AdapterName           *byte
	FirstUnicastAddress   uintptr // PIP_ADAPTER_UNICAST_ADDRESS
	FirstAnycastAddress   uintptr // PIP_ADAPTER_ANYCAST_ADDRESS
	FirstMulticastAddress uintptr // PIP_ADAPTER_MULTICAST_ADDRESS
	FirstDnsServerAddress uintptr // PIP_ADAPTER_DNS_SERVER_ADDRESS
	DnsSuffix             *uint16
	Description           *uint16
	FriendlyName          *uint16
	PhysicalAddress       [8]byte
	PhysicalAddressLength uint32
	Flags                 uint32
	Mtu                   uint32
	IfType                uint32
	OperStatus            uint32
	Ipv6IfIndex           uint32
	ZoneIndices           [16]uint32
	FirstPrefix           uintptr // PIP_ADAPTER_PREFIX
	// ... more fields, but we only need the ones above
}

const (
	// GetAdaptersAddresses flags
	GAA_FLAG_INCLUDE_PREFIX  = 0x0010
	GAA_FLAG_SKIP_ANYCAST    = 0x0002
	GAA_FLAG_SKIP_MULTICAST  = 0x0004
	GAA_FLAG_SKIP_DNS_SERVER = 0x0008
)

// getWindowsInterfaceMap creates a map of interface index to interface name
// Uses native Windows GetAdaptersAddresses API
func (n *neti) getWindowsInterfaceMap() (map[uint32]string, error) {
	interfaceMap := make(map[uint32]string)

	// First call: get the required buffer size
	var size uint32
	ret, _, _ := procGetAdaptersAddresses.Call(
		uintptr(syscall.AF_UNSPEC), // Family: AF_UNSPEC = both IPv4 and IPv6
		uintptr(GAA_FLAG_SKIP_ANYCAST|GAA_FLAG_SKIP_MULTICAST|GAA_FLAG_SKIP_DNS_SERVER),
		0, // Reserved
		0, // AdapterAddresses: nil to get size
		uintptr(unsafe.Pointer(&size)),
	)

	// ERROR_BUFFER_OVERFLOW (122) means we need a larger buffer
	// ERROR_NO_DATA (232) means there are no adapters
	// ERROR_INSUFFICIENT_BUFFER (111) also means we need a larger buffer
	if ret == 232 {
		return interfaceMap, nil
	}
	if ret != 122 && ret != 111 {
		if ret != 0 {
			return nil, errors.Errorf("GetAdaptersAddresses failed with error code: %d", ret)
		}
		// If ret == 0 and size is still 0, there are no adapters
		if size == 0 {
			return interfaceMap, nil
		}
	}

	// Allocate buffer
	buf := make([]byte, size)
	adapter := (*ipAdapterAddresses)(unsafe.Pointer(&buf[0]))

	log.Debug().
		Uint32("bufferSize", size).
		Msg("GetAdaptersAddresses: allocated buffer")

	// Second call: get the actual data
	ret, _, _ = procGetAdaptersAddresses.Call(
		uintptr(syscall.AF_UNSPEC),
		uintptr(GAA_FLAG_SKIP_ANYCAST|GAA_FLAG_SKIP_MULTICAST|GAA_FLAG_SKIP_DNS_SERVER),
		0,
		uintptr(unsafe.Pointer(adapter)),
		uintptr(unsafe.Pointer(&size)),
	)

	if ret != 0 {
		return nil, errors.Errorf("GetAdaptersAddresses failed with error code: %d", ret)
	}

	// DEBUG: Dump raw bytes for testing
	log.Debug().
		Int("rawBytesLen", len(buf)).
		Hex("rawBytes", buf).
		Msg("dumping raw GetAdaptersAddresses bytes for testing")

	// Traverse the linked list
	adapterCount := 0
	for adapter != nil {
		ifIndex := adapter.IfIndex
		if ifIndex != 0 {
			var name string
			if adapter.FriendlyName != nil {
				// Convert UTF-16 to string
				name = windows.UTF16PtrToString(adapter.FriendlyName)
			} else if adapter.AdapterName != nil {
				// Fallback to adapter name if FriendlyName is not available
				name = windows.BytePtrToString(adapter.AdapterName)
			}

			if name != "" {
				interfaceMap[ifIndex] = name
				adapterCount++
				log.Debug().
					Uint32("ifIndex", ifIndex).
					Str("name", name).
					Uint32("length", adapter.Length).
					Uint32("ifType", adapter.IfType).
					Uint32("operStatus", adapter.OperStatus).
					Msg("GetAdaptersAddresses: parsed adapter")
			}
		}

		// Move to next adapter
		if adapter.Next == nil {
			break
		}
		adapter = adapter.Next
	}

	log.Debug().
		Int("adapterCount", adapterCount).
		Int("interfaceMapSize", len(interfaceMap)).
		Interface("interfaceMap", interfaceMap).
		Msg("GetAdaptersAddresses: completed parsing")

	return interfaceMap, nil
}
