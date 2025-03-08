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

// CloudProviderType is the type of cloud provider that the cloud detect detected
type CloudProviderType string

var (
	UNKNOWN CloudProviderType = "UNKNOWN"
	AWS     CloudProviderType = "AWS"
	GCP     CloudProviderType = "GCP"
	AZURE   CloudProviderType = "AZURE"
)

var detectors = map[CloudProviderType]detectorFunc{
	AWS:   aws.Detect,
	AZURE: azure.Detect,
	GCP:   gcp.Detect,
}

// PlatformInfo contains platform information gathered from one of our cloud detectors.
type PlatformInfo struct {
	ID                 string
	Name               string
	Kind               string
	RelatedPlatformIDs []string
	CloudProvider      CloudProviderType
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

	valChan := make(chan PlatformInfo, len(detectors))
	for i := range detectors {
		go func(provider CloudProviderType, f detectorFunc) {
			defer wg.Done()

			id, name, related := f(conn, p, mgr)
			if id != "" {
				valChan <- PlatformInfo{
					ID:                 id,
					Name:               name,
					CloudProvider:      provider,
					RelatedPlatformIDs: related,
				}
			}
		}(i, detectors[i])
	}

	wg.Wait()
	close(valChan)

	platformIds := []string{}
	relatedPlatformIds := []string{}
	cloudProvider := UNKNOWN
	var name string
	for v := range valChan {
		platformIds = append(platformIds, v.ID)
		name = v.Name
		cloudProvider = v.CloudProvider
		relatedPlatformIds = append(relatedPlatformIds, v.RelatedPlatformIDs...)
	}

	if len(platformIds) == 0 {
		return nil
	} else if len(platformIds) > 1 {
		log.Error().Strs("detected", platformIds).Msg("multiple cloud platform ids detected")
		return nil
	}

	return &PlatformInfo{
		ID:                 platformIds[0],
		Name:               name,
		CloudProvider:      cloudProvider,
		Kind:               inventory.AssetKindCloudVM,
		RelatedPlatformIDs: relatedPlatformIds,
	}
}
