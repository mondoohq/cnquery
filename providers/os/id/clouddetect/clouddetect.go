// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package clouddetect

import (
	"sync"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v10/providers/os/id/aws"
	"go.mondoo.com/cnquery/v10/providers/os/id/azure"
	"go.mondoo.com/cnquery/v10/providers/os/id/gcp"
)

type (
	RelatedPlatformID = string
	PlatformName      = string
	PlatformID        = string
)

type detectorFunc func(conn shared.Connection, p *inventory.Platform) (PlatformID, PlatformName, []RelatedPlatformID)

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

func Detect(conn shared.Connection, p *inventory.Platform) (PlatformID, PlatformName, []RelatedPlatformID) {
	wg := sync.WaitGroup{}
	wg.Add(len(detectors))

	valChan := make(chan detectResult, len(detectors))
	for i := range detectors {
		go func(f detectorFunc) {
			defer wg.Done()

			v, name, related := f(conn, p)
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
		return "", "", nil
	} else if len(platformIds) > 1 {
		log.Error().Strs("detected", platformIds).Msg("multiple cloud platform ids detected")
		return "", "", nil
	}

	return platformIds[0], name, relatedPlatformIds
}
