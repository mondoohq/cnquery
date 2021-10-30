package resources

import (
	"bufio"
	"errors"
	"regexp"
	"strconv"
)

func (s *lumiPorts) id() (string, error) {
	return "ports", nil
}

func (p *lumiPorts) GetList() ([]interface{}, error) {

	pf, err := p.Runtime.Motor.Platform()
	if err != nil {
		return nil, err
	}

	switch {
	case pf.IsFamily("linux"):
		return p.listLinux()
	default:
		return nil, errors.New("could not detect suitable ports manager for platform: " + pf.Name)
	}
}

func (p *lumiPorts) GetListening() ([]interface{}, error) {
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

	return (strconv.FormatUint(a, 10) + "." +
		strconv.FormatUint(b, 10) + "." +
		strconv.FormatUint(c, 10) + "." +
		strconv.FormatUint(d, 10)), nil
}

func (p *lumiPorts) users() (map[int64]User, error) {
	obj, err := p.Runtime.CreateResource("users")
	if err != nil {
		return nil, err
	}
	users := obj.(Users)

	_, err = users.List()
	if err != nil {
		return nil, err
	}

	c, ok := users.LumiResource().Cache.Load("_map")
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

func (p *lumiPorts) processes() (map[int64]Process, error) {
	// Prerequisites: processes
	obj, err := p.Runtime.CreateResource("processes")
	if err != nil {
		return nil, err
	}
	processes := obj.(Processes)

	_, err = processes.List()
	if err != nil {
		return nil, err
	}

	c, ok := processes.LumiResource().Cache.Load("_socketsMap")
	if !ok {
		return nil, errors.New("cannot get map of processes (and their sockets)")
	}

	return c.Data.(map[int64]Process), nil
}

func (p *lumiPorts) parseProcNet(path string, protocol string, users map[int64]User, processes map[int64]Process) ([]interface{}, error) {
	motor := p.Runtime.Motor
	fs := motor.Transport.FS()
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

		obj, err := p.Runtime.CreateResource("port",
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

func (p *lumiPorts) listLinux() ([]interface{}, error) {

	users, err := p.users()
	if err != nil {
		return nil, err
	}

	processes, err := p.processes()
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

func (s *lumiPort) id() (string, error) {
	proto, err := s.Protocol()
	if err != nil {
		return "", err
	}

	port, err := s.Port()
	if err != nil {
		return "", err
	}

	return "port: " + proto + "/" + strconv.Itoa(int(port)), nil
}
