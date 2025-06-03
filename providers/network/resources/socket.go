// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"net"
	"strconv"
)

func (s *mqlSocket) id() (string, error) {
	return s.Protocol.Data + "://" + net.JoinHostPort(s.Address.Data, strconv.Itoa(int(s.Port.Data))), nil
}
