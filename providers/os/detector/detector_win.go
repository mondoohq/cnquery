// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package detector

import (
	"strconv"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers/os/connection/shared"
	win "go.mondoo.com/cnquery/v11/providers/os/detector/windows"
	"go.mondoo.com/cnquery/v11/providers/os/registry"
)

// runtimeWindowsDetector uses powershell to gather information about the windows system
func runtimeWindowsDetector(pf *inventory.Platform, conn shared.Connection) (bool, error) {
	// most systems support wmi, but windows on arm does not ship with wmic, therefore we are trying to use windows
	// builds from registry key first. If that fails, we try to use wmi
	// see https://techcommunity.microsoft.com/t5/windows-it-pro-blog/wmi-command-line-wmic-utility-deprecation-next-steps/ba-p/4039242

	if pf.Labels == nil {
		pf.Labels = map[string]string{}
	}

	//  try to get build + ubr number (win 10+, 2019+)
	current, err := win.GetWindowsOSBuild(conn)
	if err == nil && current.UBR > 0 {
		pf.Name = "windows"
		pf.Title = current.ProductName
		pf.Version = current.CurrentBuild
		pf.Build = strconv.Itoa(current.UBR)
		pf.Arch = current.Architecture

		var productType string
		switch current.ProductType {
		case "WinNT":
			productType = "1" // Workstation
		case "ServerNT":
			productType = "3" // Server
		case "LanmanNT":
			productType = "2" // Domain Controller
		}

		pf.Labels["windows.mondoo.com/product-type"] = productType
		pf.Labels["windows.mondoo.com/display-version"] = current.DisplayVersion
		return true, nil
	}

	// fallback to wmi if the registry key is not available
	data, err := win.GetWmiInformation(conn)
	if err != nil {
		log.Debug().Err(err).Msg("could not gather wmi information")
		return false, nil
	}

	pf.Name = "windows"
	pf.Title = data.Caption

	// instead of using windows major.minor.build.ubr we just use build.ubr since
	// major and minor can be derived from the build version
	pf.Version = data.BuildNumber

	// FIXME: we need to ask wmic cpu get architecture
	pf.Arch = data.OSArchitecture
	pf.Labels["windows.mondoo.com/product-type"] = data.ProductType

	return true, nil
}

func staticWindowsDetector(pf *inventory.Platform, conn shared.Connection) (bool, error) {
	rh := registry.NewRegistryHandler()
	defer func() {
		err := rh.UnloadSubkeys()
		if err != nil {
			log.Debug().Err(err).Msg("could not unload registry subkeys")
		}
	}()
	fi, err := conn.FileInfo(registry.SoftwareRegPath)
	if err != nil {
		log.Debug().Err(err).Msg("could not find SOFTWARE registry key file")
		return false, nil
	}
	err = rh.LoadSubkey(registry.Software, fi.Path)
	if err != nil {
		log.Debug().Err(err).Msg("could not load SOFTWARE registry key file")
		return false, nil
	}

	pf.Name = "windows"
	productName, err := rh.GetRegistryItemValue(registry.Software, "Microsoft\\Windows NT\\CurrentVersion", "ProductName")
	if err == nil {
		log.Debug().Str("productName", productName.Value.String).Msg("found productName")
		pf.Title = productName.Value.String
	}

	ubr, err := rh.GetRegistryItemValue(registry.Software, "Microsoft\\Windows NT\\CurrentVersion", "UBR")
	if err == nil && ubr.Value.String != "" {
		log.Debug().Str("ubr", ubr.Value.String).Msg("found ubr")
		pf.Build = ubr.Value.String
	}
	// we try both CurrentBuild and CurrentBuildNumber for the version number
	currentBuild, err := rh.GetRegistryItemValue(registry.Software, "Microsoft\\Windows NT\\CurrentVersion", "CurrentBuild")
	if err == nil && currentBuild.Value.String != "" {
		log.Debug().Str("currentBuild", currentBuild.Value.String).Msg("found currentBuild")
		pf.Version = currentBuild.Value.String
	} else {
		currentBuildNumber, err := rh.GetRegistryItemValue(registry.Software, "Microsoft\\Windows NT\\CurrentVersion", "CurrentBuildNumber")
		if err == nil && currentBuildNumber.Value.String != "" {
			log.Debug().Str("currentBuildNumber", currentBuildNumber.Value.String).Msg("found currentBuildNumber")
			pf.Version = currentBuildNumber.Value.String
		}
	}
	return true, nil
}
