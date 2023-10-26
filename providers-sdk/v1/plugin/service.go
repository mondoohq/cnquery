// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package plugin

import (
	"errors"
	"os"
	"time"
)

type Service struct {
	lastHeartbeat int64
}

var heartbeatRes HeartbeatRes

func (s *Service) Heartbeat(req *HeartbeatReq) (*HeartbeatRes, error) {
	if req.Interval == 0 {
		return nil, errors.New("heartbeat failed, requested interval is 0")
	}

	now := time.Now().UnixNano()
	s.lastHeartbeat = now

	go func() {
		time.Sleep(time.Duration(req.Interval))
		if s.lastHeartbeat == now {
			os.Exit(1)
		}
	}()

	return &heartbeatRes, nil
}
