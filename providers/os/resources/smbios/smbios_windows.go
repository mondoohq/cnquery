// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build windows
// +build windows

package smbios

import (
	"fmt"
	"runtime"
	"strconv"
	"time"

	wmi "github.com/StackExchange/wmi"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/providers/os/connection/shared"
)

// native WMI on local Windows; PowerShell otherwise or on failure
func fetchWindowsSmbios(conn shared.Connection) (smbiosWindows, error) {
	if conn.Type() == shared.Type_Local && runtime.GOOS == "windows" {
		winBios, err := nativeWindowsSmbios()
		if err == nil {
			return winBios, nil
		}
		log.Debug().Err(err).Msg("could not query smbios via WMI, falling back to PowerShell")
	}
	return fetchWindowsSmbiosPowershell(conn)
}

func nativeWindowsSmbios() (out smbiosWindows, err error) {
	// the wmi lib can panic on unexpected COM variant types; recover so we
	// fall back to PowerShell instead of crashing the scan
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic querying smbios via WMI: %v", r)
		}
	}()

	type win32Bios struct {
		Manufacturer      string
		SMBIOSBIOSVersion string
		ReleaseDate       time.Time
		SerialNumber      string
	}
	var bios []win32Bios
	if qErr := wmi.Query("SELECT Manufacturer, SMBIOSBIOSVersion, ReleaseDate, SerialNumber FROM Win32_Bios", &bios); qErr != nil {
		return out, qErr
	}
	if len(bios) > 0 {
		out.Bios = smbiosWinBios{
			Manufacturer:      bios[0].Manufacturer,
			SMBIOSBIOSVersion: bios[0].SMBIOSBIOSVersion,
			SerialNumber:      bios[0].SerialNumber,
		}
		if !bios[0].ReleaseDate.IsZero() {
			out.Bios.ReleaseDate = bios[0].ReleaseDate.Format(time.RFC3339)
		}
	}

	type win32BaseBoard struct {
		Manufacturer string
		Product      string
		Version      string
		SerialNumber string
	}
	var baseboard []win32BaseBoard
	if qErr := wmi.Query("SELECT Manufacturer, Product, Version, SerialNumber FROM Win32_BaseBoard", &baseboard); qErr != nil {
		return out, qErr
	}
	if len(baseboard) > 0 {
		out.BaseBoard = smbiosBaseBoard(baseboard[0])
	}

	type win32SystemEnclosure struct {
		Manufacturer string
		Model        *string
		// int32, not uint16: the COM SAFEARRAY returns VT_I4 elements and the
		// wmi lib panics calling reflect.Value.Uint on them.
		ChassisTypes   []int32
		Version        string
		SerialNumber   string
		SMBIOSAssetTag string
	}
	var chassis []win32SystemEnclosure
	if qErr := wmi.Query("SELECT Manufacturer, Model, ChassisTypes, Version, SerialNumber, SMBIOSAssetTag FROM Win32_SystemEnclosure", &chassis); qErr != nil {
		return out, qErr
	}
	for _, ch := range chassis {
		types := make([]string, 0, len(ch.ChassisTypes))
		for _, t := range ch.ChassisTypes {
			types = append(types, strconv.Itoa(int(t)))
		}
		out.Chassis = append(out.Chassis, smbiosChassis{
			Manufacturer:   ch.Manufacturer,
			Model:          ch.Model,
			ChassisTypes:   &smbiosChassisTypes{ChassisTypes: types},
			Version:        ch.Version,
			SerialNumber:   ch.SerialNumber,
			SMBIOSAssetTag: ch.SMBIOSAssetTag,
		})
	}

	type win32ComputerSystemProduct struct {
		Vendor            string
		Name              string
		Version           string
		SKUNumber         string
		UUID              string
		IdentifyingNumber string
	}
	var product []win32ComputerSystemProduct
	if qErr := wmi.Query("SELECT Vendor, Name, Version, SKUNumber, UUID, IdentifyingNumber FROM Win32_ComputerSystemProduct", &product); qErr != nil {
		return out, qErr
	}
	if len(product) > 0 {
		out.SystemProduct = smbiosSystemProduct(product[0])
	}

	return out, nil
}
