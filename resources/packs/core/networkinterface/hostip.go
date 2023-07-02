package networkinterface

import (
	"fmt"
	"net"
	"sort"

	"errors"
	"github.com/rs/zerolog/log"
)

func filterNetworkInterface(interfaces []Interface, flagFilter func(flags net.Flags) bool) []Interface {
	i := []Interface{}
	for _, v := range interfaces {
		if flagFilter(v.Flags) {
			i = append(i, v)
		}
	}
	return i
}

// byIfaceIndex Interface by its index
type byIfaceIndex []Interface

func (iface byIfaceIndex) Len() int           { return len(iface) }
func (iface byIfaceIndex) Less(i, j int) bool { return iface[i].Index < iface[j].Index }
func (iface byIfaceIndex) Swap(i, j int)      { iface[i], iface[j] = iface[j], iface[i] }

// HostIP extracts the best-guess for the IP of the host
// It will search ip v4 first and fallback to v6
func HostIP(interfaces []Interface) (ip string, err error) {
	log.Debug().Int("interfaces", len(interfaces)).Msg("search ip")
	// filter interfaces that are not up or a loopback/p2p interface
	interfaces = filterNetworkInterface(interfaces, func(flags net.Flags) bool {
		if (flags&net.FlagUp != 0) &&
			(flags&net.FlagLoopback == 0) &&
			(flags&net.FlagPointToPoint == 0) {
			return true
		}
		return false
	})

	// sort interfaces by its index
	sort.Sort(byIfaceIndex(interfaces))

	var foundIPv4 net.IP
	foundIPsv6 := []net.IP{}

	// search for IPv4
	for _, i := range interfaces {
		addrs := i.Addrs
		for _, addr := range addrs {
			var foundIPv6 net.IP
			switch v := addr.(type) {
			case *net.IPAddr:
				foundIPv4 = v.IP.To4()
				foundIPv6 = v.IP.To16()
			case *net.IPNet:
				foundIPv4 = v.IP.To4()
				foundIPv6 = v.IP.To16()
			case *ipAddr:
				foundIPv4 = v.IP.To4()
				foundIPv6 = v.IP.To16()
			}

			if foundIPv4 != nil {
				return foundIPv4.String(), nil
			}
			if foundIPv6 != nil {
				foundIPsv6 = append(foundIPsv6, foundIPv6)
			}
		}
	}

	// search for IPv6
	if len(foundIPsv6) > 0 {
		return foundIPsv6[0].String(), nil
	}

	return "", fmt.Errorf("no IP address found")
}

// GetOutboundIP returns the local IP that is used for outbound connections
// It does not establish a real connection and the destination does not need to valid.
// Since its using udp protocol (unlike TCP) a handshake nor connection is required,
// / then it gets the local up address if it would connect to that target
// conn.LocalAddr().String() returns the local ip and port
//
// # NOTE be aware that this code does not work on remote targets
//
// @see this approach is derived from https://stackoverflow.com/a/37382208
func GetOutboundIP() (net.IP, error) {
	conn, err := net.Dial("udp", "1.1.1.1:80")
	if err != nil {
		return nil, errors.Join(err, errors.New("could not determine outbound ip"))
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	if localAddr == nil {
		return nil, errors.New("could not determine outbound ip")
	}

	return localAddr.IP, nil
}
