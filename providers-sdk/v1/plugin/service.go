// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package plugin

import (
	"errors"
	"os"
	sync "sync"
	"time"
)

type Service struct {
	lastHeartbeat int64
	lock          sync.Mutex
}

var heartbeatRes HeartbeatRes

func (s *Service) Heartbeat(req *HeartbeatReq) (*HeartbeatRes, error) {
	if req.Interval == 0 {
		return nil, errors.New("heartbeat failed, requested interval is 0")
	}

	now := time.Now().UnixNano()
	s.lock.Lock()
	s.lastHeartbeat = now
	s.lock.Unlock()

	go func() {
		time.Sleep(time.Duration(req.Interval))

		s.lock.Lock()
		isDead := s.lastHeartbeat == now
		s.lock.Unlock()

		if isDead {
			os.Exit(1)
		}
	}()

	return &heartbeatRes, nil
}
