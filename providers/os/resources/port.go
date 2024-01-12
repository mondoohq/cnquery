// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

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
	"sync"
	"unsafe"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v10/providers/os/resources/lsof"
	"go.mondoo.com/cnquery/v10/providers/os/resources/ports"
	"go.mondoo.com/cnquery/v10/providers/os/resources/powershell"
)

type mqlPortsInternal struct {
	processes2ports plugin.TValue[map[int64]*mqlProcess]
	lock            sync.Mutex
}

func (p *mqlPorts) list() ([]interface{}, error) {
	conn := p.MqlRuntime.Connection.(shared.Connection)
	pf := conn.Asset().Platform

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

func (p *mqlPorts) listening() ([]interface{}, error) {
	all := p.GetList()
	if all.Error != nil {
		return nil, all.Error
	}

	res := []interface{}{}
	for i := range all.Data {
		cur := all.Data[i]
		port := cur.(*mqlPort)
		if port.State.Data == "listen" {
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
		// ipv6-friendly formatting with the [] brackets
		return fmt.Sprintf("[%s]", ip.String()), nil
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

func (p *mqlPorts) users() (map[int64]*mqlUser, error) {
	obj, err := CreateResource(p.MqlRuntime, "users", map[string]*llx.RawData{})
	if err != nil {
		return nil, err
	}
	users := obj.(*mqlUsers)

	err = users.refreshCache(nil)
	if err != nil {
		return nil, err
	}

	return users.usersByID, nil
}

func (p *mqlPorts) processesBySocket() (map[int64]*mqlProcess, error) {
	p.lock.Lock()
	defer p.lock.Unlock()

	if p.processes2ports.Error != nil {
		return nil, p.processes2ports.Error
	}
	if p.processes2ports.State&plugin.StateIsSet != 0 {
		return p.processes2ports.Data, nil
	}

	// Prerequisites: processes
	obj, err := CreateResource(p.MqlRuntime, "processes", map[string]*llx.RawData{})
	if err != nil {
		p.processes2ports = plugin.TValue[map[int64]*mqlProcess]{
			State: plugin.StateIsSet,
			Error: err,
		}
		return nil, err
	}
	processes := obj.(*mqlProcesses)

	err = processes.refreshCache(nil)
	if err != nil {
		p.processes2ports = plugin.TValue[map[int64]*mqlProcess]{
			State: plugin.StateIsSet,
			Error: err,
		}
		return nil, err
	}

	p.processes2ports = plugin.TValue[map[int64]*mqlProcess]{
		Data:  processes.BySocketID,
		State: plugin.StateIsSet,
	}
	return processes.BySocketID, err
}

// See:
// - socket/address parsing: https://wiki.christophchamp.com/index.php?title=Unix_sockets
func (p *mqlPorts) parseProcNet(path string, protocol string, users map[int64]*mqlUser) ([]interface{}, error) {
	conn := p.MqlRuntime.Connection.(shared.Connection)
	fs := conn.FileSystem()
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

		obj, err := CreateResource(p.MqlRuntime, "port", map[string]*llx.RawData{
			"protocol":      llx.StringData(protocol),
			"port":          llx.IntData(port.Port),
			"address":       llx.StringData(port.Address),
			"user":          llx.ResourceData(users[port.Uid], "user"),
			"state":         llx.StringData(port.State),
			"remoteAddress": llx.StringData(port.RemoteAddress),
			"remotePort":    llx.IntData(port.RemotePort),
		})
		if err != nil {
			return nil, err
		}

		po := obj.(*mqlPort)
		po.inode = port.Inode

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

	var ports []interface{}
	tcpPorts, err := p.parseProcNet("/proc/net/tcp", "tcp4", users)
	if err != nil {
		return nil, err
	}
	ports = append(ports, tcpPorts...)

	udpPorts, err := p.parseProcNet("/proc/net/udp", "udp4", users)
	if err != nil {
		return nil, err
	}
	ports = append(ports, udpPorts...)

	tcpPortsV6, err := p.parseProcNet("/proc/net/tcp6", "tcp6", users)
	if err != nil {
		return nil, err
	}
	ports = append(ports, tcpPortsV6...)

	udpPortsV6, err := p.parseProcNet("/proc/net/udp6", "udp6", users)
	if err != nil {
		return nil, err
	}
	ports = append(ports, udpPortsV6...)

	return ports, nil
}

func (p *mqlPorts) processesByPid() (map[int64]*mqlProcess, error) {
	// Prerequisites: processes
	obj, err := CreateResource(p.MqlRuntime, "processes", map[string]*llx.RawData{})
	if err != nil {
		return nil, err
	}
	processes := obj.(*mqlProcesses)

	err = processes.refreshCache(nil)
	if err != nil {
		return nil, err
	}

	return processes.ByPID, nil
}

// Windows Implementation

func (p *mqlPorts) listWindows() ([]interface{}, error) {
	processes, err := p.processesByPid()
	if err != nil {
		return nil, err
	}

	conn := p.MqlRuntime.Connection.(shared.Connection)
	encodedCmd := powershell.Encode("Get-NetTCPConnection | ConvertTo-Json")
	executedCmd, err := conn.RunCommand(encodedCmd)
	if err != nil {
		return nil, err
	}

	list, err := p.parseWindowsPorts(executedCmd.Stdout, processes)
	if err != nil {
		return nil, err
	}

	return list, nil
}

func (p *mqlPorts) parseWindowsPorts(r io.Reader, processes map[int64]*mqlProcess) ([]interface{}, error) {
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

		protocol := "tcp4"
		if strings.Contains(port.LocalAddress, ":") {
			protocol = "tcp6"
		}

		obj, err := CreateResource(p.MqlRuntime, "port", map[string]*llx.RawData{
			"protocol":      llx.StringData(protocol),
			"port":          llx.IntData(port.LocalPort),
			"address":       llx.StringData(port.LocalAddress),
			"user":          llx.ResourceData(nil, "user"),
			"process":       llx.ResourceData(process, "process"),
			"state":         llx.StringData(state),
			"remoteAddress": llx.StringData(port.RemoteAddress),
			"remotePort":    llx.IntData(port.RemotePort),
		})
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

	conn := p.MqlRuntime.Connection.(shared.Connection)
	executedCmd, err := conn.RunCommand("lsof -nP -i -F")
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

			protocol := strings.ToLower(fd.Protocol)
			if fd.Type == lsof.FileTypeIPv6 {
				protocol = protocol + "6"
			} else {
				protocol = protocol + "4"
			}

			localAddress, localPort, remoteAddress, remotePort, err := fd.NetworkFile()
			if err != nil {
				return nil, err
			}
			// lsof presents a process listening on any ipv6 address as listening on "*"
			// change this to a more ipv6-friendly formatting
			if protocol == "ipv6" && strings.HasPrefix(localAddress, "*") {
				localAddress = strings.Replace(localAddress, "*", "[::]", 1)
			}

			state, ok := TCP_STATES[fd.TcpState()]
			if !ok {
				state = "unknown"
			}

			obj, err := CreateResource(p.MqlRuntime, "port", map[string]*llx.RawData{
				"protocol":      llx.StringData(protocol),
				"port":          llx.IntData(localPort),
				"address":       llx.StringData(localAddress),
				"user":          llx.ResourceData(user, "user"),
				"process":       llx.ResourceData(mqlProcess, "process"),
				"state":         llx.StringData(state),
				"remoteAddress": llx.StringData(remoteAddress),
				"remotePort":    llx.IntData(remotePort),
			})
			if err != nil {
				log.Error().Err(err).Send()
				return nil, err
			}

			res = append(res, obj)
		}
	}

	return res, nil
}

type mqlPortInternal struct {
	inode int64
}

func (s *mqlPort) id() (string, error) {
	return fmt.Sprintf("port: %s/%s:%d/%s:%d/%s",
		s.Protocol.Data, s.Address.Data, s.Port.Data,
		s.RemoteAddress.Data, s.RemotePort.Data, s.State.Data), nil
}

func (s *mqlPort) tls(address string, port int64, proto string) (plugin.Resource, error) {
	if address == "" || address == "0.0.0.0" {
		address = "127.0.0.1"
	}

	socket, err := s.MqlRuntime.CreateSharedResource("socket", map[string]*llx.RawData{
		"protocol": llx.StringData(proto),
		"port":     llx.IntData(port),
		"address":  llx.StringData(address),
	})
	if err != nil {
		return nil, err
	}

	return s.MqlRuntime.CreateSharedResource("tls", map[string]*llx.RawData{
		"socket":     llx.ResourceData(socket, "socket"),
		"domainName": llx.StringData(""),
	})
}

func (s *mqlPort) process() (*mqlProcess, error) {
	// At this point everything except for linux should have their port identified.
	// For linux we need to scour the /proc system, which takes a long time.
	// TODO: massively speed this up on linux with more approach.
	conn := s.MqlRuntime.Connection.(shared.Connection)
	pf := conn.Asset().Platform
	if !pf.IsFamily("linux") {
		return nil, errors.New("unable to detect process for this port")
	}

	obj, err := CreateResource(s.MqlRuntime, "ports", map[string]*llx.RawData{})
	if err != nil {
		return nil, err
	}
	ports := obj.(*mqlPorts)

	// TODO: refresh on the fly, eg when loading this from a recording
	if s.inode == 0 {
		return nil, errors.New("no iNode found for this port and cannot yet refresh it")
	}

	procs, err := ports.processesBySocket()
	if err != nil {
		return nil, err
	}
	proc := procs[s.inode]
	if proc == nil {
		s.Process = plugin.TValue[*mqlProcess]{State: plugin.StateIsSet | plugin.StateIsNull}
	}
	return proc, nil
}
