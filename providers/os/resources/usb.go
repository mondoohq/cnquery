// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"bytes"
	"errors"
	"io"

	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v12/providers/os/resources/plist"
	"go.mondoo.com/cnquery/v12/providers/os/resources/usb"
)

func (d *mqlUsb) devices() ([]any, error) {
	conn := d.MqlRuntime.Connection.(shared.Connection)
	pf := conn.Asset().Platform

	switch {
	case pf.IsFamily("darwin"):
		return d.listMacos()
	default:
		return nil, errors.New("could not detect usb: " + pf.Name)
	}
}

func (d *mqlUsb) listMacos() ([]any, error) {
	conn := d.MqlRuntime.Connection.(shared.Connection)

	cmd, err := conn.RunCommand("ioreg -p IOUSB -l -w 0 -a")
	if err != nil {
		return nil, err
	}

	data, err := io.ReadAll(cmd.Stdout)
	if err != nil {
		return nil, err
	}

	plistData, err := plist.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	// Extract USB devices
	var devices []usb.USBDevice
	usb.ParseMacosIORegData(plistData, &devices)

	mqlUsbDevices := make([]any, 0, len(devices))
	for _, device := range devices {

		if device.DeviceClass == "" {
			// Skip devices without a location ID
			continue
		}

		entry, err := CreateResource(d.MqlRuntime, "usb.device", map[string]*llx.RawData{
			"__id":         llx.StringData(device.LocationID),
			"vendorId":     llx.StringData(device.VendorID),
			"manufacturer": llx.StringData(device.Manufacturer),
			"productId":    llx.StringData(device.ProductID),
			"serial":       llx.StringData(device.SerialNumber),
			"name":         llx.StringData(device.Name),
			"version":      llx.StringData(device.FormattedVersion), // BCD Version is not human readable
			"speed":        llx.StringData(device.USBSpeed),
			"class":        llx.StringData(device.DeviceClass),
			"subclass":     llx.StringData(device.DeviceSubClass),
			"className":    llx.StringData(device.DeviceClassName),
			"protocol":     llx.StringData(device.DeviceProtocol),
			"isRemovable":  llx.BoolData(device.IsRemovable),
		})
		if err != nil {
			return nil, err
		}

		mqlUsbDevices = append(mqlUsbDevices, entry)
	}

	return mqlUsbDevices, nil
}
