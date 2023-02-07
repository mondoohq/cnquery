package ports

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

type State int64

const (
	Closed      = 1
	Listen      = 2
	SynSent     = 3
	SynReceived = 4
	Established = 5
	FinWait1    = 6
	FinWait2    = 7
	CloseWait   = 8
	Closing     = 9
	LastAck     = 10
	TimeWait    = 11
	DeleteTCB   = 12
	Bound       = 13
)

type WinPort struct {
	State         State
	LocalAddress  string
	LocalPort     int64
	RemoteAddress string
	RemotePort    int64
	OwningProcess int64
}

func ParseWindowsNetTCPConnections(r io.Reader) ([]WinPort, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	entries := []WinPort{}
	err = json.Unmarshal(data, &entries)
	if err != nil {
		return nil, err
	}

	// convert any ipv6 address (basically if they contain a ':')
	// to a more ipv6-friendly address surrounded by []s
	for i := range entries {
		localAddress := entries[i].LocalAddress
		if strings.ContainsAny(entries[i].LocalAddress, ":") {
			localAddress = fmt.Sprintf("[%s]", entries[i].LocalAddress)
		}
		remoteAddress := entries[i].RemoteAddress
		if strings.ContainsAny(entries[i].RemoteAddress, ":") {
			remoteAddress = fmt.Sprintf("[%s]", entries[i].RemoteAddress)
		}

		entries[i].LocalAddress = localAddress
		entries[i].RemoteAddress = remoteAddress
	}

	return entries, nil
}
