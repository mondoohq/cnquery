// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// Package ports holds the per-platform socket listings that back the `ports`
// resource. This file parses FreeBSD's sockstat, which ships in the base
// system, unlike lsof.
package ports

import (
	"bufio"
	"io"
	"strconv"
	"strings"
)

// SockstatEntry is one socket row from `sockstat -46 -s`.
type SockstatEntry struct {
	User          string
	Command       string
	Pid           int64
	Protocol      string // tcp4, tcp6, udp4, udp6
	LocalAddress  string
	LocalPort     int64
	RemoteAddress string
	RemotePort    int64
	State         string // CONN STATE column, empty for udp
}

// ParseSockstat reads the output of `sockstat -46 -s`.
//
//	USER     COMMAND    PID   FD  PROTO  LOCAL ADDRESS   FOREIGN ADDRESS  CONN STATE
//	root     sshd       1718  7   tcp4   *:22            *:*              LISTEN
//	ntpd     ntpd       1666  22  udp4   10.45.1.14:123  *:*
//
// Columns are whitespace separated and the command may be truncated, but never
// contains a space, so splitting on fields is safe. The CONN STATE column is
// absent for udp rows and on releases whose sockstat has no -s flag, so a row
// is accepted with anything from 7 fields up.
func ParseSockstat(r io.Reader) ([]SockstatEntry, error) {
	var res []SockstatEntry

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 7 {
			continue
		}
		if fields[0] == "USER" {
			// header
			continue
		}

		proto := fields[4]
		if !isInetProtocol(proto) {
			// unix domain sockets and anything else sockstat may list
			continue
		}

		pid, err := strconv.ParseInt(fields[2], 10, 64)
		if err != nil {
			// a kernel socket has no pid ("?"); keep the row, report pid 0
			pid = 0
		}

		localAddr, localPort := splitHostPort(fields[5])
		remoteAddr, remotePort := splitHostPort(fields[6])

		state := ""
		if len(fields) > 7 {
			state = strings.Join(fields[7:], " ")
		}

		res = append(res, SockstatEntry{
			User:          fields[0],
			Command:       fields[1],
			Pid:           pid,
			Protocol:      proto,
			LocalAddress:  localAddr,
			LocalPort:     localPort,
			RemoteAddress: remoteAddr,
			RemotePort:    remotePort,
			State:         state,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return res, nil
}

func isInetProtocol(p string) bool {
	switch p {
	case "tcp4", "tcp6", "udp4", "udp6":
		return true
	}
	return false
}

// splitHostPort splits sockstat's ADDRESS:PORT, where the host may be "*", a
// bare IPv4 address, or a bracketed IPv6 address that carries its own colons
// and may include a zone ("[fe80::1%lo0]:123"). A wildcard or unknown port
// yields 0.
func splitHostPort(s string) (string, int64) {
	if s == "" || s == "*:*" {
		return "", 0
	}

	idx := strings.LastIndex(s, ":")
	if idx < 0 {
		return s, 0
	}

	host := s[:idx]
	portStr := s[idx+1:]

	// strip the brackets IPv6 literals are wrapped in
	if len(host) > 1 && host[0] == '[' && host[len(host)-1] == ']' {
		host = host[1 : len(host)-1]
	}

	port, err := strconv.ParseInt(portStr, 10, 64)
	if err != nil {
		// "*" for a wildcard port, or a service name
		port = 0
	}

	return host, port
}
