package motorid

import (
	"fmt"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/motor/discovery/aws"
	"go.mondoo.com/cnquery/motor/motorid/awsec2"
	"go.mondoo.com/cnquery/motor/motorid/clouddetect"
	"go.mondoo.com/cnquery/motor/motorid/hostname"
	"go.mondoo.com/cnquery/motor/motorid/machineid"
	"go.mondoo.com/cnquery/motor/motorid/sshhostkey"
	"go.mondoo.com/cnquery/motor/platform"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/providers/mock"
	"go.mondoo.com/cnquery/motor/providers/os"
)

type PlatformFingerprint struct {
	PlatformIDs []string
	Name        string
	Runtime     string
	Kind        providers.Kind
	// TODO: add labels detection
	// Labels      map[string]string
	RelatedAssets []PlatformFingerprint
}

func IdentifyPlatform(t providers.Instance, p *platform.Platform, idDetectors []providers.PlatformIdDetector) (*PlatformFingerprint, error) {
	if len(idDetectors) == 0 {
		idDetectors = t.PlatformIdDetectors()
	}

	var fingerprint PlatformFingerprint
	var ids []string
	var relatedIds []string

	for i := range idDetectors {
		idDetector := idDetectors[i]
		platformIds, relatedPlatformIds, err := GatherPlatformIDs(t, p, idDetector)
		if err != nil {
			// we only err if we found zero platform ids, if we try multiple, a fail of an individual one is okay
			log.Debug().Err(err).Str("detector", string(idDetector)).Msg("could not determine platform id")
			continue
		}
		if len(platformIds) > 0 {
			ids = append(ids, platformIds...)
		}
		if len(relatedPlatformIds) > 0 {
			relatedIds = append(relatedIds, relatedPlatformIds...)
		}

		// check if we get a name for the asset, eg. aws instance id
		for i := range platformIds {
			name := gatherNameForPlatformId(platformIds[i])
			if name != "" {
				fingerprint.Name = name
			}
		}

		// check whether we can extract runtime and kind information
		for i := range platformIds {
			runtime, kind := extractPlatformAndKindFromPlatformId(platformIds[i])
			if runtime != "" {
				fingerprint.Runtime = runtime
				fingerprint.Kind = kind
			}
		}
	}

	// if we found zero platform ids something went wrong
	if len(ids) == 0 {
		return nil, errors.New("could not determine a platform identifier")
	}

	fingerprint.PlatformIDs = ids
	for _, v := range relatedIds {
		fingerprint.RelatedAssets = append(fingerprint.RelatedAssets, PlatformFingerprint{
			PlatformIDs: []string{v},
			Name:        gatherNameForPlatformId(v),
		})
	}

	log.Debug().Interface("id-detector", idDetectors).Strs("platform-ids", ids).Msg("detected platform ids")
	return &fingerprint, nil
}

func gatherNameForPlatformId(id string) string {
	if awsec2.IsValidMondooInstanceId(id) {
		structId, _ := awsec2.ParseMondooInstanceID(id)
		return structId.Id
	} else if accountID, err := awsec2.ParseMondooAccountID(id); err == nil {
		return fmt.Sprintf("AWS Account %s", accountID)
	}
	return ""
}

func extractPlatformAndKindFromPlatformId(id string) (string, providers.Kind) {
	if aws.ParseEc2PlatformID(id) != nil {
		return providers.RUNTIME_AWS_EC2, providers.Kind_KIND_VIRTUAL_MACHINE
	} else if awsec2.IsValidMondooAccountId(id) {
		return providers.RUNTIME_AWS, providers.Kind_KIND_API
	}
	return "", providers.Kind_KIND_UNKNOWN
}

func GatherPlatformIDs(provider providers.Instance, pf *platform.Platform, idDetector providers.PlatformIdDetector) ([]string, []string, error) {
	// helper for recoding transport to extract the original transport
	recT, ok := provider.(*mock.MockRecordProvider)
	if ok {
		provider = recT.Watched()
	}

	osProvider, isOSProvider := provider.(os.OperatingSystemProvider)

	var identifier string
	switch {
	case isOSProvider && idDetector == providers.HostnameDetector:
		// NOTE: we need to be careful with hostname's since they are not required to be unique
		hostname, hostErr := hostname.Hostname(osProvider, pf)
		if hostErr == nil && len(hostname) > 0 {
			identifier = "//platformid.api.mondoo.app/hostname/" + hostname
		}
		return []string{identifier}, nil, hostErr
	case isOSProvider && idDetector == providers.MachineIdDetector:
		guid, hostErr := machineid.MachineId(osProvider, pf)
		if hostErr == nil && len(guid) > 0 {
			identifier = "//platformid.api.mondoo.app/machineid/" + guid
		}
		return []string{identifier}, nil, hostErr
	case isOSProvider && idDetector == providers.AWSEc2Detector:
		metadata, err := awsec2.Resolve(osProvider, pf)
		if err != nil {
			return nil, nil, err
		}
		ident, err := metadata.Identify()
		if err != nil {
			return nil, nil, err
		}
		return []string{ident.InstanceID}, []string{ident.AccountID}, nil

	case isOSProvider && idDetector == providers.CloudDetector:
		identifier, relatedIdentifiers := clouddetect.Detect(osProvider, pf)
		return []string{identifier}, relatedIdentifiers, nil
	case isOSProvider && idDetector == providers.SshHostKey:
		identifier, err := sshhostkey.Detect(osProvider, pf)
		if err != nil {
			return nil, nil, err
		}
		return identifier, nil, err
	case idDetector == providers.TransportPlatformIdentifierDetector:
		identifiable, ok := provider.(providers.PlatformIdentifier)
		if !ok {
			return nil, nil, errors.New("the transport-platform-id detector is not supported for transport")
		}

		identifier, err := identifiable.Identifier()
		if err != nil {
			return nil, nil, err
		}
		return []string{identifier}, nil, nil
	default:
		return nil, nil, errors.New(fmt.Sprintf("the provided id-detector is not supported: %s", idDetector))
	}
}
