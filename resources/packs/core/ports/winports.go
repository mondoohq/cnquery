package ports

import (
	"encoding/json"
	"io"
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

	return entries, nil
}
