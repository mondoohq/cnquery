package clouddetect

import (
	"sync"

	"go.mondoo.com/cnquery/motor/providers/os"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/motor/motorid/clouddetect/providers/aws"
	"go.mondoo.com/cnquery/motor/motorid/clouddetect/providers/azure"
	"go.mondoo.com/cnquery/motor/motorid/clouddetect/providers/gce"
	"go.mondoo.com/cnquery/motor/platform"
)

type (
	RelatedPlatformID = string
	PlatformID        = string
)

type detectorFunc func(provider os.OperatingSystemProvider, p *platform.Platform) (PlatformID, []RelatedPlatformID)

var detectors = []detectorFunc{
	aws.Detect,
	azure.Detect,
	gce.Detect,
}

type detectResult struct {
	platformId         string
	relatedPlatformIds []string
}

func Detect(provider os.OperatingSystemProvider, p *platform.Platform) (PlatformID, []RelatedPlatformID) {
	wg := sync.WaitGroup{}
	wg.Add(len(detectors))

	valChan := make(chan detectResult, len(detectors))
	for i := range detectors {
		go func(f detectorFunc) {
			defer wg.Done()

			v, related := f(provider, p)
			if v != "" {
				valChan <- detectResult{
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
	for v := range valChan {
		platformIds = append(platformIds, v.platformId)
		relatedPlatformIds = append(relatedPlatformIds, v.relatedPlatformIds...)
	}

	if len(platformIds) == 0 {
		return "", nil
	} else if len(platformIds) > 1 {
		log.Error().Strs("detected", platformIds).Msg("multiple cloud platform ids detected")
		return "", nil
	}

	return platformIds[0], relatedPlatformIds
}
