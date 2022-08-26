//go:build windows
// +build windows

package windows

import (
	"errors"
	"runtime"
	"strconv"

	wmi "github.com/StackExchange/wmi"
	"go.mondoo.com/cnquery/motor/providers/local"
	"go.mondoo.com/cnquery/motor/providers/os"
)

const wmiOSQuery = "SELECT Name, Caption, Manufacturer, OSArchitecture, Version, BuildNumber, Description, OSType, ProductType, SerialNumber FROM Win32_OperatingSystem"

func GetWmiInformation(p os.OperatingSystemProvider) (*WmicOSInformation, error) {
	// if we are running locally on windows, we want to avoid using powershell to be faster
	_, ok := p.(*local.Provider)
	if ok && runtime.GOOS == "windows" {

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

	return powershellGetWmiInformation(p)
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
