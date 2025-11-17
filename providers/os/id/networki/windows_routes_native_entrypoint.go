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
	AF_INET  = 2  // IPv4
	AF_INET6 = 23 // IPv6

	// Windows error codes for GetAdaptersAddresses
	ERROR_NO_DATA             = 232
	ERROR_BUFFER_OVERFLOW     = 122
	ERROR_INSUFFICIENT_BUFFER = 111

	// GAA_FLAG_* are used as filter for the Windows GetAdaptersAddresses API call
	GAA_FLAG_INCLUDE_PREFIX  = 0x0010
	GAA_FLAG_SKIP_ANYCAST    = 0x0002
	GAA_FLAG_SKIP_MULTICAST  = 0x0004
	GAA_FLAG_SKIP_DNS_SERVER = 0x0008
)

var (
	iphlpapi                 = windows.NewLazySystemDLL("iphlpapi.dll")
	procGetIpForwardTable2   = iphlpapi.NewProc("GetIpForwardTable2")
	procFreeMibTable         = iphlpapi.NewProc("FreeMibTable")
	procGetAdaptersAddresses = iphlpapi.NewProc("GetAdaptersAddresses")
)

// detectWindowsRoutes detects network routes on Windows using native IP Helper API
func (n *neti) detectWindowsRoutes() ([]Route, error) {
	routes, err := n.detectWindowsRoutesViaGetIpForwardTable()
	if err == nil && len(routes) > 0 {
		return routes, nil
	}
	log.Debug().Err(err).Msg("native Windows API failed")
	// Fallback to PowerShell if native APIs fail
	routes, err = n.detectWindowsRoutesViaPowerShell()
	if err == nil && len(routes) > 0 {
		return routes, nil
	}
	log.Debug().Err(err).Int("routeCount", len(routes)).Msg("PowerShell Get-NetRoute failed, trying netstat")

	// fallback to netstat
	return n.detectWindowsRoutesViaNetstat()
}

// ipv4Address represents an IPv4 socket address
type ipv4Address struct {
	SinFamily uint16
	SinPort   [2]byte
	SinAddr   [4]byte
	SinZero   [8]byte
}

// ipv6Address represents an IPv6 socket address
type ipv6Address struct {
	Sin6Family   uint16
	Sin6Port     [2]byte
	Sin6Flowinfo uint32
	Sin6Addr     [16]byte
	Sin6ScopeId  uint32
}

// socketInetAddress can represent either IPv4 or IPv6 address
type socketInetAddress struct {
	Data [28]byte
}

// mibIpForwardRow2 structure stores information about an IP route entry.
// https://learn.microsoft.com/en-us/windows/win32/api/netioapi/ns-netioapi-mib_ipforward_row2
type mibIpForwardRow2 struct {
	// NET_LUID InterfaceLuid (8 bytes) - we skip this
	_                    [8]byte
	InterfaceIndex       uint32
	DestinationPrefix    socketInetAddress
	_                    [4]byte
	NextHop              socketInetAddress
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

type mibIpForwardTable2 struct {
	NumEntries uint32
	Table      [1]mibIpForwardRow2
}

// detectWindowsRoutesViaGetIpForwardTable uses GetIpForwardTable2 to get routes (IPv4 + IPv6)
func (n *neti) detectWindowsRoutesViaGetIpForwardTable() ([]Route, error) {
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
	defer procFreeMibTable.Call(uintptr(unsafe.Pointer(table)))

	if ret != 0 {
		return nil, errors.Errorf("GetIpForwardTable2 failed with error code: %d", ret)
	}
	if table == nil || table.NumEntries == 0 {
		return []Route{}, nil
	}

	rowSize := unsafe.Sizeof(mibIpForwardRow2{})
	tableStart := uintptr(unsafe.Pointer(&table.Table[0]))
	var routes []Route

	for i := uint32(0); i < table.NumEntries; i++ {
		entryOffset := tableStart + uintptr(i)*rowSize

		// Manually read fields from correct offsets
		// Structure layout (from raw bytes analysis):
		// - Offset 0-7: InterfaceLuid (8 bytes)
		// - Offset 8-11: padding (4 bytes)
		// - Offset 12-15: InterfaceIndex (4 bytes)
		// - Offset 16-43: DestinationPrefix SOCKADDR_INET (28 bytes)
		// - Offset 44-47: PrefixLength (4 bytes)
		// - Offset 48-75: NextHop SOCKADDR_INET (28 bytes)
		// - Offset 76: SitePrefixLength (1 byte)
		// - Offset 77-79: Padding (3 bytes)
		// - Offset 80+: Rest of fields

		interfaceIndexOffset := entryOffset + 12
		interfaceIndex := *(*uint32)(unsafe.Pointer(interfaceIndexOffset))

		destPrefixOffset := entryOffset + 16
		actualFamily := *(*uint16)(unsafe.Pointer(destPrefixOffset))

		destPrefix := socketInetAddress{Data: *(*[28]byte)(unsafe.Pointer(destPrefixOffset))}
		prefixLength := *(*uint32)(unsafe.Pointer(entryOffset + 44))
		nextHop := socketInetAddress{Data: *(*[28]byte)(unsafe.Pointer(entryOffset + 48))}

		if actualFamily != AF_INET && actualFamily != AF_INET6 {
			continue
		}

		destIP, _, err := n.parseSockaddrInet(destPrefix, actualFamily)
		if err != nil {
			continue
		}

		gatewayIP, _, err := n.parseSockaddrInet(nextHop, actualFamily)
		if err != nil {
			gatewayIP = nil
		}

		dest := n.formatDestination(destIP, int(prefixLength))
		if dest == "" {
			continue
		}

		gateway := n.formatGateway(gatewayIP, actualFamily)
		iface := n.getInterfaceName(interfaceIndex, interfaceMap)

		routes = append(routes, Route{
			Destination: dest,
			Gateway:     gateway,
			Flags:       []string{},
			Interface:   iface,
		})
	}

	return routes, nil
}

// formatDestination formats a destination IP address with prefix length
func (n *neti) formatDestination(destIP net.IP, prefixLen int) string {
	if destIP == nil {
		return ""
	}
	if destIP.To4() != nil {
		if destIP.Equal(net.IPv4zero) {
			return "0.0.0.0/0"
		}
		return fmt.Sprintf("%s/%d", destIP.String(), prefixLen)
	}

	if destIP.Equal(net.IPv6unspecified) {
		return "::/0"
	}
	return fmt.Sprintf("%s/%d", destIP.String(), prefixLen)
}

func (n *neti) formatGateway(gatewayIP net.IP, family uint16) string {
	if gatewayIP == nil {
		if family == AF_INET {
			return "0.0.0.0"
		}
		return "::"
	}
	if gatewayIP.To4() != nil {
		if gatewayIP.IsUnspecified() {
			return "0.0.0.0"
		}
		return gatewayIP.String()
	}
	if gatewayIP.Equal(net.IPv6unspecified) {
		return "::"
	}
	return gatewayIP.String()
}

// getInterfaceName returns the interface name from the map, or the index as string if not found
func (n *neti) getInterfaceName(interfaceIndex uint32, interfaceMap map[uint32]string) string {
	if name, ok := interfaceMap[interfaceIndex]; ok {
		return name
	}
	return fmt.Sprintf("%d", interfaceIndex)
}

// parseSockaddrInet parses a SOCKADDR_INET union into a net.IP
func (n *neti) parseSockaddrInet(addr socketInetAddress, family uint16) (net.IP, int, error) {
	switch family {
	case AF_INET:
		sa := (*ipv4Address)(unsafe.Pointer(&addr.Data[0]))
		return net.IPv4(sa.SinAddr[0], sa.SinAddr[1], sa.SinAddr[2], sa.SinAddr[3]), 0, nil
	case AF_INET6:
		sa6 := (*ipv6Address)(unsafe.Pointer(&addr.Data[0]))
		ip := make(net.IP, 16)
		copy(ip, sa6.Sin6Addr[:])
		return ip, 0, nil
	default:
		return nil, 0, errors.Errorf("unsupported address family: %d", family)
	}
}

// https://learn.microsoft.com/en-us/windows/win32/api/iptypes/ns-iptypes-ip_adapter_addresses_lh
type ipAdapterAddresses struct {
	Length                uint32
	IfIndex               uint32
	Next                  *ipAdapterAddresses
	AdapterName           *byte
	FirstUnicastAddress   uintptr
	FirstAnycastAddress   uintptr
	FirstMulticastAddress uintptr
	FirstDnsServerAddress uintptr
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
	FirstPrefix           uintptr
}

// getWindowsInterfaceMap creates a map of interface index to interface name
// Uses native Windows GetAdaptersAddresses API https://learn.microsoft.com/en-us/windows/win32/api/iphlpapi/nf-iphlpapi-getadaptersaddresses
func (n *neti) getWindowsInterfaceMap() (map[uint32]string, error) {
	interfaceMap := make(map[uint32]string)

	var size uint32
	ret, _, err := procGetAdaptersAddresses.Call(
		uintptr(syscall.AF_UNSPEC), // Family: AF_UNSPEC = both IPv4 and IPv6
		uintptr(GAA_FLAG_SKIP_ANYCAST|GAA_FLAG_SKIP_MULTICAST|GAA_FLAG_SKIP_DNS_SERVER),
		0,
		0,
		uintptr(unsafe.Pointer(&size)),
	)
	if err != syscall.Errno(0) {
		return nil, errors.Errorf("GetAdaptersAddresses (buffer size) failed: %v", err)
	}

	if ret == ERROR_NO_DATA || (ret == 0 && size == 0) {
		return interfaceMap, nil
	}
	if ret != 0 && ret != ERROR_BUFFER_OVERFLOW && ret != ERROR_INSUFFICIENT_BUFFER {
		return nil, errors.Errorf("GetAdaptersAddresses failed with error code: %d", ret)
	}

	buf := make([]byte, size)
	adapter := (*ipAdapterAddresses)(unsafe.Pointer(&buf[0]))

	ret, _, err = procGetAdaptersAddresses.Call(
		uintptr(syscall.AF_UNSPEC),
		uintptr(GAA_FLAG_SKIP_ANYCAST|GAA_FLAG_SKIP_MULTICAST|GAA_FLAG_SKIP_DNS_SERVER),
		0,
		uintptr(unsafe.Pointer(adapter)),
		uintptr(unsafe.Pointer(&size)),
	)
	if err != syscall.Errno(0) {
		return nil, errors.Errorf("GetAdaptersAddresses failed: %v", err)
	}

	if ret != 0 {
		return nil, errors.Errorf("GetAdaptersAddresses failed with error code: %d", ret)
	}

	for adapter != nil {
		if adapter.IfIndex != 0 {
			if name := n.getAdapterName(adapter); name != "" {
				interfaceMap[adapter.IfIndex] = name
			}
		}
		if adapter.Next == nil {
			break
		}
		adapter = adapter.Next
	}

	return interfaceMap, nil
}

// getAdapterName extracts the adapter name from IP_ADAPTER_ADDRESSES structure
func (n *neti) getAdapterName(adapter *ipAdapterAddresses) string {
	if adapter.FriendlyName != nil {
		return windows.UTF16PtrToString(adapter.FriendlyName)
	}
	if adapter.AdapterName != nil {
		return windows.BytePtrToString(adapter.AdapterName)
	}
	return ""
}
