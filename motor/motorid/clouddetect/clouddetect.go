package clouddetect

import (
	"sync"

	"github.com/rs/zerolog/log"

	"go.mondoo.io/mondoo/motor/motorid/clouddetect/providers/aws"
	"go.mondoo.io/mondoo/motor/motorid/clouddetect/providers/azure"
	"go.mondoo.io/mondoo/motor/motorid/clouddetect/providers/gce"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/transports"
)

type detectorFunc func(t transports.Transport, p *platform.Platform) string

var detectors = []detectorFunc{
	aws.Detect,
	azure.Detect,
	gce.Detect,
}

func Detect(t transports.Transport, p *platform.Platform) (string, map[string]string) {
	wg := sync.WaitGroup{}
	wg.Add(len(detectors))

	valChan := make(chan string, len(detectors))
	for i := range detectors {
		go func(f detectorFunc) {
			defer wg.Done()

			v := f(t, p)
			if v != "" {
				valChan <- v
			}
		}(detectors[i])
	}

	wg.Wait()
	close(valChan)

	platformIds := []string{}
	for v := range valChan {
		platformIds = append(platformIds, v)
	}

	if len(platformIds) == 0 {
		return "", nil
	} else if len(platformIds) > 1 {
		log.Error().Strs("detected", platformIds).Msg("multiple cloud platform ids detected")
		return "", nil
	}

	return platformIds[0], nil
}
