package core

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

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
	case pf.IsFamily("darwin"):
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

	return c.Data.(map[int64]Process), nil
}

// See:
// - socket/address parsing: https://wiki.christophchamp.com/index.php?title=Unix_sockets
func (p *mqlPorts) parseProcNet(path string, protocol string, users map[int64]User, processes map[int64]Process) ([]interface{}, error) {
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

		m := reLinuxProcNet.FindStringSubmatch(line)
		if len(m) == 0 {
			continue
		}

		address, err := hex2ipv4(m[1])
		if err != nil {
			return nil, errors.New("failed to parse port address: " + m[1])
		}

		port, err := strconv.ParseUint(m[2], 16, 64)
		if err != nil {
			return nil, errors.New("failed to parse port number: " + m[2])
		}

		remoteAddress, err := hex2ipv4(m[3])
		if err != nil {
			return nil, errors.New("failed to parse port address: " + m[3])
		}

		remotePort, err := strconv.ParseUint(m[4], 16, 64)
		if err != nil {
			return nil, errors.New("failed to parse port number: " + m[4])
		}

		stateNum, err := strconv.ParseInt(m[5], 16, 64)
		if err != nil {
			return nil, errors.New("failed to parse state number: " + m[5])
		}
		state, ok := TCP_STATES[stateNum]
		if !ok {
			state = "unknown"
		}

		uid, err := strconv.ParseUint(m[6], 10, 64)
		if err != nil {
			return nil, errors.New("failed to parse port UID: " + m[6])
		}
		user := users[int64(uid)]

		inode, err := strconv.ParseUint(m[7], 10, 64)
		if err != nil {
			return nil, errors.New("failed to parse port Inode: " + m[7])
		}

		// the process may be nil, eg if the inode is 0
		process := processes[int64(inode)]

		obj, err := p.MotorRuntime.CreateResource("port",
			"protocol", protocol,
			"port", int64(port),
			"address", address,
			"user", user,
			"process", process,
			"state", state,
			"remoteAddress", remoteAddress,
			"remotePort", int64(remotePort),
		)
		if err != nil {
			return nil, err
		}

		res = append(res, obj)
	}

	return res, nil
}

func (p *mqlPorts) listLinux() ([]interface{}, error) {
	users, err := p.users()
	if err != nil {
		return nil, err
	}

	processes, err := p.processesBySocket()
	if err != nil {
		return nil, err
	}

	tcpPorts, err := p.parseProcNet("/proc/net/tcp", "tcp", users, processes)
	if err != nil {
		return nil, err
	}

	udpPorts, err := p.parseProcNet("/proc/net/udp", "udp", users, processes)
	if err != nil {
		return nil, err
	}

	return append(tcpPorts, udpPorts...), nil
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
