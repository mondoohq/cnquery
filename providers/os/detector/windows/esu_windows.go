// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build windows

package windows

import (
	wmi "github.com/StackExchange/wmi"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/providers/os/connection/shared"
	"golang.org/x/sys/windows/registry"
)

const esuLicenseQuery = "SELECT LicenseStatus FROM SoftwareLicensingProduct WHERE Name LIKE '%ESU%' AND LicenseStatus = 1"

func GetWindowsESUStatus(conn shared.Connection) (*WindowsESUStatus, error) {
	log.Debug().Msg("checking Windows 10 ESU status")

	// if we are running locally on windows, check registry and WMI directly
	if conn.Type() == shared.Type_Local {
		status := &WindowsESUStatus{}

		// Check subscription-based ESU via registry
		k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion\SoftwareProtectionPlatform\ESU`, registry.QUERY_VALUE)
		if err == nil {
			defer k.Close()

			eligible, _, err := k.GetIntegerValue("Win10CommercialW365ESUEligible")
			if err != nil && err != registry.ErrNotExist {
				log.Debug().Err(err).Msg("could not get Win10CommercialW365ESUEligible value")
			}
			status.SubscriptionEligible = eligible == 1
		} else {
			log.Debug().Err(err).Msg("could not open ESU registry key, subscription ESU may not be configured")
		}

		// Check MAK-activated ESU via WMI
		type softwareLicensingProduct struct {
			LicenseStatus *int
		}
		var products []softwareLicensingProduct
		if err := wmi.Query(esuLicenseQuery, &products); err != nil {
			log.Debug().Err(err).Msg("could not query WMI for ESU license status")
		} else {
			status.LicenseActivated = len(products) > 0
		}

		return status, nil
	}

	// for all non-local checks use powershell
	return powershellGetWindowsESUStatus(conn)
}
