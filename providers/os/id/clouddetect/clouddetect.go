// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package clouddetect

import (
	"sync"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v11/providers/os/id/aws"
	"go.mondoo.com/cnquery/v11/providers/os/id/azure"
	"go.mondoo.com/cnquery/v11/providers/os/id/gcp"
	"go.mondoo.com/cnquery/v11/providers/os/resources/smbios"
)

type (
	RelatedPlatformID = string
	PlatformName      = string
	PlatformID        = string
)

type detectorFunc func(conn shared.Connection, p *inventory.Platform, smbiosMgr smbios.SmBiosManager) (PlatformID, PlatformName, []RelatedPlatformID)

var detectors = []detectorFunc{
	aws.Detect,
	azure.Detect,
	gcp.Detect,
}

type detectResult struct {
	platformId         string
	platformName       string
	relatedPlatformIds []string
}

// PlatformInfo contains platform information gathered from one of our cloud detectors.
type PlatformInfo struct {
	ID                 string
	Name               string
	Kind               string
	RelatedPlatformIDs []string
}

// Detect tried to detect if we are running on a cloud asset, and if so, it returns
// the platform information, otherwise it returns a `nil` pointer.
func Detect(conn shared.Connection, p *inventory.Platform) *PlatformInfo {
	mgr, err := smbios.ResolveManager(conn, p)
	if err != nil {
		return nil
	}

	wg := sync.WaitGroup{}
	wg.Add(len(detectors))

	valChan := make(chan detectResult, len(detectors))
	for i := range detectors {
		go func(f detectorFunc) {
			defer wg.Done()

			v, name, related := f(conn, p, mgr)
			if v != "" {
				valChan <- detectResult{
					platformName:       name,
					platformId:         v,
					relatedPlatformIds: related,
				}
			}
		}(detectors[i])
	}

	wg.Wait()
	close(valChan)

	platformIds := []string{}
	relatedPlatformIds := []string{}
	var name string
	for v := range valChan {
		platformIds = append(platformIds, v.platformId)
		name = v.platformName
		relatedPlatformIds = append(relatedPlatformIds, v.relatedPlatformIds...)
	}

	if len(platformIds) == 0 {
		return nil
	} else if len(platformIds) > 1 {
		log.Error().Strs("detected", platformIds).Msg("multiple cloud platform ids detected")
		return nil
	}

	return &PlatformInfo{platformIds[0], name, inventory.AssetKindCloudVM, relatedPlatformIds}
}
