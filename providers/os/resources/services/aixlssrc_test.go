// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package services

import (
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func TestLssrcParse(t *testing.T) {
	testOutput := `
Subsystem         Group            PID          Status 
 syslogd          ras              3932558      active
 aso                               4653462      active
 biod             nfs              5046692      active
 rpc.lockd        nfs              5636560      active
 qdaemon          spooler          5767630      active
 ctrmc            rsct             5439966      active
 pmperfrec                         6881768      active
 IBM.HostRM       rsct_rm          5898530      active
 automountd       autofs           7340402      active
 lpd              spooler                       inoperative
 nimsh            nimclient                     inoperative
 nimhttp                                        inoperative
 timed            tcpip                         inoperative
`
	entries := parseLssrc(strings.NewReader(testOutput))
	assert.Equal(t, 13, len(entries), "detected the right amount of services")
	assert.Equal(t, "syslogd", entries[0].Subsystem, "service name detected")
	assert.Equal(t, "active", entries[0].Status, "service status detected")
	assert.Equal(t, "timed", entries[12].Subsystem, "service name detected")
	assert.Equal(t, "inoperative", entries[12].Status, "service status detected")
}
