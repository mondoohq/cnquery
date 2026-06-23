// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsUnknownCommandErr(t *testing.T) {
	// RouterOS returns this trap when a menu/command isn't available on the
	// device (e.g. /interface/wifi without the wifi package).
	assert.True(t, isUnknownCommandErr(errors.New("no such command prefix")))
	assert.True(t, isUnknownCommandErr(errors.New("from RouterOS: No Such Command")))

	assert.False(t, isUnknownCommandErr(nil))
	assert.False(t, isUnknownCommandErr(errors.New("connection refused")))
	assert.False(t, isUnknownCommandErr(errors.New("permission denied")))
}
