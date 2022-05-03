package motorid

import (
	"fmt"

	"go.mondoo.io/mondoo/motor/motorid/sshhostkey"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/motorid/awsec2"
	"go.mondoo.io/mondoo/motor/motorid/clouddetect"
	"go.mondoo.io/mondoo/motor/motorid/hostname"
	"go.mondoo.io/mondoo/motor/motorid/machineid"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/mock"
)

type PlatformFingerprint struct {
	PlatformIDs []string
	Name        string
	// TODO: add labels detection
	// Labels      map[string]string
}

func IdentifyPlatform(t transports.Transport, p *platform.Platform, idDetectors []transports.PlatformIdDetector) (*PlatformFingerprint, error) {
	if len(idDetectors) == 0 {
		idDetectors = t.PlatformIdDetectors()
	}

	var fingerprint PlatformFingerprint
	var ids []string
	for i := range idDetectors {
		idDetector := idDetectors[i]
		platformIds, err := GatherPlatformIDs(t, p, idDetector)
		if err != nil {
			// we only err if we found zero platform ids, if we try multiple, a fail of an individual one is okay
			log.Debug().Err(err).Str("detector", string(idDetector)).Msg("could not determine platform id")
			continue
		}
		if len(platformIds) > 0 {
			ids = append(ids, platformIds...)
		}

		// check if we get a name for the asset, eg. aws instance id
		for i := range platformIds {
			name := gatherNameForPlatformId(platformIds[i])
			if name != "" {
				fingerprint.Name = name
			}
		}
	}

	// if we found zero platform ids something went wrong
	if len(ids) == 0 {
		return nil, errors.New("could not determine a platform identifier")
	}

	fingerprint.PlatformIDs = ids

	log.Debug().Interface("id-detector", idDetectors).Strs("platform-ids", ids).Msg("detected platform ids")
	return &fingerprint, nil
}

func gatherNameForPlatformId(id string) string {
	if awsec2.IsValidMondooInstanceId(id) {
		structId, _ := awsec2.ParseMondooInstanceID(id)
		return structId.Id
	}
	return ""
}

func GatherPlatformIDs(t transports.Transport, p *platform.Platform, idDetector transports.PlatformIdDetector) ([]string, error) {
	transport := t
	// helper for recoding transport to extract the original transport
	recT, ok := t.(*mock.RecordTransport)
	if ok {
		transport = recT.Watched()
	}

	var identifier string
	switch idDetector {
	case transports.HostnameDetector:
		// NOTE: we need to be careful with hostname's since they are not required to be unique
		hostname, hostErr := hostname.Hostname(t, p)
		if hostErr == nil && len(hostname) > 0 {
			identifier = "//platformid.api.mondoo.app/hostname/" + hostname
		}
		return []string{identifier}, hostErr
	case transports.MachineIdDetector:
		guid, hostErr := machineid.MachineId(t, p)
		if hostErr == nil && len(guid) > 0 {
			identifier = "//platformid.api.mondoo.app/machineid/" + guid
		}
		return []string{identifier}, hostErr
	case transports.AWSEc2Detector:
		metadata, err := awsec2.Resolve(transport, p)
		if err != nil {
			return nil, err
		}
		identifier, err := metadata.InstanceID()
		if err != nil {
			return nil, err
		}
		return []string{identifier}, nil

	case transports.CloudDetector:
		identifier := clouddetect.Detect(t, p)
		return []string{identifier}, nil
	case transports.SshHostKey:
		identifier, err := sshhostkey.Detect(t, p)
		if err != nil {
			return nil, err
		}
		return identifier, err
	case transports.TransportPlatformIdentifierDetector:
		identifiable, ok := transport.(transports.TransportPlatformIdentifier)
		if !ok {
			return nil, errors.New("the transport-platform-id detector is not supported for transport")
		}

		identifier, err := identifiable.Identifier()
		if err != nil {
			return nil, err
		}
		return []string{identifier}, nil
	default:
		return nil, errors.New(fmt.Sprintf("the provided id-detector is not supported: %s", idDetector))
	}
}
