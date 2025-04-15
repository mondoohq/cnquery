// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package usb

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v12/providers/os/resources/plist"
)

func TestUsbPlist(t *testing.T) {
	data, err := os.ReadFile("testdata/usb.plist.xml")
	require.NoError(t, err)

	plistData, err := plist.Decode(bytes.NewReader(data))
	require.NoError(t, err)
	assert.NotNil(t, plistData)

	// Extract USB devices
	var devices []USBDevice
	ParseMacosIORegData(plistData, &devices)
	assert.Equal(t, 4, len(devices))
}
