package core

import (
	"bufio"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/netip"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core/lsof"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/motor/providers/os/powershell"
	"go.mondoo.com/cnquery/resources/packs/core/ports"
)

func (s *mqlPorts) id() (string, error) {
	return "ports", nil
}

func (p *mqlPorts) GetList() ([]interface{}, error) {
	pf, err := p.MotorRuntime.Motor.Platform()
	if err != nil {
		return nil, err
	}

	switch {
	case pf.IsFamily("linux"):
		return p.listLinux()
	case pf.IsFamily("windows"):
		return p.listWindows()

	case pf.IsFamily("darwin") || pf.Name == "freebsd":
		// both macOS and FreeBSD support lsof
		// FreeBSD may need an installation via `pkg install sysutils/lsof`
		return p.listMacos()
	default:
		return nil, errors.New("could not detect suitable ports manager for platform: " + pf.Name)
	}
}

func (p *mqlPorts) GetListening() ([]interface{}, error) {
	all, err := p.GetList()
	if err != nil {
		return all, err
	}

	res := []interface{}{}
	for i := range all {
		cur := all[i]
		port := cur.(Port)
		state, err := port.State()
		if err != nil {
			return nil, err
		}
		if state == "listen" {
			res = append(res, cur)
		}
	}

	return res, nil
}

// Linux Implementation

var reLinuxProcNet = regexp.MustCompile(
	"^\\s*\\d+: " +
		"([0-9A-F]+):([0-9A-F]+) " + // local_address
		"([0-9A-F]+):([0-9A-F]+) " + // rem_address
		"([0-9A-F]+) " + // state
		"[^ ]+:[^ ]+ " + // tx/rx
		"[^ ]+:[^ ]+ " + // tr/tm
		"[^ ]+\\s+" + // retrnsmt
		"(\\d+)\\s+" + // uid
		"\\d+\\s+" + // timeout
		"(\\d+)\\s+" + // inode
		"", // lots of other stuff if we want it...
)

// "lrwx------ 1 0 0 64 Dec  6 13:56 /proc/1/fd/12 -> socket:[37364]"
var reFindSockets = regexp.MustCompile(
	"^[lrwx-]+\\s+" +
		"\\d+\\s+" +
		"\\d+\\s+" + // uid
		"\\d+\\s+" + // gid
		"\\d+\\s+" +
		"[^ ]+\\s+" + // month, e.g. Dec
		"\\d+\\s+" + // day
		"\\d+:\\d+\\s+" + // time
		"/proc/(\\d+)/fd/\\d+\\s+" + // path
		"->\\s+" +
		".*socket:\\[(\\d+)\\].*\\s*") // target

var TCP_STATES = map[int64]string{
	1:  "established",
	2:  "syn sent",
	3:  "syn recv",
	4:  "fin wait1",
	5:  "fin wait2",
	6:  "time wait",
	7:  "close",
	8:  "close wait",
	9:  "last ack",
	10: "listen",
	11: "closing",
	12: "new syn recv",
}

func hex2ipv4(s string) (string, error) {
	a, err := strconv.ParseUint(s[0:2], 16, 0)
	if err != nil {
		return "", err
	}

	b, err := strconv.ParseUint(s[2:4], 16, 0)
	if err != nil {
		return "", err
	}

	c, err := strconv.ParseUint(s[4:6], 16, 0)
	if err != nil {
		return "", err
	}

	d, err := strconv.ParseUint(s[6:8], 16, 0)
	if err != nil {
		return "", err
	}

	return (strconv.FormatUint(d, 10) + "." +
		strconv.FormatUint(c, 10) + "." +
		strconv.FormatUint(b, 10) + "." +
		strconv.FormatUint(a, 10)), nil
}

func hex2ipv6(s string) (string, error) {
	networkEndian := ipv6EndianTranslation(s)
	ipBytes, err := hex.DecodeString(networkEndian)
	if err != nil {
		return "", err
	}

	var ipBytes16 [16]byte

	copy(ipBytes16[:], ipBytes)
	ip := netip.AddrFrom16(ipBytes16)

	if ip.Next().Is6() {
		return ip.String(), nil
	} else {
		return "", err
	}
}

func ipv6EndianTranslation(s string) string {
	var nativeEndianness binary.ByteOrder

	buf := [2]byte{}
	*(*uint16)(unsafe.Pointer(&buf[0])) = uint16(0xABCD)

	switch buf {
	case [2]byte{0xCD, 0xAB}:
		nativeEndianness = binary.LittleEndian
	case [2]byte{0xAB, 0xCD}:
		nativeEndianness = binary.BigEndian
	default:
		panic("neither little nor big endian detected...")
	}

	if nativeEndianness == binary.BigEndian {
		return s
	}

	if len(s) != 32 {
		// not an IPv6 address in hex format
		return ""
	}

	// read 8 bytes at a time and little-to-big byte swap
	// Ex: fe80:0000:0000:0000:5578:afa9:4caf:27a1 becomes
	//     0000:80fe:0000:0000:a9af:7855:a127:af4c
	swappedBytes := make([]byte, len(s))
	for i := 0; i < len(s); i += 8 {
		swappedBytes[i] = s[i+6]
		swappedBytes[i+1] = s[i+7]
		swappedBytes[i+2] = s[i+4]
		swappedBytes[i+3] = s[i+5]

		swappedBytes[i+4] = s[i+2]
		swappedBytes[i+5] = s[i+3]
		swappedBytes[i+6] = s[i+0]
		swappedBytes[i+7] = s[i+1]
	}

	return string(swappedBytes)
}

func (p *mqlPorts) users() (map[int64]User, error) {
	obj, err := p.MotorRuntime.CreateResource("users")
	if err != nil {
		return nil, err
	}
	users := obj.(Users)

	_, err = users.List()
	if err != nil {
		return nil, err
	}

	c, ok := users.MqlResource().Cache.Load("_map")
	if !ok {
		return nil, errors.New("cannot get map of users")
	}
	userNameMap := c.Data.(map[string]User)
	userUidMap := make(map[int64]User, len(userNameMap))
	for name, user := range userNameMap {
		uid, err := user.Uid()
		if err != nil {
			return nil, errors.New("failed to look up user uid for '" + name + "'")
		}
		userUidMap[uid] = user
	}

	return userUidMap, nil
}

func (p *mqlPorts) processesBySocket() (map[int64]Process, error) {
	// Prerequisites: processes
	obj, err := p.MotorRuntime.CreateResource("processes")
	if err != nil {
		return nil, err
	}
	processes := obj.(Processes)

	_, err = processes.List()
	if err != nil {
		return nil, err
	}

	c, ok := processes.MqlResource().Cache.Load("_socketsMap")
	if !ok {
		return nil, errors.New("cannot get map of processes (and their sockets)")
	}

	res := c.Data.(map[int64]Process)
	err = nil
	if c.Error != nil {
		err = errors.New("cannot read related processess: " + c.Error.Error())
	}

	if len(res) == 0 {
		processesByPid, err := p.processesByPid()
		if err != nil {
			return nil, err
		}
		osProvider, err := osProvider(p.MotorRuntime.Motor)
		if err != nil {
			return nil, err
		}
		c, err := osProvider.RunCommand("find /proc -maxdepth 4 -path '/proc/*/fd/*' -exec ls -n {} \\;")
		if err != nil {
			return nil, fmt.Errorf("processes> could not run command: %v", err)
		}

		processesBySocket := map[int64]Process{}
		scanner := bufio.NewScanner(c.Stdout)
		for scanner.Scan() {
			line := scanner.Text()
			pid, inode, err := parseLinuxFindLine(line)
			if err != nil || (pid == 0 && inode == 0) {
				continue
			}

			processesBySocket[inode] = processesByPid[pid]
		}
		processes.MqlResource().Cache.Store("_socketsMap", &resources.CacheEntry{Data: processesBySocket, Error: nil})
		res = processesBySocket
	}

	return res, err
}

func parseLinuxFindLine(line string) (int64, int64, error) {
	if strings.HasSuffix(line, "Permission denied") || strings.HasSuffix(line, "No such file or directory") {
		return 0, 0, nil
	}

	m := reFindSockets.FindStringSubmatch(line)
	if len(m) == 0 {
		return 0, 0, nil
	}

	pid, err := strconv.ParseInt(m[1], 10, 64)
	if err != nil {
		log.Error().Err(err).Msg("cannot parse unix pid " + m[1])
		return 0, 0, err
	}

	inode, err := strconv.ParseInt(m[2], 10, 64)
	if err != nil {
		log.Error().Err(err).Msg("cannot parse socket inode " + m[2])
		return 0, 0, err
	}

	return pid, inode, nil
}

// See:
// - socket/address parsing: https://wiki.christophchamp.com/index.php?title=Unix_sockets
func (p *mqlPorts) parseProcNet(path string, protocol string, users map[int64]User, getProcess func(int64) *resources.CacheEntry) ([]interface{}, error) {
	osProvider, err := osProvider(p.MotorRuntime.Motor)
	if err != nil {
		return nil, err
	}

	fs := osProvider.FS()
	stat, err := fs.Stat(path)
	if err != nil {
		return nil, errors.New("cannot access stat for " + path)
	}
	if stat.IsDir() {
		return nil, errors.New("something is wrong, looks like " + path + " is a folder")
	}

	fi, err := fs.Open(path)
	if err != nil {
		return nil, err
	}
	defer fi.Close()

	var res []interface{}
	scanner := bufio.NewScanner(fi)
	for scanner.Scan() {
		line := scanner.Text()

		port, err := parseProcNetLine(line)
		if err != nil {
			return nil, fmt.Errorf("failed to parse proc net line: %v", err)
		}
		if port == nil {
			continue
		}

		obj, err := p.MotorRuntime.CreateResource("port",
			"protocol", protocol,
			"port", port.Port,
			"address", port.Address,
			"user", users[port.Uid],
			"process", nil,
			"state", port.State,
			"remoteAddress", port.RemoteAddress,
			"remotePort", port.RemotePort,
		)
		if err != nil {
			return nil, err
		}

		obj.MqlResource().Cache.Store("process", getProcess(port.Inode))

		res = append(res, obj)
	}

	return res, nil
}

type procNetPort struct {
	Address       string
	Port          int64
	RemoteAddress string
	RemotePort    int64
	State         string
	Uid           int64
	Inode         int64
}

func parseProcNetLine(line string) (*procNetPort, error) {
	m := reLinuxProcNet.FindStringSubmatch(line)
	port := &procNetPort{}
	if len(m) == 0 {
		return nil, nil
	}

	var address string
	var err error
	if len(m[1]) > 8 {
		address, err = hex2ipv6(m[1])
	} else {
		address, err = hex2ipv4(m[1])
	}
	if err != nil {
		return nil, errors.New("failed to parse port address: " + m[1])
	}
	port.Address = address

	localPort, err := strconv.ParseUint(m[2], 16, 64)
	if err != nil {
		return nil, errors.New("failed to parse port number: " + m[2])
	}
	port.Port = int64(localPort)

	var remoteAddress string
	if len(m[1]) > 8 {
		remoteAddress, err = hex2ipv6(m[3])
	} else {
		remoteAddress, err = hex2ipv4(m[3])
	}
	if err != nil {
		return nil, errors.New("failed to parse port address: " + m[3])
	}
	port.RemoteAddress = remoteAddress

	remotePort, err := strconv.ParseUint(m[4], 16, 64)
	if err != nil {
		return nil, errors.New("failed to parse port number: " + m[4])
	}
	port.RemotePort = int64(remotePort)

	stateNum, err := strconv.ParseInt(m[5], 16, 64)
	if err != nil {
		return nil, errors.New("failed to parse state number: " + m[5])
	}
	state, ok := TCP_STATES[stateNum]
	if !ok {
		state = "unknown"
	}
	port.State = state

	uid, err := strconv.ParseUint(m[6], 10, 64)
	if err != nil {
		return nil, errors.New("failed to parse port UID: " + m[6])
	}
	port.Uid = int64(uid)

	inode, err := strconv.ParseUint(m[7], 10, 64)
	if err != nil {
		return nil, errors.New("failed to parse port Inode: " + m[7])
	}
	port.Inode = int64(inode)

	return port, nil
}

func (p *mqlPorts) listLinux() ([]interface{}, error) {
	users, err := p.users()
	if err != nil {
		return nil, err
	}

	processes, processErr := p.processesBySocket()
	getProcess := func(inode int64) *resources.CacheEntry {
		found, ok := processes[inode]
		if ok {
			return &resources.CacheEntry{
				Valid:     true,
				Data:      found,
				Timestamp: time.Now().Unix(),
			}
		}

		return &resources.CacheEntry{
			Valid:     true,
			Error:     processErr,
			Timestamp: time.Now().Unix(),
		}
	}

	var ports []interface{}
	tcpPorts, err := p.parseProcNet("/proc/net/tcp", "tcp", users, getProcess)
	if err != nil {
		return nil, err
	}
	ports = append(ports, tcpPorts...)

	udpPorts, err := p.parseProcNet("/proc/net/udp", "udp", users, getProcess)
	if err != nil {
		return nil, err
	}
	ports = append(ports, udpPorts...)

	tcpPortsV6, err := p.parseProcNet("/proc/net/tcp6", "tcp", users, getProcess)
	if err != nil {
		return nil, err
	}
	ports = append(ports, tcpPortsV6...)

	udpPortsV6, err := p.parseProcNet("/proc/net/udp6", "udp", users, getProcess)
	if err != nil {
		return nil, err
	}
	ports = append(ports, udpPortsV6...)

	return ports, nil
}

func (p *mqlPorts) processesByPid() (map[int64]Process, error) {
	// Prerequisites: processes
	obj, err := p.MotorRuntime.CreateResource("processes")
	if err != nil {
		return nil, err
	}
	processes := obj.(Processes)

	_, err = processes.List()
	if err != nil {
		return nil, err
	}

	c, ok := processes.MqlResource().Cache.Load("_map")
	if !ok {
		return nil, errors.New("cannot get map of processes")
	}

	return c.Data.(map[int64]Process), nil
}

// Windows Implementation

func (p *mqlPorts) listWindows() ([]interface{}, error) {
	processes, err := p.processesByPid()
	if err != nil {
		return nil, err
	}

	osProvider, err := osProvider(p.MotorRuntime.Motor)
	if err != nil {
		return nil, err
	}
	encodedCmd := powershell.Encode("Get-NetTCPConnection | ConvertTo-Json")
	executedCmd, err := osProvider.RunCommand(encodedCmd)
	if err != nil {
		return nil, err
	}

	list, err := p.parseWindowsPorts(executedCmd.Stdout, processes)
	if err != nil {
		return nil, err
	}

	return list, nil
}

func (p *mqlPorts) parseWindowsPorts(r io.Reader, processes map[int64]Process) ([]interface{}, error) {
	portList, err := ports.ParseWindowsNetTCPConnections(r)
	if err != nil {
		return nil, err
	}

	var res []interface{}
	for i := range portList {
		port := portList[i]

		var state string
		switch port.State {
		case ports.Listen:
			state = TCP_STATES[10]
		case ports.Closed:
			state = TCP_STATES[7]
		case ports.SynSent:
			state = TCP_STATES[2]
		case ports.SynReceived:
			state = TCP_STATES[3]
		case ports.Established:
			state = TCP_STATES[1]
		case ports.FinWait1:
			state = TCP_STATES[4]
		case ports.FinWait2:
			state = TCP_STATES[5]
		case ports.CloseWait:
			state = TCP_STATES[8]
		case ports.Closing:
			state = TCP_STATES[11]
		case ports.LastAck:
			state = TCP_STATES[9]
		case ports.TimeWait:
			state = TCP_STATES[6]
		case ports.DeleteTCB:
			state = "deletetcb"
		case ports.Bound:
			state = "bound"
		}

		process := processes[port.OwningProcess]

		protocol := "ipv4"
		if strings.Contains(port.LocalAddress, ":") {
			protocol = "ipv6"
		}

		obj, err := p.MotorRuntime.CreateResource("port",
			"protocol", protocol,
			"port", port.LocalPort,
			"address", port.LocalAddress,
			"user", nil,
			"process", process,
			"state", state,
			"remoteAddress", port.RemoteAddress,
			"remotePort", port.RemotePort,
		)
		if err != nil {
			log.Error().Err(err).Send()
			return nil, err
		}

		res = append(res, obj)
	}
	return res, nil
}

// macOS Implementation

// listMacos reads the lsof information of all open files that are tcp sockets
func (p *mqlPorts) listMacos() ([]interface{}, error) {
	users, err := p.users()
	if err != nil {
		return nil, err
	}

	processes, err := p.processesByPid()
	if err != nil {
		return nil, err
	}

	osProvider, err := osProvider(p.MotorRuntime.Motor)
	if err != nil {
		return nil, err
	}

	executedCmd, err := osProvider.RunCommand("lsof -nP -i -F")
	if err != nil {
		return nil, err
	}

	lsofProcesses, err := lsof.Parse(executedCmd.Stdout)
	if err != nil {
		return nil, err
	}

	// iterating over all processes to find the once that have network file descriptors
	var res []interface{}
	for i := range lsofProcesses {
		process := lsofProcesses[i]
		for j := range process.FileDescriptors {
			fd := process.FileDescriptors[j]
			if fd.Type != lsof.FileTypeIPv4 && fd.Type != lsof.FileTypeIPv6 {
				continue
			}

			uid, err := strconv.Atoi(process.UID)
			if err != nil {
				return nil, err
			}
			user := users[int64(uid)]

			pid, err := strconv.Atoi(process.PID)
			if err != nil {
				return nil, err
			}
			mqlProcess := processes[int64(pid)]

			protocol := "ipv4"
			if fd.Type == lsof.FileTypeIPv6 {
				protocol = "ipv6"
			}

			localAddress, localPort, remoteAddress, remotePort, err := fd.NetworkFile()
			if err != nil {
				return nil, err
			}

			state, ok := TCP_STATES[fd.TcpState()]
			if !ok {
				state = "unknown"
			}

			obj, err := p.MotorRuntime.CreateResource("port",
				"protocol", protocol,
				"port", localPort,
				"address", localAddress,
				"user", user,
				"process", mqlProcess,
				"state", state,
				"remoteAddress", remoteAddress,
				"remotePort", remotePort,
			)
			if err != nil {
				log.Error().Err(err).Send()
				return nil, err
			}

			res = append(res, obj)
		}
	}

	return res, nil
}

func (s *mqlPort) id() (string, error) {
	proto, err := s.Protocol()
	if err != nil {
		return "", err
	}

	port, err := s.Port()
	if err != nil {
		return "", err
	}

	addr, err := s.Address()
	if err != nil {
		return "", err
	}

	remoteAddress, err := s.RemoteAddress()
	if err != nil {
		return "", err
	}

	remotePort, err := s.RemotePort()
	if err != nil {
		return "", err
	}

	state, err := s.State()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("port: %s/%s:%d/%s:%d/%s", proto, addr, port, remoteAddress, remotePort, state), nil
}

func (s *mqlPort) GetTls(address string, port int64, proto string) (interface{}, error) {
	if address == "" || address == "0.0.0.0" {
		address = "127.0.0.1"
	}

	socket, err := s.MotorRuntime.CreateResource("socket",
		"protocol", proto,
		"port", port,
		"address", address,
	)
	if err != nil {
		return nil, err
	}

	return s.MotorRuntime.CreateResource("tls",
		"socket", socket,
		"domainName", "",
	)
}
