package motorid

import (
	"fmt"

	"errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/motor/discovery/aws"
	"go.mondoo.com/cnquery/motor/motorid/awsec2"
	awsecsid "go.mondoo.com/cnquery/motor/motorid/awsecs"
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

type PlatformInfo struct {
	IDs                []string
	Name               string
	RelatedPlatformIDs []string
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
		platformInfo, err := GatherPlatformInfo(t, p, idDetector)
		if err != nil {
			// we only err if we found zero platform ids, if we try multiple, a fail of an individual one is okay
			log.Debug().Err(err).Str("detector", string(idDetector)).Msg("could not determine platform info")
			continue
		}
		if len(platformInfo.IDs) > 0 {
			ids = append(ids, platformInfo.IDs...)
		}
		if len(platformInfo.RelatedPlatformIDs) > 0 {
			relatedIds = append(relatedIds, platformInfo.RelatedPlatformIDs...)
		}

		if len(platformInfo.Name) > 0 {
			fingerprint.Name = platformInfo.Name
		} else {
			// check if we get a name for the asset, eg. aws instance id
			for _, id := range platformInfo.IDs {
				name := gatherNameForPlatformId(id)
				if name != "" {
					fingerprint.Name = name
				}
			}
		}
		// check whether we can extract runtime and kind information
		for _, id := range platformInfo.IDs {
			runtime, kind := extractPlatformAndKindFromPlatformId(id)
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
	} else if awsecsid.IsValidMondooECSContainerId(id) {
		return providers.RUNTIME_AWS_ECS, providers.Kind_KIND_CONTAINER
	}
	return "", providers.Kind_KIND_UNKNOWN
}

func GatherPlatformInfo(provider providers.Instance, pf *platform.Platform, idDetector providers.PlatformIdDetector) (*PlatformInfo, error) {
	// helper for recoding provider to extract the original provider
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
			return &PlatformInfo{
				IDs:                []string{identifier},
				Name:               "",
				RelatedPlatformIDs: []string{},
			}, hostErr
		}
		return &PlatformInfo{}, nil
	case isOSProvider && idDetector == providers.MachineIdDetector:
		guid, hostErr := machineid.MachineId(osProvider, pf)
		if hostErr == nil && len(guid) > 0 {
			identifier = "//platformid.api.mondoo.app/machineid/" + guid
			return &PlatformInfo{
				IDs:                []string{identifier},
				Name:               "",
				RelatedPlatformIDs: []string{},
			}, hostErr
		}
		return &PlatformInfo{}, nil
	case isOSProvider && idDetector == providers.AWSEc2Detector:
		metadata, err := awsec2.Resolve(osProvider, pf)
		if err != nil {
			return nil, err
		}
		ident, err := metadata.Identify()
		if err != nil {
			return nil, err
		}
		if ident.InstanceID != "" {
			return &PlatformInfo{
				IDs:                []string{ident.InstanceID},
				Name:               ident.InstanceName,
				RelatedPlatformIDs: []string{ident.AccountID},
			}, nil
		}
		return &PlatformInfo{}, nil
	case isOSProvider && idDetector == providers.AWSEcsDetector:
		metadata, err := awsecsid.Resolve(osProvider, pf)
		if err != nil {
			return nil, err
		}
		ident, err := metadata.Identify()
		if err != nil {
			return nil, err
		}
		if len(ident.PlatformIds) != 0 {
			return &PlatformInfo{
				IDs:                ident.PlatformIds,
				Name:               ident.Name,
				RelatedPlatformIDs: []string{ident.AccountPlatformID},
			}, nil
		}
		return &PlatformInfo{}, nil
	case isOSProvider && idDetector == providers.CloudDetector:
		identifier, name, relatedIdentifiers := clouddetect.Detect(osProvider, pf)
		if identifier != "" {
			return &PlatformInfo{
				IDs:                []string{identifier},
				Name:               name,
				RelatedPlatformIDs: relatedIdentifiers,
			}, nil
		}
		return &PlatformInfo{}, nil
	case isOSProvider && idDetector == providers.SshHostKey:
		identifier, err := sshhostkey.Detect(osProvider, pf)
		if err != nil {
			return nil, err
		}
		return &PlatformInfo{
			IDs:                identifier,
			Name:               "",
			RelatedPlatformIDs: []string{},
		}, nil
	case idDetector == providers.TransportPlatformIdentifierDetector:
		identifiable, ok := provider.(providers.PlatformIdentifier)
		if !ok {
			return nil, errors.New("the transport-platform-id detector is not supported for transport")
		}

		identifier, err := identifiable.Identifier()
		if err != nil {
			return nil, err
		}
		return &PlatformInfo{
			IDs:                []string{identifier},
			Name:               "",
			RelatedPlatformIDs: []string{},
		}, nil
	default:
		return nil, errors.New(fmt.Sprintf("the provided id-detector is not supported: %s", idDetector))
	}
}
