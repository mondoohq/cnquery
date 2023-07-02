package networkinterface

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"regexp"
	"strconv"
	"strings"

	"errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/motor"
	"go.mondoo.com/cnquery/motor/platform"
	"go.mondoo.com/cnquery/motor/providers/os"
	"go.mondoo.com/cnquery/motor/providers/os/powershell"
)

var errNoSuchInterface = errors.New("no such network interface")

// mimics https://golang.org/src/net/interface.go to provide a similar api
type Interface struct {
	Index          int              // positive integer that starts at one, zero is never used
	MTU            int              // maximum transmission unit
	Name           string           // e.g., "en0", "lo0", "eth0.100"
	HardwareAddr   net.HardwareAddr // IEEE MAC-48, EUI-48 and EUI-64 form
	Flags          net.Flags        // e.g., FlagUp, FlagLoopback, FlagMulticast
	Addrs          []net.Addr
	MulticastAddrs []net.Addr
}

func New(motor *motor.Motor) *InterfaceResource {
	return &InterfaceResource{
		motor: motor,
	}
}

type InterfaceResource struct {
	motor *motor.Motor
}

func (r *InterfaceResource) Interfaces() ([]Interface, error) {
	pi, err := r.motor.Platform()
	if err != nil {
		return nil, err
	}

	osProvider, isOSProvider := r.motor.Provider.(os.OperatingSystemProvider)

	log.Debug().Strs("families", pi.Family).Msg("check if platform is supported for network interface")
	if r.motor.IsLocalProvider() {
		handler := &GoNativeInterfaceHandler{}
		return handler.Interfaces()
	} else if isOSProvider && pi.Name == "macos" {
		handler := &MacOSInterfaceHandler{
			provider: osProvider,
		}
		return handler.Interfaces()
	} else if isOSProvider && pi.IsFamily(platform.FAMILY_LINUX) {
		log.Debug().Msg("detected linux platform")
		handler := &LinuxInterfaceHandler{
			provider: osProvider,
		}
		return handler.Interfaces()
	} else if isOSProvider && pi.Name == "windows" {
		handler := &WindowsInterfaceHandler{
			provider: osProvider,
		}
		return handler.Interfaces()
	}

	return nil, errors.New("interfaces does not support platform: " + pi.Name)
}

func (r *InterfaceResource) InterfaceByName(name string) (*Interface, error) {
	ifaces, err := r.Interfaces()
	if err != nil {
		return nil, err
	}

	for i := range ifaces {
		if ifaces[i].Name == name {
			return &ifaces[i], nil
		}
	}
	return nil, errNoSuchInterface
}

type GoNativeInterfaceHandler struct{}

func (i *GoNativeInterfaceHandler) Interfaces() ([]Interface, error) {
	var goInterfaces []net.Interface
	var err error
	if goInterfaces, err = net.Interfaces(); err != nil {
		return nil, errors.Join(err, errors.New("failed to load network interfaces"))
	}

	ifaces := make([]Interface, len(goInterfaces))
	for i := range goInterfaces {

		addrs, err := goInterfaces[i].Addrs()
		if err != nil {
			log.Debug().Err(err).Str("iface", goInterfaces[i].Name).Msg("could not retrieve ip addresses")
		}
		multicastAddrs, err := goInterfaces[i].MulticastAddrs()
		if err != nil {
			log.Debug().Err(err).Str("iface", goInterfaces[i].Name).Msg("could not retrieve multicast addresses")
		}

		ifaces[i] = Interface{
			Name:           goInterfaces[i].Name,
			Index:          goInterfaces[i].Index,
			MTU:            goInterfaces[i].MTU,
			HardwareAddr:   goInterfaces[i].HardwareAddr,
			Flags:          goInterfaces[i].Flags,
			Addrs:          addrs,
			MulticastAddrs: multicastAddrs,
		}
	}

	return ifaces, nil
}

type LinuxInterfaceHandler struct {
	provider os.OperatingSystemProvider
}

func (i *LinuxInterfaceHandler) Interfaces() ([]Interface, error) {
	// TODO: support extracting the information via /sys/class/net/, /proc/net/fib_trie
	// fetch all network adapter via ip addr show
	cmd, err := i.provider.RunCommand("ip -o addr show")
	if err != nil {
		return nil, errors.Join(err, errors.New("could not fetch macos network adapter"))
	}

	return i.ParseIpAddr(cmd.Stdout)
}

func (i *LinuxInterfaceHandler) ParseIpAddr(r io.Reader) ([]Interface, error) {
	interfaces := map[string]Interface{}

	scanner := bufio.NewScanner(r)
	ipaddrParse := regexp.MustCompile(`^(\d):\s([\w\d\./\:]+)\s*(inet|inet6)\s([\w\d\./\:]+)\s(.*)$`)

	for scanner.Scan() {
		line := scanner.Text()

		m := ipaddrParse.FindStringSubmatch(line)

		// check that we have a match
		if len(m) < 4 {
			return nil, fmt.Errorf("cannot parse ip: %s", line)
		}

		name := m[2]

		idx, err := strconv.Atoi(strings.TrimSpace(m[1]))
		if err != nil {
			log.Warn().Err(err).Msg("could not parse ip addr idx")
			continue
		}

		inet, ok := interfaces[name]
		if !ok {
			inet = Interface{
				Index: idx,
				Name:  name,
			}
		}

		var ip net.IP
		if m[3] == "inet" {
			ipv4Addr, _, err := net.ParseCIDR(m[4])
			if err != nil {
				log.Error().Err(err).Msg("could not parse ipv4")
			}

			ip = ipv4Addr
		} else if m[3] == "inet6" {
			ipv6Addr, _, err := net.ParseCIDR(m[4])
			if err != nil {
				log.Error().Err(err).Msg("could not parse ipv6")
			}
			ip = ipv6Addr
		}

		inet.Addrs = append(inet.Addrs, &ipAddr{IP: ip})

		var flags net.Flags
		flags |= net.FlagUp

		if strings.Contains(m[5], "host") {
			flags |= net.FlagLoopback
		} else {
			flags |= net.FlagBroadcast
		}

		inet.Flags = flags

		interfaces[name] = inet
	}

	res := []Interface{}
	for i := range interfaces {
		res = append(res, interfaces[i])
	}

	return res, nil
}

type MacOSInterfaceHandler struct {
	provider os.OperatingSystemProvider
}

func (i *MacOSInterfaceHandler) Interfaces() ([]Interface, error) {
	// fetch all network adapter
	cmd, err := i.provider.RunCommand("ifconfig")
	if err != nil {
		return nil, errors.Join(err, errors.New("could not fetch macos network adapter"))
	}

	return i.ParseMacOS(cmd.Stdout)
}

var (
	IfconfigInterfaceLine = regexp.MustCompile(`^(.*):\ flags=(\d*)\<(.*)>\smtu\s(\d*)$`)
	IfconfigInetLine      = regexp.MustCompile(`^\s+inet(\d*)\s([\w\d.:%]+)\s`)
)

func (i *MacOSInterfaceHandler) ParseMacOS(r io.Reader) ([]Interface, error) {
	interfaces := []Interface{}
	ifIndex := -1
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		line := scanner.Text()

		// new interface
		m := IfconfigInterfaceLine.FindStringSubmatch(line)
		if len(m) > 0 {
			var mtu int
			var err error
			mtu, err = strconv.Atoi(strings.TrimSpace(m[4]))
			if err != nil {
				return nil, errors.Join(err, errors.New("cannot parse macos ifconfig mtu"))
			}

			var flags net.Flags
			if len(m[3]) > 0 {
				ifConfigFlags := strings.Split(m[3], ",")
				for i := range ifConfigFlags {
					switch strings.ToLower(ifConfigFlags[i]) {
					case "up":
						flags |= net.FlagUp
					case "broadcast":
						flags |= net.FlagBroadcast
					case "multicast":
						flags |= net.FlagMulticast
					case "loopback":
						flags |= net.FlagLoopback
					case "pointtopoint":
						flags |= net.FlagPointToPoint
					}
				}
			}

			i := Interface{
				Index: ifIndex + 2,
				Name:  m[1],
				MTU:   mtu,
				Flags: flags,
			}

			interfaces = append(interfaces, i)
			ifIndex++
		}

		// parse mac address
		if strings.HasPrefix(line, "	ether") {
			macaddress := strings.TrimSpace(strings.TrimPrefix(line, "	ether"))
			mac, err := net.ParseMAC(macaddress)
			if err != nil {
				return nil, err
			}
			interfaces[ifIndex].HardwareAddr = mac
		}

		m = IfconfigInetLine.FindStringSubmatch(line)
		if len(m) > 0 {
			ip := parseIpAddr(m[2])
			if ip != nil {
				interfaces[ifIndex].Addrs = append(interfaces[ifIndex].Addrs, &ipAddr{IP: ip})
			}
		}

	}
	return interfaces, nil
}

type WindowsInterface struct {
	Name          string `json:"Name"`
	IfIndex       int    `json:"ifIndex"`
	InterfaceType int    `json:"InterfaceType"`
	Status        string `json:"Status"`
	MacAddress    string `json:"MacAddress"`
}

type WindowsNetIp struct {
	InterfaceAlias string  `json:"InterfaceAlias"`
	IPv4Address    *string `json:"IPv4Address"`
	IPv6Address    *string `json:"IPv6Address"`
}

const (
	WinGetNetAdapter   = "Get-NetAdapter | Select-Object -Property Name, ifIndex, InterfaceType, InterfaceDescription, Status, State, MacAddress, LinkSpeed, ReceiveLinkSpeed, TransmitLinkSpeed, Virtual | ConvertTo-Json"
	WinGetNetIPAddress = "Get-NetIPAddress | Select-Object -Property IPv6Address, IPv4Address, InterfaceAlias | ConvertTo-Json"
)

const (
	IF_TYPE_OTHER              = 1
	IF_TYPE_ETHERNET_CSMACD    = 6
	IF_TYPE_ISO88025_TOKENRING = 9
	IF_TYPE_PPP                = 23
	IF_TYPE_SOFTWARE_LOOPBACK  = 24
	IF_TYPE_ATM                = 37
	IF_TYPE_IEEE80211          = 71
	IF_TYPE_TUNNEL             = 131
	IF_TYPE_IEEE1394           = 144
)

// derived from https://golang.org/src/net/interface_windows.go
func windowsInterfaceFlags(status string, ifType int) net.Flags {
	var flags net.Flags
	if status == "Up" {
		flags |= net.FlagUp
	}

	switch ifType {
	case IF_TYPE_ETHERNET_CSMACD, IF_TYPE_ISO88025_TOKENRING, IF_TYPE_IEEE80211, IF_TYPE_IEEE1394:
		flags |= net.FlagBroadcast | net.FlagMulticast
	case IF_TYPE_PPP, IF_TYPE_TUNNEL:
		flags |= net.FlagPointToPoint | net.FlagMulticast
	case IF_TYPE_SOFTWARE_LOOPBACK:
		flags |= net.FlagLoopback | net.FlagMulticast
	case IF_TYPE_ATM:
		flags |= net.FlagBroadcast | net.FlagPointToPoint | net.FlagMulticast
	}
	return flags
}

type ipAddr struct {
	IP net.IP
}

// name of the network (for example, "tcp", "udp")
func (a *ipAddr) Network() string {
	return "tcp"
}

// string form of address (for example, "192.0.2.1:25", "[2001:db8::1]:80")
func (a *ipAddr) String() string {
	return a.IP.String()
}

func parseIpAddr(ip string) net.IP {
	// filter network id https://sid-500.com/2017/01/10/cisco-ipv6-link-local-adressen-und-router-advertisements/
	// "fe80::ed94:1267:afb5:bb7%6" becomes "fe80::ed94:1267:afb5:bb7"
	m := strings.Split(ip, "%")
	return net.ParseIP(m[0])
}

func filterWinIpByInterface(iName string, ips []WindowsNetIp) []net.Addr {
	addrs := []net.Addr{}

	for i := range ips {
		if ips[i].InterfaceAlias == iName {
			var ip net.IP
			if ips[i].IPv4Address != nil {
				ip = parseIpAddr(*ips[i].IPv4Address)
			} else if ips[i].IPv6Address != nil {
				ip = parseIpAddr(*ips[i].IPv6Address)
			}
			if ip != nil {
				addrs = append(addrs, &ipAddr{IP: ip})
			}
		}
	}

	return addrs
}

type WindowsInterfaceHandler struct {
	provider os.OperatingSystemProvider
}

func (i *WindowsInterfaceHandler) Interfaces() ([]Interface, error) {
	// fetch all network adapter
	cmd, err := i.provider.RunCommand(powershell.Wrap(WinGetNetAdapter))
	if err != nil {
		return nil, errors.Join(err, errors.New("could not fetch windows network adapter"))
	}
	winAdapter, err := i.ParseNetAdapter(cmd.Stdout)
	if err != nil {
		return nil, errors.Join(err, errors.New("could not parse windows network adapter list"))
	}

	// fetch all ip addresses
	cmd, err = i.provider.RunCommand(powershell.Wrap(WinGetNetIPAddress))
	if err != nil {
		return nil, errors.Join(err, errors.New("could not fetch windows ip addresses"))
	}
	winIpAddresses, err := i.ParseNetIpAdresses(cmd.Stdout)
	if err != nil {
		return nil, errors.Join(err, errors.New("could not parse windows ip addresses"))
	}

	// map information together
	interfaces := make([]Interface, len(winAdapter))
	for i := range winAdapter {
		mac, err := net.ParseMAC(winAdapter[i].MacAddress)
		if err != nil {
			return nil, err
		}

		interfaces[i] = Interface{
			Name:         winAdapter[i].Name,
			Index:        winAdapter[i].IfIndex,
			HardwareAddr: mac,
			Flags:        windowsInterfaceFlags(winAdapter[i].Status, winAdapter[i].InterfaceType),
			Addrs:        filterWinIpByInterface(winAdapter[i].Name, winIpAddresses),
		}
	}

	return interfaces, nil
}

func (i *WindowsInterfaceHandler) ParseNetAdapter(r io.Reader) ([]WindowsInterface, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	var winInterfaces []WindowsInterface
	err = json.Unmarshal(data, &winInterfaces)
	if err != nil {

		// try again without array (powershell returns single values different)
		var winInterface WindowsInterface
		err = json.Unmarshal(data, &winInterface)
		if err != nil {
			return nil, err
		}

		return []WindowsInterface{winInterface}, nil
	}
	return winInterfaces, nil
}

func (i *WindowsInterfaceHandler) ParseNetIpAdresses(r io.Reader) ([]WindowsNetIp, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	var winNetIps []WindowsNetIp
	err = json.Unmarshal(data, &winNetIps)
	if err != nil {
		return nil, err
	}
	return winNetIps, nil
}
