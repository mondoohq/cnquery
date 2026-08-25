// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFreebsdPortState(t *testing.T) {
	// mapped onto the same vocabulary the other platforms use
	assert.Equal(t, "listen", freebsdPortState("LISTEN"))
	assert.Equal(t, "established", freebsdPortState("ESTABLISHED"))
	assert.Equal(t, "time wait", freebsdPortState("TIME_WAIT"))
	assert.Equal(t, "close wait", freebsdPortState("CLOSE_WAIT"))

	// udp rows carry no state at all
	assert.Equal(t, "", freebsdPortState(""))

	// a state FreeBSD adds later must stay visible rather than read as "no
	// state", which is what a bare map lookup would have produced
	assert.Equal(t, "SOME_FUTURE_STATE", freebsdPortState("SOME_FUTURE_STATE"))
}
