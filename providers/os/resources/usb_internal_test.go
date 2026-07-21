// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/mql/v13/providers/os/resources/usb"
)

func TestUsbDevicesWithLocation(t *testing.T) {
	devices := []usb.USBDevice{
		// Composite device: empty DeviceClass but a valid LocationID — must be kept.
		{Name: "composite", LocationID: "0x14100000", DeviceClass: ""},
		// Has a class but no LocationID — must be dropped (blank __id otherwise).
		{Name: "no-location", LocationID: "", DeviceClass: "9"},
		{Name: "normal", LocationID: "0x14200000", DeviceClass: "0"},
	}

	got := usbDevicesWithLocation(devices)

	names := make([]string, 0, len(got))
	for _, d := range got {
		names = append(names, d.Name)
		assert.NotEmpty(t, d.LocationID)
	}
	assert.ElementsMatch(t, []string{"composite", "normal"}, names)
}
