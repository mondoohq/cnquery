// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build windows
// +build windows

package windows

import (
	"errors"
	"runtime"
	"strconv"

	wmi "github.com/StackExchange/wmi"
	"go.mondoo.com/cnquery/v9/providers/os/connection/shared"
)

const wmiOSQuery = "SELECT Name, Caption, Manufacturer, OSArchitecture, Version, BuildNumber, Description, OSType, ProductType, SerialNumber FROM Win32_OperatingSystem"

func GetWmiInformation(conn shared.Connection) (*WmicOSInformation, error) {
	// if we are running locally on windows, we want to avoid using powershell to be faster
	if conn.Type() == shared.Type_Local && runtime.GOOS == "windows" {

		// we always get a list or entries
		type win32_OperatingSystem struct {
			Name           *string
			Caption        *string
			Manufacturer   *string
			OSArchitecture *string
			Version        *string
			BuildNumber    *string
			Description    *string
			OSType         *int
			ProductType    *int
		}

		// query wmi to retrieve information
		var entries []win32_OperatingSystem
		if err := wmi.Query(wmiOSQuery, &entries); err != nil {
			return nil, err
		}

		if len(entries) != 1 || entries[0].Version == nil {
			return nil, errors.New("could not query machine id on windows")
		}

		entry := entries[0]
		return &WmicOSInformation{
			Name:           toString(entry.Name),
			Caption:        toString(entry.Caption),
			Manufacturer:   toString(entry.Manufacturer),
			OSArchitecture: toString(entry.OSArchitecture),
			Version:        toString(entry.Version),
			BuildNumber:    toString(entry.BuildNumber),
			Description:    toString(entry.Description),
			OSType:         intToString(entry.OSType),
			ProductType:    intToString(entry.ProductType),
		}, nil
	}

	return powershellGetWmiInformation(conn)
}

func toString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func intToString(i *int) string {
	if i == nil {
		return ""
	}
	return strconv.Itoa(*i)
}
